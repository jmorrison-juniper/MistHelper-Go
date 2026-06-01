// Package output -- unit tests for the Strategies map and PKType constants.
package output

import (
	"testing" // for testing.T -- standard Go test runner
)

// TestPKType_Constants verifies that the three PKType constants have the correct string values.
// These values are used in switch statements throughout the writer; a typo would silently break upserts.
func TestPKType_Constants(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	if PKTypeNatural != "natural_pk" { // Must match the Python reference implementation exactly
		t.Errorf("PKTypeNatural = %q, want %q", PKTypeNatural, "natural_pk") // Report the mismatch
	}
	if PKTypeComposite != "composite_pk" { // Must match the Python reference implementation exactly
		t.Errorf("PKTypeComposite = %q, want %q", PKTypeComposite, "composite_pk") // Report the mismatch
	}
	if PKTypeAutoIncrement != "auto_increment_with_unique" { // Must match the Python reference implementation exactly
		t.Errorf("PKTypeAutoIncrement = %q, want %q", PKTypeAutoIncrement, "auto_increment_with_unique") // Report the mismatch
	}
}

// TestStrategies_KnownNaturalPK verifies that "listOrgSites" has the expected natural-PK strategy.
// listOrgSites is the most commonly used operation; its strategy must be correct for upserts to work.
func TestStrategies_KnownNaturalPK(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	strategy, ok := Strategies["listOrgSites"] // Look up a well-known natural-PK endpoint
	if !ok {                                   // The endpoint must be registered in the map
		t.Fatal("Strategies missing key \"listOrgSites\"") // Fatal -- downstream tests depend on this entry
	}
	if strategy.Type != PKTypeNatural { // Must be a natural PK strategy
		t.Errorf("listOrgSites Type = %q, want %q", strategy.Type, PKTypeNatural) // Report the mismatch
	}
	if len(strategy.PrimaryKey) == 0 { // Must have at least one primary key column
		t.Error("listOrgSites PrimaryKey is empty") // Empty PK means the upsert has no dedup key
	}
	if strategy.PrimaryKey[0] != "id" { // The UUID "id" column is the expected primary key
		t.Errorf("listOrgSites PrimaryKey[0] = %q, want %q", strategy.PrimaryKey[0], "id") // Report the mismatch
	}
}

// TestStrategies_GetOrgInventoryNaturalPK verifies getOrgInventory uses natural PK strategy.
// Option 26 writes must upsert by stable inventory device ID for deduplication parity.
func TestStrategies_GetOrgInventoryNaturalPK(t *testing.T) {
	t.Parallel() // Safe to run concurrently with other map-lookup tests

	strategy, ok := Strategies["getOrgInventory"] // Look up strategy used by menu option 26 export path
	if !ok {                                       // Missing key would break endpoint strategy routing
		t.Fatal("Strategies missing key \"getOrgInventory\"") // Fail fast when strategy is not registered
	}
	if strategy.Type != PKTypeNatural { // Inventory rows should dedupe by stable UUID identity
		t.Errorf("getOrgInventory Type = %q, want %q", strategy.Type, PKTypeNatural) // Report mismatched PK type
	}
	if len(strategy.PrimaryKey) == 0 { // Natural strategy must define at least one PK column
		t.Fatal("getOrgInventory PrimaryKey is empty") // Fail because writer cannot dedupe without PK column
	}
	if strategy.PrimaryKey[0] != "id" { // Mist inventory primary identifier is device id
		t.Errorf("getOrgInventory PrimaryKey[0] = %q, want %q", strategy.PrimaryKey[0], "id") // Report wrong PK column
	}
}

// TestStrategies_KnownCompositePK verifies that "searchOrgDeviceEvents" has a composite PK.
// Composite PKs are required for time-series data so re-imports don't create duplicates.
func TestStrategies_KnownCompositePK(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	strategy, ok := Strategies["searchOrgDeviceEvents"] // Look up a well-known composite-PK endpoint
	if !ok {                                            // The endpoint must be registered in the map
		t.Fatal("Strategies missing key \"searchOrgDeviceEvents\"") // Fatal -- needed for composite-PK test
	}
	if strategy.Type != PKTypeComposite { // Must be a composite PK strategy
		t.Errorf("searchOrgDeviceEvents Type = %q, want %q", strategy.Type, PKTypeComposite) // Report mismatch
	}
	if len(strategy.PrimaryKey) < 2 { // Composite PK needs at least two columns for uniqueness
		t.Errorf("searchOrgDeviceEvents PrimaryKey has %d column(s), want >= 2", len(strategy.PrimaryKey)) // Report count
	}
}

// TestStrategies_KnownAutoIncrement verifies that "getOrgLicensesSummary" has auto-increment strategy.
// Summary endpoints have no stable UUID so they fall back to rowid for deduplication.
func TestStrategies_KnownAutoIncrement(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	strategy, ok := Strategies["getOrgLicensesSummary"] // Look up a well-known auto-increment endpoint
	if !ok {                                            // The endpoint must be registered in the map
		t.Fatal("Strategies missing key \"getOrgLicensesSummary\"") // Fatal -- needed for auto-increment test
	}
	if strategy.Type != PKTypeAutoIncrement { // Must be an auto-increment strategy
		t.Errorf("getOrgLicensesSummary Type = %q, want %q", strategy.Type, PKTypeAutoIncrement) // Report mismatch
	}
}

// TestStrategies_UnknownEndpoint verifies that looking up an unknown key returns ok=false.
// The writer must handle unknown endpoints gracefully rather than panicking.
func TestStrategies_UnknownEndpoint(t *testing.T) {
	t.Parallel()                                        // Safe to run concurrently
	_, ok := Strategies["nonExistentEndpointXYZ12345"] // Look up a key that is definitely not registered
	if ok {                                            // Must return false for unknown endpoints
		t.Error("expected ok=false for unknown endpoint, got true") // Report false positive
	}
}

// TestStrategies_AllEntriesHaveType verifies that every registered strategy has a non-empty Type.
// A missing Type would cause the writer's switch statement to fall through silently.
func TestStrategies_AllEntriesHaveType(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	for name, strategy := range Strategies { // Iterate every registered endpoint
		if strategy.Type == "" { // Type must never be empty -- it drives the upsert logic
			t.Errorf("strategy %q has empty Type", name) // Report the endpoint with the missing type
		}
	}
}

// TestStrategies_AllEntriesHavePrimaryKey verifies that every strategy has at least one PK column.
// An empty PrimaryKey slice would produce a SQL statement with no WHERE clause.
func TestStrategies_AllEntriesHavePrimaryKey(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	for name, strategy := range Strategies { // Iterate every registered endpoint
		if strategy.Type == PKTypeAutoIncrement { // Auto-increment entries use internal rowid, not an API key
			continue // Skip auto-increment strategies -- they legitimately use internal PKs
		}
		if len(strategy.PrimaryKey) == 0 { // All other strategies require at least one PK column
			t.Errorf("strategy %q (type=%s) has no primary key columns", name, strategy.Type) // Report the missing PK
		}
	}
}
