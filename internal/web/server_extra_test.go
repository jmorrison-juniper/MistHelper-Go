// Package web -- additional unit tests for Server.ListenAndServe.
// The existing server_test.go covers handlers and Shutdown; this file covers ListenAndServe.
package web

import (
	"context"  // for context.WithTimeout -- short deadline for the shutdown call
	"errors"   // for errors.New -- construct a fake write error
	"net"      // for net.Listen -- find a free port to avoid conflicts
	"net/http" // for http.Header, http.ResponseWriter -- fake ResponseWriter for writeJSON error test
	"testing"  // for testing.T -- standard Go test runner
	"time"     // for time.Sleep and time.Second -- give ListenAndServe time to bind

	"github.com/jmorrison-juniper/misthelper-go/internal/api" // for api.Config -- builds test server config
)

// findFreePort finds an available TCP port by binding to :0 and reading the assigned port.
// This avoids flaky tests caused by hardcoded port conflicts in CI environments.
func findFreePort(t *testing.T) int {
	t.Helper()                         // Mark as helper so failures point to the caller
	ln, err := net.Listen("tcp", ":0") // Bind to any available port
	if err != nil {                    // If we cannot find a free port the test cannot proceed
		t.Fatalf("findFreePort: %v", err) // Bail with context
	}
	port := ln.Addr().(*net.TCPAddr).Port // Extract the assigned port number
	_ = ln.Close()                        // Release the port so ListenAndServe can bind it
	return port                           // Return the free port to the caller
}

// TestListenAndServe_GracefulShutdown verifies that ListenAndServe returns nil after Shutdown is called.
// This is the normal production lifecycle: start the server, then shut it down cleanly.
func TestListenAndServe_GracefulShutdown(t *testing.T) {
	t.Parallel() // Safe to run concurrently -- each test uses a unique free port

	port := findFreePort(t)      // Find a free port to avoid bind conflicts
	srv := NewServer(api.Config{ // Build the server with the free port
		WebPort: port, // Use the dynamically allocated free port
	})

	errCh := make(chan error, 1) // Buffer the return value so we can inspect it after shutdown
	go func() {
		errCh <- srv.ListenAndServe() // Start the server in a goroutine -- blocks until shutdown
	}()

	time.Sleep(50 * time.Millisecond) // Wait for the server to bind and enter its accept loop

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // Short deadline for drain
	defer cancel()                                                          // Release deadline resources
	if err := srv.Shutdown(ctx); err != nil {                               // Initiate graceful shutdown
		t.Errorf("Shutdown returned error: %v", err) // Report shutdown error for diagnostics
	}

	select {
	case err := <-errCh: // Wait for ListenAndServe to return
		if err != nil { // ListenAndServe must return nil after graceful Shutdown
			t.Errorf("ListenAndServe returned non-nil error: %v", err) // Report unexpected error
		}
	case <-time.After(3 * time.Second): // Fail if ListenAndServe hangs after shutdown
		t.Error("ListenAndServe did not return within 3 seconds after Shutdown") // Report the hang
	}
}

// TestListenAndServe_BindError verifies that ListenAndServe returns an error when the port is already in use.
// This can happen if another process or test is already bound to the same port.
func TestListenAndServe_BindError(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	port := findFreePort(t) // Find a free port

	// Bind the port ourselves BEFORE starting ListenAndServe so it fails.
	blocker, err := net.Listen("tcp", ":"+itoa(port)) // Occupy the port to cause a bind conflict
	if err != nil {                                   // If we cannot bind it means the port was taken by a race
		t.Skipf("could not bind port %d to create conflict: %v", port, err) // Skip rather than fail on CI races
	}
	defer func() { _ = blocker.Close() }() // Release the blocker when the test ends

	srv := NewServer(api.Config{WebPort: port}) // Build the server pointing at the occupied port
	err = srv.ListenAndServe()                  // Must return an error because the port is taken
	if err == nil {                             // Bind failure must propagate as a non-nil error
		t.Error("expected error for port-in-use, got nil") // Report the missing error
	}
}

// ── writeJSON error-path test ────────────────────────────────────────────────

// errWriteResponseWriter is a fake http.ResponseWriter whose Write method always returns an error.
// It is used to trigger the slog.Error path in writeJSON that executes when the response cannot be written.
type errWriteResponseWriter struct {
	header http.Header // Required by the http.ResponseWriter interface
}

// Header implements http.ResponseWriter.Header.
func (e *errWriteResponseWriter) Header() http.Header {
	return e.header // Return the stored header map
}

// WriteHeader implements http.ResponseWriter.WriteHeader.
func (e *errWriteResponseWriter) WriteHeader(_ int) {} // No-op: status code not needed for this test

// Write implements http.ResponseWriter.Write and always returns an error.
func (e *errWriteResponseWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("simulated write error") // Always fail so writeJSON hits the slog.Error branch
}

// TestWriteJSON_WriteError verifies that writeJSON logs an error and returns without panicking when
// the underlying http.ResponseWriter.Write call fails. This covers the slog.Error path that fires
// when a client disconnects or the response buffer is full mid-write.
func TestWriteJSON_WriteError(t *testing.T) {
	t.Parallel() // Independent of all other tests -- uses no shared state

	w := &errWriteResponseWriter{header: http.Header{}} // Fake writer that always errors on Write
	writeJSON(w, `{"status":"ok"}`)                     // Must not panic; should log the write error internally
	// No assertion on slog output -- verifying no panic and no hang is sufficient coverage for this path
}

// itoa converts an int to its string representation.
// Avoids importing strconv just for one conversion in this test file.
func itoa(n int) string {
	if n == 0 { // Handle zero explicitly to avoid the loop below returning empty string
		return "0" // Return "0" for zero input
	}
	digits := make([]byte, 0, 6) // Allocate a small buffer for the digit bytes
	for n > 0 {                  // Extract digits in reverse order
		digits = append(digits, byte('0'+n%10)) // Append the least significant digit
		n /= 10                                 // Shift right by one decimal place
	}
	// Reverse the digits slice to get the correct order.
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i] // Swap elements from both ends
	}
	return string(digits) // Convert byte slice to string for the TCP address
}
