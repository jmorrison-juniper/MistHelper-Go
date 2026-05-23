package menu

import (
	"bufio"   // for bufio.NewReader -- wrap bytes.Reader as buffered stdin
	"bytes"   // for bytes.NewBufferString -- create in-memory stdin for tests
	"context" // for context.Background -- simple non-cancelling context for tests
	"errors"  // for errors.Is -- assert specific error types
	"io"      // for io.EOF -- expected return from SafeInput on empty reader
	"testing" // for testing.T -- standard test runner

	"github.com/jmorrison-juniper/misthelper-go/internal/output" // for output.Writer -- handler signature requires it
)

// stubWriter implements output.Writer without touching the filesystem.
// It records the number of Write calls so tests can assert handler invocation.
type stubWriter struct {
	writes int // Count of Write calls received
}

func (s *stubWriter) Write(_ context.Context, _ string, _ []map[string]any) error {
	s.writes++ // Increment counter on each Write call
	return nil // Always succeed -- tests control failure via the handler, not the writer
}

func (s *stubWriter) Close() error {
	return nil // No resources to release in the stub
}

// TestDispatch_ValidOption verifies that Dispatch invokes the registered handler exactly once.
func TestDispatch_ValidOption(t *testing.T) {
	t.Parallel()         // Run this test concurrently with others
	reg := NewRegistry() // Create a fresh registry for this test
	called := 0          // Track how many times the stub handler is invoked
	reg.Register(Entry{  // Register a non-destructive entry at number 11
		Number:   11,               // Menu number to exercise
		Title:    "Test Operation", // Human-readable title
		Category: "Test",           // Category grouping (irrelevant to this test)
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			called++   // Increment counter to verify the handler was invoked
			return nil // Return success to the dispatcher
		},
	})
	reader := bufio.NewReader(bytes.NewBufferString(""))       // Empty stdin -- Dispatch doesn't need stdin
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher with the test registry (discard terminal output)
	err := d.Dispatch(context.Background(), 11)                // Invoke entry 11 directly
	if err != nil {                                            // Dispatch must return nil for a valid option
		t.Fatalf("expected nil error, got %v", err)
	}
	if called != 1 { // Handler must be called exactly once
		t.Errorf("expected handler called 1 time, got %d", called)
	}
}

// TestDispatch_UnknownOption verifies that Dispatch returns an error for an unregistered number.
func TestDispatch_UnknownOption(t *testing.T) {
	t.Parallel()                                               // Safe to run concurrently
	reg := NewRegistry()                                       // Empty registry -- no entries registered
	reader := bufio.NewReader(bytes.NewBufferString(""))       // Empty stdin
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Build dispatcher (discard terminal output)
	err := d.Dispatch(context.Background(), 999)               // 999 is not registered
	if err == nil {                                            // Must return an error for unknown numbers
		t.Fatal("expected error for unknown option 999, got nil")
	}
}

// TestDispatch_DestructiveRequiresConfirm verifies that a destructive entry runs only after "CONFIRM".
func TestDispatch_DestructiveRequiresConfirm(t *testing.T) {
	t.Parallel()         // Safe to run concurrently
	reg := NewRegistry() // Fresh registry for this test
	called := 0          // Track handler invocations
	reg.Register(Entry{  // Register a destructive entry
		Number:      90,                 // Destructive operations are 90-100
		Title:       "Firmware Upgrade", // Human-readable title for the confirmation warning
		Category:    "Destructive",      // Category grouping
		Destructive: true,               // Gate requires "CONFIRM" before handler runs
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			called++   // Increment to verify the handler actually ran
			return nil // Return success
		},
	})
	// Feed "CONFIRM\n" as stdin so confirmDestructive accepts it.
	reader := bufio.NewReader(bytes.NewBufferString("CONFIRM\n")) // Exact match required
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{})    // Build dispatcher (discard terminal output)
	err := d.Dispatch(context.Background(), 90)                   // Invoke the destructive entry
	if err != nil {                                               // Must return nil on successful confirm+run
		t.Fatalf("expected nil error, got %v", err)
	}
	if called != 1 { // Handler must run exactly once after confirmation
		t.Errorf("expected handler called 1 time, got %d", called)
	}
}

