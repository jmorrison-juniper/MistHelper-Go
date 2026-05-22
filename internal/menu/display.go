// Package menu provides the interactive TUI menu, entry registry, and dispatcher.
package menu

import (
	"fmt"     // for fmt.Fprintf and fmt.Fprintln -- write formatted output to any io.Writer
	"io"      // for io.Writer -- PrintMenu accepts any writer so tests can capture output
	"log/slog" // for slog.Info / slog.Debug -- structured logging before and after output
	"os"      // for os.Stdout -- default output destination when w is nil
	"sort"    // for sort.Strings -- deterministic alphabetical category ordering
	"strings" // for strings.Repeat -- build separator and padding strings
)

// contentWidth is the number of printable characters between the box borders.
// Total box width = contentWidth + 4 (two pipes + two spaces).
const contentWidth = 36 // 36 chars keeps lines under 42 chars total -- matches terminal width

// PrintMenu prints the full menu to w (defaults to os.Stdout if w is nil).
// Entries are grouped by Category, then sorted by Number within each group.
func PrintMenu(w io.Writer, r *Registry) {
	if w == nil {                            // Guard: default to stdout when no override is provided
		w = os.Stdout                        // Write to the terminal for interactive sessions
	}
	groups := groupByCategory(r.Sorted())   // Partition the sorted entries by their Category field
	categories := sortedCategories(groups)  // Get category names in deterministic alphabetical order
	slog.Info("printing menu", "categories", len(categories)) // Log before the output loop
	for _, category := range categories {   // Iterate categories in sorted order for consistent display
		printCategory(w, category, groups[category]) // Print one category section per iteration
	}
	slog.Debug("menu printed", "total_entries", len(r.entries)) // Log total after all output
}

// groupByCategory partitions a sorted slice of entries into a map keyed by Category.
func groupByCategory(entries []Entry) map[string][]Entry {
	groups := make(map[string][]Entry) // Allocate the map before iterating
	for _, e := range entries {        // Entries are already sorted by Number from Sorted()
		groups[e.Category] = append(groups[e.Category], e) // Append preserves within-category order
	}
	return groups // Return the populated map to PrintMenu
}

// sortedCategories returns the category keys in ascending alphabetical order.
func sortedCategories(groups map[string][]Entry) []string {
	names := make([]string, 0, len(groups)) // Pre-allocate slice to the exact number of categories
	for k := range groups {                 // Map iteration order is random -- collect then sort
		names = append(names, k) // Gather each category name for sorting
	}
	sort.Strings(names) // Sort alphabetically so the menu order is stable across runs
	return names        // Return sorted list to PrintMenu
}

// printCategory writes the ASCII box section for one category.
func printCategory(w io.Writer, category string, entries []Entry) {
	sep := "+" + strings.Repeat("-", contentWidth+2) + "+" // Build separator: +---...---+
	_, _ = fmt.Fprintln(w, " "+category)                           // Print category header above the box; Fprintln error discarded (write failure unrecoverable in TUI)
	_, _ = fmt.Fprintln(w, sep)                                    // Print top border of the box; Fprintln error discarded
	for _, e := range entries {                             // One row per entry in this category
		printEntry(w, e) // Delegate single-row rendering to keep this function small
	}
	_, _ = fmt.Fprintln(w, sep) // Print bottom border to close the box; Fprintln error discarded
	_, _ = fmt.Fprintln(w, "")  // Blank line separates consecutive category sections; Fprintln error discarded
}

// printEntry writes a single formatted entry row inside the box borders.
func printEntry(w io.Writer, e Entry) {
	label := fmt.Sprintf("[%d] %s", e.Number, e.Title) // Compose label from number and display title
	if len(label) > contentWidth {                      // Truncate oversized labels to prevent box overflow
		label = label[:contentWidth]                    // Clip to the maximum printable width
	}
	padding := strings.Repeat(" ", contentWidth-len(label)) // Right-pad label so all rows are the same width
	_, _ = fmt.Fprintf(w, "| %s%s |\n", label, padding) // Write padded row between box borders; Fprintf error discarded (TUI write failure is unrecoverable)
}
