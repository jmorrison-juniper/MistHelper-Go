// Package api provides the Mist API client wrapper for MistHelper-Go.
// It wraps the mistapi-go SDK with retry logic, rate limiting, and structured logging.
package api

import (
	"context"       // context.Context for cancellation and timeouts
	"encoding/json" // JSON marshal/unmarshal for Site struct conversion
	"fmt"           // fmt.Errorf for error wrapping with context
	"log/slog"      // slog for structured logging (Go 1.21+ standard library)
	"time"          // time.Sleep for rate limiting between API pages

	"github.com/google/uuid"                       // uuid.Parse for OrgID string → UUID
	"github.com/tmunzer/mistapi-go/mistapi"        // SDK client and config types
	"github.com/tmunzer/mistapi-go/mistapi/models" // models.Site for site struct conversion
)

// defaultPageLimit is the maximum number of sites returned per API page request.
// 1000 matches the Python MistHelper DEFAULT_API_PAGE_LIMIT constant.
const defaultPageLimit = 1000

// Client wraps the mistapi-go SDK with project-level config, retry, and logging.
type Client struct {
	sdk         mistapi.ClientInterface                                                                // unexported SDK handle; ClientInterface is returned by NewClient
	cfg         Config                                                                                 // config by value; avoids pointer aliasing issues
	pageFetcher func(ctx context.Context, orgID uuid.UUID, limit int, page int) ([]models.Site, error) // production default wraps the SDK; tests can replace it to avoid real API calls
	inventoryFetcher func(ctx context.Context, orgID uuid.UUID, limit int, page int, includeVC bool) ([]models.Inventory, error) // inventory page fetcher; tests can replace it for deterministic behavior
}

// NewClient constructs a Client from a Config, initialising the mistapi-go SDK.
// Returns an error if APIToken is empty — all other validation happens in LoadConfig.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIToken == "" { // guard: SDK does not validate token at construction time
		return nil, fmt.Errorf("NewClient: APIToken is required -- set MIST_API_TOKEN in .env")
	}
	slog.Info("Initialising Mist API client", "org_id", cfg.OrgID) // log before SDK construction

	authHeader := "Token " + cfg.APIToken                 // Mist API requires "Token {apitoken}" format in the Authorization header
	creds := mistapi.NewApiTokenCredentials(authHeader)   // build API token credentials with required "Token " prefix
	conf := mistapi.CreateConfiguration(                  // build SDK configuration using functional options
		mistapi.WithApiTokenCredentials(creds), // apply API token auth
	)
	sdk := mistapi.NewClient(conf) // construct the SDK client; no network call at this point

	client := &Client{sdk: sdk, cfg: cfg}                                 // Build the client before wiring the page fetcher closure
	client.pageFetcher = client.fetchPageFromSDK                          // Default to the SDK-backed fetcher in production
	client.inventoryFetcher = client.fetchInventoryPageFromSDK            // Default inventory fetcher to SDK-backed implementation
	slog.Debug("Mist API client ready", "rate_limit_ms", cfg.RateLimitMs) // log after successful construction
	return client, nil                                                    // return Client with embedded SDK and config
}

// fetchPageFromSDK fetches a single page of org sites directly from the SDK.
// It is split out so tests can swap in a stub pageFetcher while still exercising
// the retry logic inside fetchSitePage.
func (c *Client) fetchPageFromSDK(ctx context.Context, orgID uuid.UUID, limit int, page int) ([]models.Site, error) {
	resp, callErr := c.sdk.OrgsSites().ListOrgSites(ctx, orgID, &limit, &page) // one SDK call per attempt
	if callErr != nil {                                                        // SDK failures are treated as retryable by the caller
		return nil, callErr // Return the raw SDK error so fetchSitePage can wrap it
	}
	return resp.Data, nil // Return the page of sites so fetchSitePage can accumulate it
}

