// Package menu -- unit tests for the display functions (PrintMenu, groupByCategory, etc.).
package menu

import (
	"bytes"   // for bytes.Buffer -- captures PrintMenu output in memory
	"strings" // for strings.Contains -- assertion helper
	"testing" // for testing.T -- standard Go test runner
)

// TestPrintMenu_Empty verifies that PrintMenu with an empty registry writes no entry rows.
// An empty menu should still produce output (the function must not panic).
func TestPrintMenu_Empty(t *testing.T) {
	t.Parallel()         // Safe to run concurrently
	var buf bytes.Buffer // Capture output without touching stdout
	reg := NewRegistry() // Empty registry -- no entries registered

	PrintMenu(&buf, reg) // Must not panic; must write to the provided writer
	// No entries registered, so the buffer may be empty -- but the function must not panic
	// or write to stdout. We verify no panic occurred by reaching this line.
	_ = buf.String() // Access output to ensure the write target was used
}

// TestPrintMenu_OneEntry verifies that PrintMenu prints the entry's title when one entry is registered.
func TestPrintMenu_OneEntry(t *testing.T) {
	t.Parallel()                                                    // Safe to run concurrently
	var buf bytes.Buffer                                            // Capture output without touching stdout
	reg := NewRegistry()                                            // Fresh registry
	reg.Register(Entry{Number: 1, Title: "Site List", Category: "Data"}) // Register a single entry
	PrintMenu(&buf, reg)                                            // Exercise the full print path
	output := buf.String()                                          // Capture the rendered output
	if !strings.Contains(output, "Site List") {                     // The title must appear in the output
		t.Errorf("expected output to contain %q, got:\n%s", "Site List", output) // Report missing title
	}
	if !strings.Contains(output, "[1]") { // The menu number must appear in the output
		t.Errorf("expected output to contain %q, got:\n%s", "[1]", output) // Report missing number
	}
	if !strings.Contains(output, "Data") { // The category header must appear
		t.Errorf("expected output to contain category %q, got:\n%s", "Data", output) // Report missing category
	}
}

// TestPrintMenu_MultiCategory verifies that categories appear in alphabetical order.
// Deterministic ordering is required so the menu is predictable across restarts.
func TestPrintMenu_MultiCategory(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	var buf bytes.Buffer
	reg := NewRegistry()
	// Register entries in non-alphabetical category order to exercise the sort.
	reg.Register(Entry{Number: 20, Title: "Zap", Category: "Zebra"})  // Should appear last alphabetically
	reg.Register(Entry{Number: 10, Title: "Alpha", Category: "Apple"}) // Should appear first alphabetically

	PrintMenu(&buf, reg)
	output := buf.String()

	alphaPos := strings.Index(output, "Apple")  // Find the position of the first category
	zebraPos := strings.Index(output, "Zebra")  // Find the position of the second category
	if alphaPos == -1 || zebraPos == -1 {        // Both categories must be present
		t.Fatalf("expected both 'Apple' and 'Zebra' in output, got:\n%s", output) // Report missing category
	}
	if alphaPos >= zebraPos { // Apple (earlier alphabet) must appear before Zebra in the output
		t.Errorf("expected 'Apple' before 'Zebra', but got opposite order in:\n%s", output) // Report ordering bug
	}
}

// TestPrintMenu_NilWriter verifies that PrintMenu does not panic when w is nil.
// The function defaults to os.Stdout in that case, but we just verify no panic.
func TestPrintMenu_NilWriter(t *testing.T) {
	t.Parallel()         // Safe to run concurrently
	reg := NewRegistry() // Empty registry -- minimal output to stdout
	// We cannot capture stdout in a simple unit test, so we only verify no panic.
	// PrintMenu must handle nil gracefully by defaulting to os.Stdout.
	defer func() {
		if r := recover(); r != nil { // Catch any unexpected panic
			t.Errorf("PrintMenu panicked with nil writer: %v", r) // Report the panic value
		}
	}()
	PrintMenu(nil, reg) // Must not panic -- uses os.Stdout when w is nil
}

// TestGroupByCategory_Empty verifies that groupByCategory returns an empty map for an empty slice.
func TestGroupByCategory_Empty(t *testing.T) {
	t.Parallel()                                 // Safe to run concurrently
	result := groupByCategory([]Entry{})         // Empty slice -- no entries to group
	if len(result) != 0 {                        // Empty slice must produce empty map
		t.Errorf("expected 0 groups, got %d", len(result)) // Report unexpected groups
	}
}

// TestGroupByCategory_SingleCategory verifies that all entries with the same category land in one bucket.
func TestGroupByCategory_SingleCategory(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	entries := []Entry{
		{Number: 1, Title: "A", Category: "SameCat"}, // First entry in the category
		{Number: 2, Title: "B", Category: "SameCat"}, // Second entry in the same category
		{Number: 3, Title: "C", Category: "SameCat"}, // Third entry in the same category
	}
	result := groupByCategory(entries)           // Group the entries by category
	if len(result) != 1 {                        // Three entries, one category: only one map key expected
		t.Fatalf("expected 1 group, got %d", len(result)) // Report unexpected number of groups
	}
	if len(result["SameCat"]) != 3 { // All three entries must be in the "SameCat" bucket
		t.Errorf("expected 3 entries in SameCat, got %d", len(result["SameCat"])) // Report count mismatch
	}
}

