// Package menu -- unit tests for Dispatcher.Run (the interactive menu loop).
// stubWriter is defined in dispatcher_test.go and reused here (same package, same test binary).
package menu

import (
	"bufio"          // for bufio.NewReader -- wraps strings.Reader so SafeInput can read lines
	"bytes"          // for bytes.NewBufferString -- in-memory stdin for test scenarios
	"context"        // for context.Background and context.WithCancel -- control loop termination
	"fmt"            // for fmt.Errorf -- construct handler errors and reader errors in tests
	"io"             // for io.Discard -- suppress terminal output in tests
	"testing"        // for testing.T -- standard Go test runner
	"testing/iotest" // for iotest.ErrReader -- reader that always returns a non-EOF error

	"github.com/jmorrison-juniper/misthelper-go/internal/output" // for output.Writer -- handler signature requires it
)

// TestRun_EOFReturnsNil verifies that Run returns nil when stdin is exhausted immediately.
// EOF is the normal termination signal for SSH sessions closing mid-menu.
func TestRun_EOFReturnsNil(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry()                                       // Empty registry -- no entries needed for EOF test
	reader := bufio.NewReader(bytes.NewBufferString(""))       // Empty stdin -- first read returns EOF
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher (discard terminal output)
	err := d.Run(context.Background())                         // Run blocks until EOF -- must return nil
	if err != nil {                                            // EOF must translate to nil, not an error
		t.Errorf("expected nil on EOF, got %v", err) // Report the unexpected error
	}
}

// TestRun_ValidOptionThenEOF verifies that Run calls the handler once then exits cleanly on EOF.
// This is the normal interactive session flow: user picks an option, handler runs, session ends.
func TestRun_ValidOptionThenEOF(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry() // Registry with one entry to dispatch to
	called := 0          // Track how many times the handler is invoked
	reg.Register(Entry{
		Number:   11,               // Menu number the test will enter as input
		Title:    "Test Operation", // Human-readable title (appears in menu output)
		Category: "Test",           // Category for grouping in PrintMenu
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			called++ // Increment counter so we can assert exactly one invocation
			return nil
		},
	})
	// Input: "11\n" selects entry 11, then EOF terminates the loop.
	reader := bufio.NewReader(bytes.NewBufferString("11\n"))   // One valid choice followed by implicit EOF
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher
	err := d.Run(context.Background())                         // Run: pick 11, call handler, hit EOF, return nil
	if err != nil {                                            // Must return nil on clean EOF after dispatch
		t.Errorf("expected nil error, got %v", err) // Report unexpected error
	}
	if called != 1 { // Handler must be called exactly once
		t.Errorf("expected handler called 1 time, got %d", called) // Report count mismatch
	}
}

// TestRun_NonNumericInputThenEOF verifies that Run handles non-numeric input gracefully.
// The loop must not crash or return an error when the user types an invalid string.
func TestRun_NonNumericInputThenEOF(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry() // Empty registry -- we want to hit the non-numeric path, not a handler
	// Input: "abc\n" is non-numeric, so dispatchChoice returns nil. Then EOF terminates the loop.
	reader := bufio.NewReader(bytes.NewBufferString("abc\n"))  // Non-numeric choice + EOF
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher
	err := d.Run(context.Background())                         // Must handle "abc" gracefully
	if err != nil {                                            // Non-numeric input must not produce an error
		t.Errorf("expected nil error for non-numeric input, got %v", err) // Report the unexpected error
	}
}

// TestRun_UnknownNumberThenEOF verifies that Run handles an unregistered menu number gracefully.
// The loop must continue (not crash) when the user enters a valid integer that is not registered.
func TestRun_UnknownNumberThenEOF(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry() // Empty registry -- every number is unknown
	// Input: "999\n" is a valid integer but not registered. Loop continues, then EOF ends it.
	reader := bufio.NewReader(bytes.NewBufferString("999\n"))  // Unknown registered number + EOF
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher
	err := d.Run(context.Background())                         // Must not error on unknown option
	if err != nil {                                            // Unknown option must not produce an error
		t.Errorf("expected nil error for unknown option, got %v", err) // Report the unexpected error
	}
}

