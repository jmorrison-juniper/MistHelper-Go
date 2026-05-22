package output

import (
	"testing" // for *testing.T and test registration
)

// TestFlattenRecord_Simple verifies that a flat map passes through unchanged.
// A map with no nested values should be identical after flattening.
func TestFlattenRecord_Simple(t *testing.T) {
	input := map[string]any{     // Simple flat map with no nesting
		"id":   "site-1",        // String value -- should pass through as-is
		"name": "Site One",      // Another string value -- should pass through as-is
	}
	result := FlattenRecord(input) // Flatten the simple map

	if result["id"] != "site-1" { // Verify string value is preserved unchanged
		t.Errorf("expected id=site-1, got %v", result["id"]) // Report mismatch with actual value
	}
	if result["name"] != "Site One" { // Verify second string value is preserved unchanged
		t.Errorf("expected name=Site One, got %v", result["name"]) // Report mismatch with actual value
	}
	if len(result) != 2 { // No extra keys should be added by flattening a flat map
		t.Errorf("expected 2 keys, got %d", len(result)) // Report unexpected key count
	}
}

// TestFlattenRecord_Nested verifies that one level of nesting is flattened with "_" separator.
// {"location": {"x": 1}} should become {"location_x": 1}.
func TestFlattenRecord_Nested(t *testing.T) {
	input := map[string]any{                   // Map with one nested level
		"location": map[string]any{"x": 1.0}, // Nested map should be flattened with underscore
	}
	result := FlattenRecord(input) // Flatten the nested map

	if result["location_x"] != 1.0 { // Verify nested key was joined with underscore separator
		t.Errorf("expected location_x=1.0, got %v", result["location_x"]) // Report mismatch
	}
	if len(result) != 1 { // Parent key should be replaced by the flattened child key
		t.Errorf("expected 1 key, got %d: %v", len(result), result) // Report unexpected key count
	}
}

// TestFlattenRecord_DeepNested verifies that two levels of nesting are both flattened.
// {"a": {"b": {"c": 42}}} should become {"a_b_c": 42}.
func TestFlattenRecord_DeepNested(t *testing.T) {
	input := map[string]any{                               // Map with two levels of nesting
		"a": map[string]any{                               // First nesting level
			"b": map[string]any{"c": 42}, // Second nesting level -- deepest value
		},
	}
	result := FlattenRecord(input) // Flatten two levels deep

	if result["a_b_c"] != 42 { // Verify both levels were joined with underscore separators
		t.Errorf("expected a_b_c=42, got %v", result["a_b_c"]) // Report mismatch with actual
	}
	if len(result) != 1 { // Both parent keys should be consumed by flattening
		t.Errorf("expected 1 key, got %d: %v", len(result), result) // Report unexpected key count
	}
}

// TestFlattenRecord_SliceBecomesJSON verifies that a slice value is JSON-encoded to a string.
// Slices must be stored as a single TEXT value so they fit in a CSV cell or SQLite column.
func TestFlattenRecord_SliceBecomesJSON(t *testing.T) {
	input := map[string]any{             // Map containing a slice value
		"tags": []any{"wifi", "guest"}, // Slice should be serialised to JSON string
	}
	result := FlattenRecord(input) // Flatten the map with a slice value

	encoded, ok := result["tags"].(string) // Verify the slice was encoded to a string type
	if !ok {                               // Must be a string, not the original []any
		t.Fatalf("expected tags to be string, got %T", result["tags"]) // Fail with type info
	}
	if encoded != `["wifi","guest"]` { // Verify the JSON encoding matches expected format
		t.Errorf("expected [\"wifi\",\"guest\"], got %s", encoded) // Report encoding mismatch
	}
}

// TestFlattenRecord_EmptyInput verifies that nil and empty maps return an empty result map.
// Callers must not receive a nil map -- an empty map[string]any is always safe to range over.
func TestFlattenRecord_EmptyInput(t *testing.T) {
	nilResult := FlattenRecord(nil)               // Nil input should not panic
	if nilResult == nil {                         // Result must never be nil -- callers range over it
		t.Error("expected non-nil result for nil input") // Report nil result as a bug
	}
	if len(nilResult) != 0 { // Nil input should produce an empty (not populated) map
		t.Errorf("expected 0 keys for nil input, got %d", len(nilResult)) // Report unexpected entries
	}

	emptyResult := FlattenRecord(map[string]any{}) // Empty map should also produce an empty result
	if len(emptyResult) != 0 {                     // No keys in means no keys out
		t.Errorf("expected 0 keys for empty input, got %d", len(emptyResult)) // Report unexpected entries
	}
}
