// Package api -- pagination tests covering fetchSitePage and ListSites.
// Uses a test hook (pageFetcher) injected into Client to avoid real Mist API calls
// while still exercising the retry loop inside fetchSitePage.
package api

import (
	"context" // for context.WithCancel and context.Background in all tests
	"errors"  // for errors.New -- constructs sentinel errors for error-path tests
	"testing" // for testing.T -- standard test runner

	"github.com/google/uuid"                       // for uuid.UUID in testPageFn signature
	"github.com/tmunzer/mistapi-go/mistapi/models" // for models.Site -- return type of testPageFn
)

// ── testClient helper ─────────────────────────────────────────────────────────

// testClientWithFn builds a Client that uses the supplied function instead of the real SDK.
// The Client is configured with a zero RateLimitMs so tests complete without sleeping.
func testClientWithFn(fn func(ctx context.Context, orgID uuid.UUID, limit int, page int) ([]models.Site, error)) *Client {
	return &Client{ // Construct Client with only test-relevant fields set
		cfg: Config{
			OrgID:       "00000000-0000-0000-0000-000000000000", // Valid UUID so uuid.Parse succeeds
			RateLimitMs: 0,                                      // Zero rate limit avoids sleeps in tests
		},
		pageFetcher: fn, // Inject mock page-fetcher; keeps the retry loop active in tests
	}
}

// ── ListSites tests ───────────────────────────────────────────────────────────

// TestListSites_InvalidOrgID verifies that a malformed OrgID in Config returns a parse error.
// This exercises the uuid.Parse failure branch in ListSites without any API call.
func TestListSites_InvalidOrgID(t *testing.T) {
	t.Parallel()                                              // Independent of all other tests
	client := &Client{cfg: Config{OrgID: "not-a-valid-uuid"}} // OrgID that uuid.Parse must reject
	_, err := client.ListSites(context.Background())          // Must fail at the UUID parse step
	if err == nil {                                           // Nil error means the invalid UUID was accepted (wrong)
		t.Error("ListSites returned nil error for invalid OrgID; want parse error") // Report missing validation
	}
}

// TestListSites_EmptyResult verifies that ListSites returns an empty slice (not nil) when
// the first page contains zero sites.
func TestListSites_EmptyResult(t *testing.T) {
	t.Parallel() // Independent of all other tests
	client := testClientWithFn(func(_ context.Context, _ uuid.UUID, _ int, _ int) ([]models.Site, error) {
		return []models.Site{}, nil // Return empty page -- pagination must stop immediately
	})
	result, err := client.ListSites(context.Background()) // Must succeed and return empty slice
	if err != nil {                                       // No error expected for empty org
		t.Fatalf("ListSites returned unexpected error: %v", err) // Bail with error detail
	}
	if len(result) != 0 { // Result must be empty, matching the empty page
		t.Errorf("ListSites returned %d rows; want 0", len(result)) // Report unexpected rows
	}
}

// TestListSites_SinglePage verifies that ListSites stops paginating when the first page
// returns fewer sites than defaultPageLimit.
func TestListSites_SinglePage(t *testing.T) {
	t.Parallel()   // Independent of all other tests
	callCount := 0 // Tracks how many pages were fetched
	client := testClientWithFn(func(_ context.Context, _ uuid.UUID, _ int, page int) ([]models.Site, error) {
		callCount++    // Count this page fetch
		if page == 1 { // Only page 1 should be fetched
			return make([]models.Site, 5), nil // Return 5 sites -- less than limit, so pagination stops
		}
		return nil, errors.New("should not reach page 2") // Page 2 must never be requested
	})
	result, err := client.ListSites(context.Background()) // Must succeed with all 5 sites
	if err != nil {                                       // No error expected for valid single-page response
		t.Fatalf("ListSites returned error: %v", err) // Bail with error detail
	}
	if len(result) != 5 { // Must return exactly the 5 sites from page 1
		t.Errorf("ListSites returned %d rows; want 5", len(result)) // Report wrong count
	}
	if callCount != 1 { // Must only call the page function once
		t.Errorf("testPageFn called %d times; want 1", callCount) // Report extra page fetches
	}
}

// TestListSites_MultiPage verifies that ListSites fetches additional pages when the current
// page is full (len == defaultPageLimit) and stops when a partial page is returned.
func TestListSites_MultiPage(t *testing.T) {
	t.Parallel()   // Independent of all other tests
	callCount := 0 // Tracks how many pages were fetched
	client := testClientWithFn(func(_ context.Context, _ uuid.UUID, _ int, page int) ([]models.Site, error) {
		callCount++ // Count this page fetch
		switch page {
		case 1:
			return make([]models.Site, defaultPageLimit), nil // Full page -- pagination continues to page 2
		case 2:
			return make([]models.Site, 3), nil // Partial page -- pagination stops after this page
		default:
			return nil, errors.New("should not reach page 3+") // Pages beyond 2 must not be requested
		}
	})
	result, err := client.ListSites(context.Background()) // Must succeed with all rows from both pages
	if err != nil {                                       // No error expected for valid multi-page response
		t.Fatalf("ListSites returned error: %v", err) // Bail with error detail
	}
	want := defaultPageLimit + 3 // Page 1 full + 3 from page 2
	if len(result) != want {     // Must accumulate rows from all pages
		t.Errorf("ListSites returned %d rows; want %d", len(result), want) // Report wrong accumulation
	}
	if callCount != 2 { // Must fetch exactly 2 pages
		t.Errorf("testPageFn called %d times; want 2", callCount) // Report wrong page count
	}
}