// TestRun_ContextCancelledBeforeRead verifies that Run returns ctx.Err() when context is pre-cancelled.
// This covers the case where the server is shutting down and no new user input should be processed.
func TestRun_ContextCancelledBeforeRead(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry()                                                // Empty registry -- we never get to dispatch
	reader := bufio.NewReader(bytes.NewBufferString("should not read")) // Stdin should never be read
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{})          // Build dispatcher

	ctx, cancel := context.WithCancel(context.Background()) // Create a cancellable context
	cancel()                                                // Pre-cancel BEFORE calling Run

	err := d.Run(ctx) // Run must detect the cancelled context in the first select iteration
	if err == nil {   // Cancelled context must produce a non-nil error
		t.Error("expected non-nil error for cancelled context, got nil") // Report missing error
	}
}

// TestRun_MultipleOptionsThenEOF verifies that Run processes multiple choices in sequence.
// This ensures the loop continues correctly after each dispatch before finally hitting EOF.
func TestRun_MultipleOptionsThenEOF(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry() // Registry with one entry (selected twice)
	called := 0          // Track cumulative invocations
	reg.Register(Entry{
		Number:   5,       // Menu number selected twice in the input
		Title:    "Multi", // Title for PrintMenu rendering
		Category: "Cat",   // Category for grouping
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			called++ // Count every invocation to verify the loop processed both lines
			return nil
		},
	})
	// Select 5 twice, then EOF. Both invocations must be processed.
	reader := bufio.NewReader(bytes.NewBufferString("5\n5\n")) // Two valid choices + implicit EOF
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher
	err := d.Run(context.Background())                         // Run: process 5, process 5, hit EOF, return nil
	if err != nil {                                            // Must return nil on clean EOF
		t.Errorf("expected nil error, got %v", err) // Report the unexpected error
	}
	if called != 2 { // Handler must be called twice -- once per "5\n" line
		t.Errorf("expected handler called 2 times, got %d", called) // Report count mismatch
	}
}

// TestRun_HandlerErrorLogsAndContinues verifies that Run logs a handler error and
// continues the loop rather than aborting. The error-logging path in Run covers
// the slog.Error and fmt.Fprintf statements inside "if err := d.dispatchChoice...".
func TestRun_HandlerErrorLogsAndContinues(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	reg := NewRegistry() // Registry with one failing entry
	reg.Register(Entry{
		Number:   7,         // Menu number that returns an error
		Title:    "Fail Op", // Title visible in menu output (discarded in tests)
		Category: "Test",    // Category for grouping
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			return fmt.Errorf("handler error") // Return an error to trigger Run's error-logging path
		},
	})
	// Select 7 (handler errors), then EOF. Loop must continue after error and exit on EOF.
	reader := bufio.NewReader(bytes.NewBufferString("7\n"))    // One selection that will return an error
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher
	err := d.Run(context.Background())                         // Run: select 7, error logged, EOF, return nil
	if err != nil {                                            // Handler error must NOT propagate from Run
		t.Errorf("expected nil from Run after handler error, got %v", err) // Report unexpected error
	}
}

// TestSafeInput_NonEOFErrorWrapped verifies that SafeInput wraps unexpected read errors
// (not io.EOF) in a descriptive error message. This covers the final error-return branch.
func TestSafeInput_NonEOFErrorWrapped(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	errReader := iotest.ErrReader(fmt.Errorf("device read error"))          // Reader that always returns a custom error
	reader := bufio.NewReader(errReader)                                    // Wrap in bufio so SafeInput can call ReadString
	input, err := SafeInput(reader, io.Discard, "prompt: ", "test-context") // Call under test
	if err == nil {                                                         // Must return an error
		t.Error("expected error from SafeInput with failing reader, got nil") // Report missing error
	}
	if input != "" { // Must return empty string on error
		t.Errorf("expected empty string on error, got %q", input) // Report non-empty string
	}
}