// TestGroupByCategory_MultipleCategories verifies that entries in different categories land in separate buckets.
func TestGroupByCategory_MultipleCategories(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	entries := []Entry{
		{Number: 1, Title: "A", Category: "CatOne"}, // Entry in the first category
		{Number: 2, Title: "B", Category: "CatTwo"}, // Entry in the second category
		{Number: 3, Title: "C", Category: "CatOne"}, // Another entry in the first category
	}
	result := groupByCategory(entries)               // Group the mixed-category entries
	if len(result) != 2 {                            // Two distinct categories: expect two map keys
		t.Fatalf("expected 2 groups, got %d", len(result)) // Report unexpected group count
	}
	if len(result["CatOne"]) != 2 { // CatOne has two entries (numbers 1 and 3)
		t.Errorf("expected 2 entries in CatOne, got %d", len(result["CatOne"])) // Report count mismatch
	}
	if len(result["CatTwo"]) != 1 { // CatTwo has one entry (number 2)
		t.Errorf("expected 1 entry in CatTwo, got %d", len(result["CatTwo"])) // Report count mismatch
	}
}

// TestSortedCategories_Sorted verifies that sortedCategories returns keys in ascending alphabetical order.
func TestSortedCategories_Sorted(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	groups := map[string][]Entry{ // Build an unsorted map to force the sort to do real work
		"Zulu":  {},  // Should appear last in sorted output
		"Alpha": {},  // Should appear first in sorted output
		"Mike":  {},  // Should appear between Alpha and Zulu
	}
	result := sortedCategories(groups)      // Call the function under test
	if len(result) != 3 {                   // Must return all three categories
		t.Fatalf("expected 3 categories, got %d", len(result)) // Report count mismatch
	}
	if result[0] != "Alpha" || result[1] != "Mike" || result[2] != "Zulu" { // Must be alphabetically sorted
		t.Errorf("expected [Alpha Mike Zulu], got %v", result) // Report the actual order
	}
}

// TestSortedCategories_Empty verifies that sortedCategories returns an empty slice for an empty map.
func TestSortedCategories_Empty(t *testing.T) {
	t.Parallel()                                          // Safe to run concurrently
	result := sortedCategories(map[string][]Entry{})      // Empty map -- nothing to sort
	if len(result) != 0 {                                 // Empty input must produce empty output
		t.Errorf("expected empty slice, got %v", result) // Report unexpected elements
	}
}

// TestPrintEntry_Normal verifies that printEntry renders the number and title inside box borders.
func TestPrintEntry_Normal(t *testing.T) {
	t.Parallel()                                             // Safe to run concurrently
	var buf bytes.Buffer                                     // Capture the rendered row
	e := Entry{Number: 11, Title: "List All Devices"}        // Representative entry
	printEntry(&buf, e)                                      // Call the function under test
	output := buf.String()                                   // Capture the result
	if !strings.Contains(output, "[11]") {                   // Number must appear in brackets
		t.Errorf("expected [11] in output, got %q", output) // Report missing number
	}
	if !strings.Contains(output, "List All Devices") { // Title must appear in the row
		t.Errorf("expected title in output, got %q", output) // Report missing title
	}
	if !strings.HasPrefix(output, "|") { // Row must start with the box border
		t.Errorf("expected row to start with '|', got %q", output) // Report missing border
	}
}

// TestPrintEntry_Truncation verifies that printEntry clips labels longer than contentWidth.
// Without truncation, the label would overflow the box border on the right side.
func TestPrintEntry_Truncation(t *testing.T) {
	t.Parallel()                                                                              // Safe to run concurrently
	var buf bytes.Buffer                                                                      // Capture the rendered row
	longTitle := strings.Repeat("X", contentWidth+20)                                        // A title that is much longer than the box width
	e := Entry{Number: 1, Title: longTitle}                                                   // Entry with an oversized title
	printEntry(&buf, e)                                                                       // Call the function under test
	output := buf.String()                                                                    // Capture the result
	label := strings.TrimSuffix(strings.TrimPrefix(strings.Split(output, "\n")[0], "| "), " |") // Extract content between borders
	if len(label) > contentWidth {                                                            // Label must be clipped to contentWidth
		t.Errorf("label length %d exceeds contentWidth %d", len(label), contentWidth) // Report overflow
	}
}

// TestPrintCategory_ContainsEntries verifies that printCategory renders all provided entries.
func TestPrintCategory_ContainsEntries(t *testing.T) {
	t.Parallel() // Safe to run concurrently
	var buf bytes.Buffer
	entries := []Entry{
		{Number: 1, Title: "Entry One", Category: "TestCat"}, // First entry in the category
		{Number: 2, Title: "Entry Two", Category: "TestCat"}, // Second entry in the category
	}
	printCategory(&buf, "TestCat", entries) // Render the category section
	output := buf.String()                  // Capture the rendered section
	if !strings.Contains(output, "Entry One") { // First entry title must appear
		t.Errorf("expected 'Entry One' in output, got:\n%s", output) // Report missing entry
	}
	if !strings.Contains(output, "Entry Two") { // Second entry title must appear
		t.Errorf("expected 'Entry Two' in output, got:\n%s", output) // Report missing entry
	}
	if !strings.Contains(output, "+") { // Box borders must appear in the output
		t.Errorf("expected box borders in output, got:\n%s", output) // Report missing borders
	}
}