// fetchInventoryPageFromSDK fetches one organisation inventory page from the Mist SDK.
// includeVC controls whether virtual-chassis physical members are included (Python parity requires true).
func (c *Client) fetchInventoryPageFromSDK(ctx context.Context, orgID uuid.UUID, limit int, page int, includeVC bool) ([]models.Inventory, error) {
	resp, callErr := c.sdk.OrgsInventory().GetOrgInventory( // Call org inventory endpoint with parity flags and pagination values
		ctx,         // Request context for cancellation and deadlines
		orgID,       // Target organisation UUID
		nil,         // serial filter disabled for full inventory export
		nil,         // model filter disabled for full inventory export
		nil,         // device type filter disabled for full inventory export
		nil,         // mac filter disabled for full inventory export
		nil,         // site filter disabled for full inventory export
		nil,         // vc_mac filter disabled for full inventory export
		&includeVC,  // include VC member devices to match Python MistHelper behavior
		nil,         // unassigned filter disabled for full inventory export
		nil,         // modified-after filter disabled for full inventory export
		&limit,      // page size limit for deterministic pagination behavior
		&page,       // page number for iterative retrieval
	)
	if callErr != nil { // SDK failure should be retried by fetchInventoryPage wrapper
		return nil, callErr // Return raw SDK error for retry classification upstream
	}
	return resp.Data, nil // Return inventory records for this page
}

// fetchSitePage fetches a single page of org sites with retry on transient errors.
// The caller is responsible for sleeping between pages and detecting the last page.
func (c *Client) fetchSitePage(ctx context.Context, orgID uuid.UUID, page int) ([]models.Site, error) {
	limit := defaultPageLimit // page size constant; captured by the closure below
	var sites []models.Site   // result variable populated inside the retry closure

	err := withRetry(ctx, func() error { // withRetry handles exponential backoff on transient failures
		pageData, callErr := c.pageFetcher(ctx, orgID, limit, page) // pageFetcher is injected in tests and backed by the SDK in production
		if callErr != nil {                                         // SDK or test stub failures are treated as retryable here
			return RetryableError(callErr) // mark as retryable so withRetry will back off and try again
		}
		sites = pageData // copy results out of the closure on success
		return nil
	}, DefaultRetryConfig)

	if err != nil { // non-retryable or max attempts exceeded
		return nil, fmt.Errorf("fetchSitePage page %d: %w", page, err) // wrap with page number for debuggability
	}
	return sites, nil // return the fetched site slice
}

// fetchInventoryPage fetches a single organisation inventory page with retry on transient failures.
func (c *Client) fetchInventoryPage(ctx context.Context, orgID uuid.UUID, page int, includeVC bool) ([]models.Inventory, error) {
	limit := defaultPageLimit // Use shared page-size constant for parity with other paginated operations
	var inventory []models.Inventory // Capture page data from retry closure on success

	err := withRetry(ctx, func() error { // Retry transient SDK errors with exponential backoff
		pageData, callErr := c.inventoryFetcher(ctx, orgID, limit, page, includeVC) // Fetch one inventory page via injected fetcher
		if callErr != nil { // Any fetch error should be treated as retryable in this layer
			return RetryableError(callErr) // Mark error as retryable for withRetry policy
		}
		inventory = pageData // Persist successful page data outside closure for return
		return nil // Signal success for this attempt
	}, DefaultRetryConfig)

	if err != nil { // Exhausted retries or permanent failure
		return nil, fmt.Errorf("fetchInventoryPage page %d: %w", page, err) // Add page context for diagnostics
	}
	return inventory, nil // Return inventory page data to pagination loop
}

// sitesToMaps converts a slice of Site structs to generic string-keyed maps via JSON round-trip.
// The round-trip preserves all JSON-tagged fields and normalises Optional[T] wrappers to plain values.
func sitesToMaps(sites []models.Site) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(sites)) // pre-allocate capacity to avoid repeated reallocations
	for _, site := range sites {                  // iterate over each site in the slice
		b, marshalErr := json.Marshal(site) // serialise struct to JSON bytes using json tags
		if marshalErr != nil {
			return nil, fmt.Errorf("sitesToMaps: marshal: %w", marshalErr) // should not occur for valid SDK structs
		}
		var row map[string]any // target map for deserialisation
		if unmarshalErr := json.Unmarshal(b, &row); unmarshalErr != nil {
			return nil, fmt.Errorf("sitesToMaps: unmarshal: %w", unmarshalErr) // should not occur for valid JSON
		}
		rows = append(rows, row) // add converted map to results
	}
	return rows, nil // return all converted rows
}

