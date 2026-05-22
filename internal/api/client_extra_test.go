// Package api -- additional unit tests for sitesToMaps not covered by client_test.go.
package api

import (
	"testing" // for testing.T -- standard Go test runner

	"github.com/tmunzer/mistapi-go/mistapi/models" // for models.Site -- the type under test
)

// TestSitesToMaps_EmptySlice verifies that a nil and an empty slice both produce an empty result.
// Callers must receive a non-nil slice so they can range over it safely.
func TestSitesToMaps_EmptySlice(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	result, err := sitesToMaps(nil) // Nil input should not panic and should return empty slice
	if err != nil {                 // No error expected for nil input
		t.Fatalf("unexpected error for nil input: %v", err) // Report the unexpected error
	}
	if result == nil { // Must return non-nil so callers can range over it
		t.Error("expected non-nil slice for nil input, got nil") // Report nil slice as a bug
	}
	if len(result) != 0 { // Nil input produces zero rows
		t.Errorf("expected 0 rows for nil input, got %d", len(result)) // Report unexpected rows
	}

	result2, err := sitesToMaps([]models.Site{}) // Empty slice should also produce empty result
	if err != nil {                              // No error expected for empty slice
		t.Fatalf("unexpected error for empty slice: %v", err) // Report the unexpected error
	}
	if len(result2) != 0 { // Zero sites in, zero maps out
		t.Errorf("expected 0 rows for empty slice, got %d", len(result2)) // Report unexpected rows
	}
}

// TestSitesToMaps_SingleSite verifies that one Site struct produces exactly one map.
// The conversion must not drop or duplicate entries.
func TestSitesToMaps_SingleSite(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	sites := []models.Site{
		{}, // Minimal zero-value Site -- JSON round-trip must succeed without panicking
	}
	result, err := sitesToMaps(sites) // Convert one site to one map
	if err != nil {                   // The round-trip must not error for a valid zero-value struct
		t.Fatalf("unexpected error for single site: %v", err) // Report the error with context
	}
	if len(result) != 1 { // One site in must produce exactly one map out
		t.Errorf("expected 1 row, got %d", len(result)) // Report count mismatch
	}
	if result[0] == nil { // The resulting map must not be nil
		t.Error("expected non-nil map for site, got nil") // Report nil map as a bug
	}
}

// TestSitesToMaps_MultiSite verifies that multiple Site structs produce the same number of maps.
func TestSitesToMaps_MultiSite(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	sites := []models.Site{{}, {}, {}} // Three zero-value sites
	result, err := sitesToMaps(sites)  // Convert three sites to three maps
	if err != nil {                    // The round-trip must not error for valid zero-value structs
		t.Fatalf("unexpected error for multi-site: %v", err) // Report the error with context
	}
	if len(result) != 3 { // Three sites in must produce exactly three maps out
		t.Errorf("expected 3 rows, got %d", len(result)) // Report count mismatch
	}
}