// TestSafeInput_EOF verifies that SafeInput returns io.EOF without panicking on an empty reader.
func TestSafeInput_EOF(t *testing.T) {
	t.Parallel()                                                             // Safe to run concurrently
	reader := bufio.NewReader(bytes.NewBufferString(""))                     // Empty buffer -- first read returns EOF
	result, err := SafeInput(reader, io.Discard, "prompt> ", "test-context") // Call under test (discard prompt output)
	if !errors.Is(err, io.EOF) {                                             // Must return io.EOF, not nil or another error
		t.Errorf("expected io.EOF, got %v", err)
	}
	if result != "" { // Must return empty string on EOF
		t.Errorf("expected empty string, got %q", result)
	}
}

// TestRegistry_Sorted verifies that Sorted returns entries in ascending Number order.
func TestRegistry_Sorted(t *testing.T) {
	t.Parallel()                                                      // Safe to run concurrently
	reg := NewRegistry()                                              // Fresh registry
	reg.Register(Entry{Number: 30, Title: "Third", Category: "Cat"})  // Register out of order
	reg.Register(Entry{Number: 10, Title: "First", Category: "Cat"})  // Lowest number
	reg.Register(Entry{Number: 20, Title: "Second", Category: "Cat"}) // Middle number
	sorted := reg.Sorted()                                            // Call under test
	if len(sorted) != 3 {                                             // Must contain all three entries
		t.Fatalf("expected 3 entries, got %d", len(sorted))
	}
	if sorted[0].Number != 10 || sorted[1].Number != 20 || sorted[2].Number != 30 { // Must be ascending
		t.Errorf("expected [10 20 30], got [%d %d %d]", sorted[0].Number, sorted[1].Number, sorted[2].Number)
	}
}

// TestDispatch_DestructiveEOFCancels verifies that execute returns nil (not an error) when
// SafeInput returns EOF during the destructive confirmation prompt.
// This covers the confirmDestructive error path (return false, err) AND the
// execute "err != nil || !confirmed" cancellation path (slog.Info + return nil).
func TestDispatch_DestructiveEOFCancels(t *testing.T) {
	t.Parallel()         // Safe to run concurrently
	reg := NewRegistry() // Fresh registry for this test
	called := 0          // Track whether the handler was (incorrectly) invoked
	reg.Register(Entry{  // Destructive entry -- confirmation required before handler runs
		Number:      91,            // Arbitrary destructive op number
		Title:       "AP Reboot",   // Title shown in the confirmation warning
		Category:    "Destructive", // Category grouping
		Destructive: true,          // Triggers the confirmDestructive gate
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			called++ // Must NOT be incremented -- handler should be blocked by EOF
			return nil
		},
	})
	reader := bufio.NewReader(bytes.NewBufferString(""))       // EOF immediately -- no confirmation text
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{}) // Dispatcher with empty stdin
	err := d.Dispatch(context.Background(), 91)                // Dispatch destructive op -- must be cancelled
	if err != nil {                                            // Cancellation must return nil, not the EOF error
		t.Errorf("expected nil error on EOF cancellation, got %v", err) // Report unexpected error
	}
	if called != 0 { // Handler must NOT run when confirmation is aborted by EOF
		t.Errorf("handler must not run on EOF; invoked %d time(s)", called) // Report incorrect invocation
	}
}

// TestDispatch_DestructiveWrongTextCancels verifies that execute returns nil when the user
// types the wrong confirmation text. This covers the "!confirmed" branch in execute.
func TestDispatch_DestructiveWrongTextCancels(t *testing.T) {
	t.Parallel()         // Safe to run concurrently
	reg := NewRegistry() // Fresh registry for this test
	called := 0          // Track whether the handler was (incorrectly) invoked
	reg.Register(Entry{  // Destructive entry
		Number:      92,              // Arbitrary destructive op number
		Title:       "AP Reboot All", // Title shown in the confirmation warning
		Category:    "Destructive",   // Category grouping
		Destructive: true,            // Triggers the confirmDestructive gate
		Handler: func(ctx context.Context, r *bufio.Reader, term io.Writer, w output.Writer) error {
			called++ // Must NOT be incremented -- handler blocked by wrong confirmation text
			return nil
		},
	})
	reader := bufio.NewReader(bytes.NewBufferString("WRONG\n")) // Wrong text -- not "CONFIRM"
	d := NewDispatcher(reg, reader, io.Discard, &stubWriter{})  // Dispatcher with wrong-text stdin
	err := d.Dispatch(context.Background(), 92)                 // Dispatch -- should be cancelled
	if err != nil {                                             // Cancellation must return nil
		t.Errorf("expected nil error on wrong confirmation, got %v", err) // Report unexpected error
	}
	if called != 0 { // Handler must NOT run when confirmation text is wrong
		t.Errorf("handler must not run on wrong confirmation; invoked %d time(s)", called) // Report incorrect invocation
	}
}