// inventoryToMaps converts SDK inventory structs into generic maps via JSON round-trip.
// This normalization keeps writer behavior consistent across different endpoint record types.
func inventoryToMaps(inventory []models.Inventory) ([]map[string]any, error) {
	rows := make([]map[string]any, 0, len(inventory)) // Pre-allocate result capacity for all inventory rows
	for _, item := range inventory {                  // Iterate each inventory model from SDK response
		b, marshalErr := json.Marshal(item) // Serialize typed model into JSON using API field tags
		if marshalErr != nil {              // Serialization error should never happen for valid SDK models
			return nil, fmt.Errorf("inventoryToMaps: marshal: %w", marshalErr) // Wrap with conversion context
		}
		var row map[string]any // Destination generic map for output writer compatibility
		if unmarshalErr := json.Unmarshal(b, &row); unmarshalErr != nil { // Convert JSON object into generic map
			return nil, fmt.Errorf("inventoryToMaps: unmarshal: %w", unmarshalErr) // Wrap with conversion context
		}
		rows = append(rows, row) // Append normalized row to output slice
	}
	return rows, nil // Return fully normalized inventory rows
}

// ListSites retrieves all sites for the configured org, paginating until exhausted.
// A rate-limit sleep is applied after each page to respect the Mist API throttle limits.
func (c *Client) ListSites(ctx context.Context) ([]map[string]any, error) {
	slog.Info("Listing org sites", "org_id", c.cfg.OrgID) // log action before first API call

	orgID, err := uuid.Parse(c.cfg.OrgID) // parse OrgID string to uuid.UUID required by SDK
	if err != nil {
		return nil, fmt.Errorf("ListSites: invalid org ID %q: %w", c.cfg.OrgID, err) // surface parse error with value
	}

	var all []map[string]any  // accumulator for all pages
	for page := 1; ; page++ { // paginate from page 1 until we detect the last page
		sites, fetchErr := c.fetchSitePage(ctx, orgID, page) // fetch one page with retry
		if fetchErr != nil {
			return nil, fmt.Errorf("ListSites: %w", fetchErr) // propagate with context
		}

		rows, convertErr := sitesToMaps(sites) // convert []models.Site to []map[string]any
		if convertErr != nil {
			return nil, fmt.Errorf("ListSites: %w", convertErr) // propagate with context
		}
		all = append(all, rows...) // append this page's rows to the accumulator

		time.Sleep(time.Duration(c.cfg.RateLimitMs) * time.Millisecond) // honour configured rate limit between pages

		if len(sites) < defaultPageLimit { // fewer results than limit means this was the last page
			break
		}
	}

	slog.Debug("Listed org sites", "count", len(all)) // log result count after all pages fetched
	return all, nil                                   // return complete flattened site list
}

// GetOrgInventory retrieves full organisation inventory with vc member inclusion and pagination.
// Endpoint parity target: Python MistHelper getOrgInventory(vc=true, limit=1000) behavior.
func (c *Client) GetOrgInventory(ctx context.Context) ([]map[string]any, error) {
	slog.Info("Listing org inventory", "org_id", c.cfg.OrgID) // Log operation start for traceability

	orgID, err := uuid.Parse(c.cfg.OrgID) // Parse configured org UUID required by SDK endpoints
	if err != nil {                       // Invalid org IDs must fail before making API calls
		return nil, fmt.Errorf("GetOrgInventory: invalid org ID %q: %w", c.cfg.OrgID, err) // Return descriptive parse failure
	}

	includeVCMembers := true // Preserve Python parity by including VC physical members in inventory response
	var all []map[string]any // Accumulate all normalized rows across pages
	for page := 1; ; page++ { // Continue paging until partial page signals completion
		inventory, fetchErr := c.fetchInventoryPage(ctx, orgID, page, includeVCMembers) // Fetch one inventory page with retry behavior
		if fetchErr != nil { // Abort on retrieval error and return deterministic failure
			return nil, fmt.Errorf("GetOrgInventory: %w", fetchErr) // Wrap with operation-level context
		}

		rows, convertErr := inventoryToMaps(inventory) // Normalize SDK structs for output writer compatibility
		if convertErr != nil {                         // Conversion failures are fatal for this operation
			return nil, fmt.Errorf("GetOrgInventory: %w", convertErr) // Wrap with operation-level conversion context
		}
		all = append(all, rows...) // Append current page rows to overall result set

		time.Sleep(time.Duration(c.cfg.RateLimitMs) * time.Millisecond) // Respect configured fixed delay between paginated calls

		if len(inventory) < defaultPageLimit { // Partial page means no more records to request
			break // Exit loop after final page
		}
	}

	slog.Debug("Listed org inventory", "count", len(all)) // Log final row count after pagination completes
	return all, nil                                        // Return full normalized inventory for writer routing
}
