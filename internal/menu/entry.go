// Package menu provides the interactive TUI menu, entry registry, and dispatcher.
package menu

import (
	"bufio"   // for bufio.Reader -- SafeInput reads stdin through this
	"context" // for context.Context -- every handler receives a cancellable context
	"sort"    // for sort.Slice -- ascending sort by menu number in Sorted()

	"github.com/jmorrison-juniper/misthelper-go/internal/output" // for output.Writer -- passed to every handler
)

// HandlerFunc is the function signature every menu operation implements.
// ctx carries cancellation; reader provides stdin; w is the output backend.
type HandlerFunc func(ctx context.Context, reader *bufio.Reader, w output.Writer) error

// Entry is one item in the menu registry.
type Entry struct {
	Number      int         // Menu number shown to the user (e.g. 11)
	Title       string      // Display name (e.g. "List All Devices")
	Category    string      // Category header (e.g. "Data Extraction")
	Handler     HandlerFunc // Function that executes this operation
	Destructive bool        // True for operations 90-100 (require CONFIRM prompt)
}

// Registry maps menu numbers to their Entry definitions.
type Registry struct {
	entries map[int]Entry // Internal map from menu number to its Entry struct
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[int]Entry)} // Initialize with empty map -- prevents nil map panics on first Register
}

// Register adds or replaces an Entry in the registry.
func (r *Registry) Register(e Entry) {
	r.entries[e.Number] = e // Keyed by menu number so re-registering replaces without error
}

// Get returns the Entry for a given number and whether it was found.
func (r *Registry) Get(number int) (Entry, bool) {
	entry, ok := r.entries[number] // Map lookup returns zero-value Entry and false when missing
	return entry, ok               // Caller decides what to do with a missing entry
}

// Sorted returns all entries sorted ascending by Entry.Number.
func (r *Registry) Sorted() []Entry {
	result := make([]Entry, 0, len(r.entries)) // Pre-allocate to the exact capacity needed
	for _, e := range r.entries {              // Iterate map -- order is non-deterministic, hence the sort
		result = append(result, e) // Collect each entry into the sortable slice
	}
	sort.Slice(result, func(i, j int) bool { // Sort ascending by Number for predictable menu display
		return result[i].Number < result[j].Number // Lower numbers appear first in the menu
	})
	return result // Return the sorted slice -- caller owns the copy
}