// TestListSites_PageError verifies that ListSites propagates errors returned by fetchSitePage.
func TestListSites_PageError(t *testing.T) {
	t.Parallel()                                // Independent of all other tests
	want := errors.New("transient API failure") // Sentinel error for comparison
	client := testClientWithFn(func(_ context.Context, _ uuid.UUID, _ int, _ int) ([]models.Site, error) {
		return nil, want // Simulate a page-fetch failure (e.g., network error)
	})
	_, err := client.ListSites(context.Background()) // Must fail and propagate the error
	if err == nil {                                  // Nil error means the failure was silently swallowed (wrong)
		t.Fatal("ListSites returned nil error for a failing page; want error") // Bail -- error was not propagated
	}
	if !errors.Is(err, want) { // Must preserve the original error in the chain
		t.Errorf("ListSites error chain missing sentinel: %v", err) // Report broken error chain
	}
}

// TestListSites_ContextCancelled verifies that ListSites propagates context cancellation
// from the page fetcher, exercising the error-return path in ListSites.
func TestListSites_ContextCancelled(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context for this test
	cancel()                                                // Pre-cancel to ensure immediate failure
	client := testClientWithFn(func(callCtx context.Context, _ uuid.UUID, _ int, _ int) ([]models.Site, error) {
		return nil, callCtx.Err() // Return context error so ListSites treats it as a page failure
	})
	_, err := client.ListSites(ctx) // Must fail because context is already cancelled
	if err == nil {                 // Nil error means cancellation was ignored (wrong)
		t.Error("ListSites returned nil error for cancelled context; want error")
	}
}

// ── fetchSitePage tests ───────────────────────────────────────────────────────

// TestFetchSitePage_TestHookReturnsData verifies the pageFetcher path:
// when a pageFetcher is set, fetchSitePage returns its result through the retry wrapper.
func TestFetchSitePage_TestHookReturnsData(t *testing.T) {
	t.Parallel()                                                   // Independent of all other tests
	orgID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000") // Valid UUID for the call
	client := &Client{                                             // Construct Client with test hook set
		cfg: Config{OrgID: orgID.String()}, // Config carries the OrgID string
		pageFetcher: func(_ context.Context, _ uuid.UUID, _ int, _ int) ([]models.Site, error) {
			return make([]models.Site, 7), nil // Return 7 fake sites to verify they pass through
		},
	}
	sites, err := client.fetchSitePage(context.Background(), orgID, 1) // Must delegate to pageFetcher
	if err != nil {                                                    // No error expected from the test hook
		t.Fatalf("fetchSitePage returned unexpected error: %v", err) // Bail with error detail
	}
	if len(sites) != 7 { // Must return exactly the 7 sites from the hook
		t.Errorf("fetchSitePage returned %d sites; want 7", len(sites)) // Report wrong count
	}
}

// TestFetchSitePage_CancelledContext verifies that fetchSitePage (without pageFetcher)
// returns an error when the context is cancelled before withRetry can execute the closure.
// This covers the outer withRetry + error-return path without a real SDK client.
func TestFetchSitePage_CancelledContext(t *testing.T) {
	t.Parallel()                                                   // Independent of all other tests
	orgID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000") // Valid UUID for the call
	client := &Client{                                             // No pageFetcher set -- uses real withRetry path
		cfg:         Config{OrgID: orgID.String()}, // Minimal config (no real SDK)
		pageFetcher: nil,                           // Explicitly nil so the real withRetry path runs
	}
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context
	cancel()                                                // Pre-cancel so withRetry returns immediately (no SDK call)
	_, err := client.fetchSitePage(ctx, orgID, 1)           // withRetry must detect cancelled ctx before calling op()
	if err == nil {                                         // Nil error means cancelled context was ignored (wrong)
		t.Error("fetchSitePage returned nil error for cancelled context; want error") // Report missing cancellation check
	}
}

// TestFetchSitePage_TestHookError verifies that an error from pageFetcher is propagated
// through fetchSitePage to the caller unchanged.
func TestFetchSitePage_TestHookError(t *testing.T) {
	t.Parallel()                                                   // Independent of all other tests
	orgID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000") // Valid UUID for the call
	want := errors.New("hook error")                               // Sentinel error for comparison
	client := &Client{                                             // Client with error-returning test hook
		pageFetcher: func(_ context.Context, _ uuid.UUID, _ int, _ int) ([]models.Site, error) {
			return nil, want // Simulate a failure in the page-fetcher (e.g., SDK error)
		},
	}
	_, err := client.fetchSitePage(context.Background(), orgID, 1) // Must propagate the hook error
	if !errors.Is(err, want) {                                     // Must preserve the original error
		t.Errorf("fetchSitePage error chain missing sentinel: %v", err) // Report broken error chain
	}
}
