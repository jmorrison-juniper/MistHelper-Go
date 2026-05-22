package output

import (
	"encoding/json" // for json.Marshal to encode slice values into TEXT columns
)

// FlattenRecord converts a nested map[string]any into a flat map[string]any
// where nested keys are joined with the "_" separator -- identical to Python's flatten_dict().
// Example: {"a": {"b": 1}} → {"a_b": 1}
// Slices are JSON-encoded into a string so they fit in a single CSV cell or SQLite TEXT column.
// Nil and empty maps return an empty result map rather than panicking.
func FlattenRecord(record map[string]any) map[string]any {
	result := make(map[string]any) // Allocate output map for accumulated flattened key-value pairs
	if record == nil {             // Guard against nil input -- API responses can occasionally be nil
		return result // Return empty map rather than panicking on nil range
	}
	flattenInto(result, record, "") // Recursively flatten starting at root with no key prefix
	return result                   // Return the fully flattened map to the caller
}

// flattenInto recursively walks src, accumulating flattened entries into dst.
// prefix is the dot-free key path built so far from ancestor keys joined by "_".
func flattenInto(dst map[string]any, src map[string]any, prefix string) {
	for key, value := range src { // Iterate every key at the current nesting level
		fullKey := buildKey(prefix, key) // Compute the fully-qualified flat key for this entry
		switch typed := value.(type) {  // Dispatch on value type to recurse or encode
		case map[string]any:
			flattenInto(dst, typed, fullKey) // Nested map: recurse deeper with the current key as prefix
		case []any:
			encoded, _ := json.Marshal(typed)  // Slice: JSON-encode so the whole list fits in one TEXT cell
			dst[fullKey] = string(encoded)     // Store the JSON string -- callers can decode if needed
		default:
			dst[fullKey] = value // Scalar (string, int, float, bool, nil): store the value as-is
		}
	}
}

// buildKey joins a parent prefix and the current key with an underscore separator.
// When prefix is empty (i.e. we are at the root level), the key is returned unchanged
// to avoid a leading underscore on top-level fields.
func buildKey(prefix, key string) string {
	if prefix == "" { // Root level: no parent path to prepend
		return key // Return bare key without any separator
	}
	return prefix + "_" + key // Non-root: join parent path and current key with underscore
}
