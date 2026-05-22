// Package menu provides the interactive TUI menu, entry registry, and dispatcher.
package menu

import (
	"bufio"    // for bufio.Reader -- stdin source for SafeInput calls
	"context"  // for context.Context -- propagated to every handler for cancellation
	"fmt"      // for fmt.Fprintf and fmt.Errorf -- user messages and error wrapping
	"log/slog" // for slog.Info / slog.Debug / slog.Error -- structured logging
	"os"       // for os.Stdout -- display prompt and unknown-option messages
	"strconv"  // for strconv.Atoi -- parse text menu choice to integer

	"github.com/jmorrison-juniper/misthelper-go/internal/output" // for output.Writer -- passed to handlers
)

// Dispatcher reads menu choices from stdin and executes the matching handler.
type Dispatcher struct {
	registry *Registry    // All registered menu entries
	reader   *bufio.Reader // Buffered stdin -- shared across SafeInput calls
	writer   output.Writer // Output backend (CSV/SQLite) for handler results
}

// NewDispatcher creates a Dispatcher using the provided registry, stdin reader, and output writer.
func NewDispatcher(r *Registry, reader *bufio.Reader, w output.Writer) *Dispatcher {
	return &Dispatcher{registry: r, reader: reader, writer: w} // Wire up the three required dependencies
}

// Run starts the interactive menu loop, reading choices from stdin until EOF or context cancellation.
func (d *Dispatcher) Run(ctx context.Context) error {
	for { // Loop indefinitely -- exits on EOF or context cancellation
		select { // Poll context before blocking on user input
		case <-ctx.Done(): // Context cancelled (e.g. SIGINT or test timeout)
			slog.Info("menu: context cancelled, stopping dispatcher") // Log graceful shutdown
			return ctx.Err()                                          // Propagate cancellation to the caller
		default: // Context still active -- proceed to show menu and read input
		}
		PrintMenu(os.Stdout, d.registry)                                  // Render the full menu to the terminal
		choice, err := SafeInput(d.reader, "Enter option: ", "dispatcher") // Block until the user types a number
		if err != nil {                                                    // io.EOF means the session has ended
			return nil // Return nil for EOF -- clean termination, not an application error
		}
		if err := d.dispatchChoice(ctx, choice); err != nil { // Parse and run the selected option
			slog.Error("menu: handler error", "error", err) // Log errors but keep the loop running
		}
	}
}

// dispatchChoice parses a string choice and runs the matching handler.
func (d *Dispatcher) dispatchChoice(ctx context.Context, choice string) error {
	n, err := strconv.Atoi(choice) // Convert the user's text input to an integer menu number
	if err != nil {                // Non-numeric input is a user mistake, not an application failure
		_, _ = fmt.Fprintf(os.Stdout, "Unknown option: %s -- type a number from the menu\n", choice) // Guide the user; Fprintf error discarded (stdout write failure is unrecoverable here)
		return nil // Non-numeric input is not a fatal error -- keep looping
	}
	entry, ok := d.registry.Get(n) // Look up the entry corresponding to this number
	if !ok {                        // Number not in registry means the user typed something out of range
		_, _ = fmt.Fprintf(os.Stdout, "Unknown option: %d -- type a number from the menu\n", n) // Guide the user; Fprintf error discarded (stdout write failure is unrecoverable here)
		return nil // Unknown option is not a fatal error -- keep looping
	}
	return d.execute(ctx, entry) // Route to execute for the confirmation gate and handler invocation
}

// execute runs a single handler after applying the destructive confirmation gate.
func (d *Dispatcher) execute(ctx context.Context, entry Entry) error {
	if entry.Destructive { // Destructive operations (90-100) require explicit "CONFIRM" before running
		confirmed, err := d.confirmDestructive(entry) // Gate: prompt user for confirmation text
		if err != nil || !confirmed {                  // EOF or wrong text cancels the operation silently
			slog.Info("menu: destructive operation cancelled", "option", entry.Number) // Log so operators can audit
			return nil // Cancellation is not an error -- user chose not to proceed
		}
	}
	slog.Info("menu: dispatching", "option", entry.Number, "handler", entry.Title)  // Log before invoking handler
	err := entry.Handler(ctx, d.reader, d.writer)                                   // Invoke the registered handler
	slog.Debug("menu: handler returned", "option", entry.Number, "error", err)      // Log result after handler returns
	return err // Propagate any handler error to the caller
}

// confirmDestructive prompts for an exact "CONFIRM" string and returns true if the user typed it.
func (d *Dispatcher) confirmDestructive(entry Entry) (bool, error) {
	_, _ = fmt.Fprintf(os.Stdout, "WARNING: '%s' is a destructive operation.\n", entry.Title) // Warn before prompting; Fprintf error discarded (stdout write failure is unrecoverable here)
	slog.Info("menu: prompting for destructive confirmation", "option", entry.Number)   // Log before the blocking read
	text, err := SafeInput(d.reader, "Type 'CONFIRM' to proceed: ", "destructive-confirm") // Read the confirmation
	if err != nil {                                                                         // EOF means session ended mid-prompt
		return false, err // Propagate EOF so execute() can cancel cleanly
	}
	confirmed := text == "CONFIRM"                                                             // Exact case-sensitive match required
	slog.Debug("menu: destructive confirmation result", "option", entry.Number, "confirmed", confirmed) // Log the outcome
	return confirmed, nil // Return the boolean to execute()
}

// Dispatch executes a single menu entry by number without showing the full menu.
// Used for --menu N direct invocation. Returns error if entry not found.
func (d *Dispatcher) Dispatch(ctx context.Context, number int) error {
	entry, ok := d.registry.Get(number) // Look up the entry by its menu number
	if !ok {                             // Missing entry means the caller passed an invalid --menu value
		return fmt.Errorf("dispatch: unknown menu option %d", number) // Return error so main() can report it
	}
	return d.execute(ctx, entry) // Run through the same gate as interactive dispatch
}
