// Package ssh -- integration tests covering ListenAndServe, acceptLoop, handleConn,
// handleChannel, processChannelRequests, runMenuSession, prepareSessionDir,
// crlfWriter.Write, and Shutdown.
package ssh

import (
	"bytes"         // for bytes.Buffer -- captures crlfWriter output in unit tests
	"context"       // for context.WithCancel -- controls server lifecycle in integration tests
	"fmt"           // for fmt.Sprintf -- builds listener address strings
	"net"           // for net.Listen and net.DialTimeout -- port detection helpers
	"os"            // for os.Stat and os.WriteFile -- prepareSessionDir verification
	"path/filepath" // for filepath.Join -- cross-platform path construction
	"testing"       // for testing.T -- standard test runner
	"time"          // for time.After -- timeout guards in goroutine-based tests

	gossh "golang.org/x/crypto/ssh" // SSH client for integration tests

	"github.com/jmorrison-juniper/misthelper-go/internal/api"    // for api.Config -- test server configuration
	"github.com/jmorrison-juniper/misthelper-go/internal/menu"   // for menu.NewRegistry -- empty registry for tests
	"github.com/jmorrison-juniper/misthelper-go/internal/output" // for output.Writer -- no-op implementation
)

// ── No-op output writer ───────────────────────────────────────────────────────

// integrationNoopWriter satisfies output.Writer with no side effects.
// Used so integration tests don't create real CSV or SQLite output during the test run.
type integrationNoopWriter struct{}

// Write discards all records -- output backend behaviour is not under test here.
func (integrationNoopWriter) Write(_ context.Context, _ string, _ []map[string]any) error {
	return nil // Discard all records so tests leave no output files on disk
}

// Close is a no-op -- no real backend was opened.
func (integrationNoopWriter) Close() error {
	return nil // Nothing to close -- no real backend was opened
}

// compile-time check that integrationNoopWriter satisfies the Writer interface.
var _ output.Writer = integrationNoopWriter{}

// ── Port helpers ──────────────────────────────────────────────────────────────

// findFreePort binds :0 on localhost to let the OS assign a free port, then releases it.
// The caller binds the same port immediately; a TOCTOU race is acceptable in test code.
func findFreePort(t *testing.T) int {
	t.Helper()                                  // Mark as helper so failure attribution points to the caller
	ln, err := net.Listen("tcp", "127.0.0.1:0") // Bind :0 to request an OS-assigned free port
	if err != nil {                             // net.Listen failure is unexpected in a test environment
		t.Fatalf("findFreePort: net.Listen: %v", err) // Fail immediately with the error detail
	}
	port := ln.Addr().(*net.TCPAddr).Port // Extract the assigned port number from the listener address
	_ = ln.Close()                        // Release the port so the server under test can bind it
	return port                           // Return the port number to the caller
}

// waitForListening polls a TCP address until it accepts a connection or the deadline passes.
// Used to synchronise tests with the background goroutine running ListenAndServe.
func waitForListening(t *testing.T, addr string) {
	t.Helper()                                  // Mark as helper so failure attribution is clear
	deadline := time.Now().Add(3 * time.Second) // Give the server 3 seconds to bind the port
	for time.Now().Before(deadline) {           // Poll until the deadline expires
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond) // Probe the address
		if err == nil {                                                 // A successful dial means the server is accepting connections
			_ = conn.Close() // Close the probe connection -- it was for detection only
			return           // Return; the server is ready
		}
		time.Sleep(10 * time.Millisecond) // Brief pause to avoid a busy-wait CPU spin
	}
	t.Fatalf("server not listening on %s after 3 seconds", addr) // Fail if the server never started
}

// ── Test server factory ───────────────────────────────────────────────────────

// newTestServer creates a Server wired with test credentials, a temp host key,
// and a writable temp sessions directory so tests do not write to the source tree.
func newTestServer(t *testing.T, port int) *Server {
	t.Helper()                                 // Mark as helper for clean failure attribution
	keyDir := t.TempDir()                      // Isolated temp dir so each test has its own host key
	signer, err := LoadOrCreateHostKey(keyDir) // Generate a fresh test-only host key
	if err != nil {                            // Key generation must succeed or the test is invalid
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail immediately with the error detail
	}
	cfg := api.Config{ // Minimal config with known credentials
		SSHPort:      port,       // Port assigned by findFreePort -- avoids conflicts
		SSHUser:      "testuser", // Known username matched by clientConfig()
		SSHPassword:  "testpass", // Known password matched by clientConfig()
		OutputFormat: "csv",      // CSV format -- no real files created (noopWriter)
	}
	return &Server{ // Wire the server with test-only dependencies
		cfg:         cfg,                     // Config carries port and credentials
		signer:      signer,                  // Host key for SSH server identity
		registry:    menu.NewRegistry(),      // Empty registry -- no real menu entries needed
		writer:      integrationNoopWriter{}, // No-op writer -- tests must not create output files
		sessionsDir: t.TempDir(),             // Writable temp dir for per-session subdirectories
	}
}

// clientConfig returns a gossh.ClientConfig for the credentials used by newTestServer.
// HostKeyCallback uses InsecureIgnoreHostKey because test servers use ephemeral host keys.
func clientConfig() *gossh.ClientConfig {
	return &gossh.ClientConfig{
		User:            "testuser",                                     // Must match Server.cfg.SSHUser
		Auth:            []gossh.AuthMethod{gossh.Password("testpass")}, // Password matches Server.cfg.SSHPassword
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),                  // Accept any host key -- tests use ephemeral keys
		Timeout:         5 * time.Second,                                // Fail fast if the server doesn't respond
	}
}

// ── crlfWriter unit tests ─────────────────────────────────────────────────────

// TestCRLFWriter_ConvertsLFtoCRLF verifies that bare \n bytes are converted to \r\n.
// PTY terminals require CRLF; without this conversion the cursor does not return to column 0.
func TestCRLFWriter_ConvertsLFtoCRLF(t *testing.T) {
	t.Parallel()                                 // Independent of all other tests
	var buf bytes.Buffer                         // In-memory buffer captures the converted output
	cw := &crlfWriter{w: &buf}                   // Wrap the buffer in the CRLF converter under test
	n, err := cw.Write([]byte("hello\nworld\n")) // Write a string with bare LF characters
	if err != nil {                              // Write to an in-memory buffer must never fail
		t.Fatalf("Write returned unexpected error: %v", err) // Bail with the error detail
	}
	if n != 12 { // Must return the ORIGINAL byte count per io.Writer contract
		t.Errorf("Write returned n=%d; want 12 (original input length)", n) // Report the wrong count
	}
	want := "hello\r\nworld\r\n"          // Expected CRLF output
	if got := buf.String(); got != want { // Every LF must become CRLF in the output
		t.Errorf("crlfWriter output = %q; want %q", got, want) // Report the conversion failure
	}
}

// TestCRLFWriter_NoNewlines verifies that data without LF is passed through unchanged.
func TestCRLFWriter_NoNewlines(t *testing.T) {
	t.Parallel()                                        // Independent of all other tests
	var buf bytes.Buffer                                // In-memory buffer captures output
	cw := &crlfWriter{w: &buf}                          // Wrap buffer in the CRLF converter
	_, _ = cw.Write([]byte("no newlines here"))         // Write data that contains no LF characters
	if got := buf.String(); got != "no newlines here" { // Output must be bit-for-bit identical to input
		t.Errorf("crlfWriter modified data without LF: got %q", got) // Report the unexpected modification
	}
}

// TestCRLFWriter_EmptyInput verifies that writing zero bytes succeeds without error.
func TestCRLFWriter_EmptyInput(t *testing.T) {
	t.Parallel()                 // Independent of all other tests
	var buf bytes.Buffer         // In-memory buffer
	cw := &crlfWriter{w: &buf}   // Wrap buffer in the CRLF converter
	n, err := cw.Write([]byte{}) // Write an empty byte slice
	if err != nil {              // Empty writes must not return an error
		t.Fatalf("Write returned error for empty input: %v", err) // Bail with error detail
	}
	if n != 0 { // Empty input must report length 0
		t.Errorf("Write returned n=%d for empty input; want 0", n) // Report the wrong count
	}
}

// TestCRLFWriter_MultipleNewlines verifies that multiple LF characters are all converted.
func TestCRLFWriter_MultipleNewlines(t *testing.T) {
	t.Parallel()                                                   // Independent of all other tests
	var buf bytes.Buffer                                           // In-memory buffer
	cw := &crlfWriter{w: &buf}                                     // Wrap buffer in the CRLF converter
	_, _ = cw.Write([]byte("a\nb\nc\n"))                           // Write three lines ending with LF
	if got, want := buf.String(), "a\r\nb\r\nc\r\n"; got != want { // Every LF must be converted
		t.Errorf("crlfWriter output = %q; want %q", got, want) // Report the conversion failure
	}
}

// ── Shutdown unit tests ───────────────────────────────────────────────────────

// TestShutdown_NoActiveSessions verifies that Shutdown returns nil immediately
// when no sessions are active (the WaitGroup counter is zero from the start).
func TestShutdown_NoActiveSessions(t *testing.T) {
	t.Parallel()                                                  // Independent of all other tests
	port := findFreePort(t)                                       // Allocate a port (server won't listen -- Shutdown only)
	server := newTestServer(t, port)                              // Build a server with no active sessions
	if err := server.Shutdown(context.Background()); err != nil { // WaitGroup is zero -- must return nil immediately
		t.Errorf("Shutdown returned unexpected error: %v; want nil", err) // Report unexpected error
	}
}

// TestShutdown_ContextCancelled verifies that Shutdown returns ctx.Err() when the
// caller cancels the shutdown context before all sessions finish.
func TestShutdown_ContextCancelled(t *testing.T) {
	t.Parallel()                     // Independent of all other tests
	port := findFreePort(t)          // Allocate a port (not used -- testing Shutdown only)
	server := newTestServer(t, port) // Build a server with no real sessions
	server.wg.Add(1)                 // Simulate one active session that never finishes

	ctx, cancel := context.WithCancel(context.Background()) // Cancellable shutdown context
	cancel()                                                // Pre-cancel to simulate an impatient caller

	err := server.Shutdown(ctx) // Must return ctx.Err() because context is already cancelled
	server.wg.Done()            // Release the fake session counter so the goroutine does not leak
	if err == nil {             // A cancelled context must produce a non-nil error
		t.Error("Shutdown returned nil for a pre-cancelled context; want ctx.Err()")
	}
}

// ── ListenAndServe tests ──────────────────────────────────────────────────────

// TestListenAndServe_BindError verifies that ListenAndServe returns a wrapped error
// when the configured port is already occupied by another listener.
func TestListenAndServe_BindError(t *testing.T) {
	t.Parallel()                                                 // Independent of all other tests
	port := findFreePort(t)                                      // Allocate a free port
	occupier, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // Hold the wildcard bind so the server cannot bind on dual-stack hosts
	if err != nil {                                              // If we can't occupy the port the test is invalid
		t.Fatalf("failed to occupy port %d: %v", port, err) // Bail with context
	}
	defer func() { _ = occupier.Close() }() // Release the occupying listener after the test

	server := newTestServer(t, port)                                               // Build a server targeting the occupied port
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond) // Safety timeout so a false-positive bind cannot hang this test
	defer cancel()                                                                 // Ensure context resources are always released
	errCh := make(chan error, 1)                                                   // Buffered channel so goroutine cannot block on send
	go func() { errCh <- server.ListenAndServe(ctx) }()                            // Run ListenAndServe in goroutine to avoid blocking this test forever

	select {
	case err = <-errCh: // Collect result and assert bind failure
		if err == nil { // A nil error means the bind unexpectedly succeeded and returned only after timeout
			t.Error("ListenAndServe returned nil on occupied port; want bind error")
		}
	case <-time.After(3 * time.Second): // Hard stop if goroutine somehow wedges despite timeout
		t.Fatal("ListenAndServe bind-error test timed out")
	}
}

// TestListenAndServe_ContextCancelReturnsNil verifies that cancelling the context
// causes ListenAndServe to close the listener and return nil (clean shutdown).
func TestListenAndServe_ContextCancelReturnsNil(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	port := findFreePort(t)                                 // Allocate a free port for the server
	server := newTestServer(t, port)                        // Build the test server
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context for clean shutdown
	done := make(chan error, 1)                             // Buffered so the goroutine never blocks on send
	go func() { done <- server.ListenAndServe(ctx) }()      // Start server in a background goroutine

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	waitForListening(t, addr) // Synchronise: wait until the server is actually accepting connections

	cancel() // Signal shutdown -- the goroutine inside ListenAndServe will close the listener
	select {
	case err := <-done: // ListenAndServe must return nil on clean context cancellation
		if err != nil {
			t.Errorf("ListenAndServe returned error %v; want nil", err) // Report unexpected error
		}
	case <-time.After(5 * time.Second): // Safety net -- fail if ListenAndServe never returns
		t.Error("ListenAndServe did not return within 5 seconds after ctx cancel")
	}
}

// ── prepareSessionDir unit tests ─────────────────────────────────────────────

// TestPrepareSessionDir_Success verifies that prepareSessionDir creates the expected
// session subdirectory under sessionsDir.
func TestPrepareSessionDir_Success(t *testing.T) {
	t.Parallel()                                                // Independent of all other tests
	port := findFreePort(t)                                     // Port is not used -- testing prepareSessionDir directly
	server := newTestServer(t, port)                            // newTestServer sets sessionsDir to t.TempDir()
	if err := server.prepareSessionDir("test123"); err != nil { // Must create the directory without error
		t.Fatalf("prepareSessionDir returned unexpected error: %v", err) // Bail with the error detail
	}
	expected := filepath.Join(server.sessionsDir, "session_test123") // Expected path under sessionsDir
	if _, statErr := os.Stat(expected); statErr != nil {             // Directory must exist on disk
		t.Errorf("session directory not created at %s: %v", expected, statErr) // Report missing directory
	}
}

// TestPrepareSessionDir_Error verifies that prepareSessionDir returns a wrapped error
// when the parent path is a file (not a directory), making MkdirAll impossible.
func TestPrepareSessionDir_Error(t *testing.T) {
	t.Parallel()                     // Independent of all other tests
	port := findFreePort(t)          // Port is not used -- testing prepareSessionDir directly
	server := newTestServer(t, port) // newTestServer sets sessionsDir to t.TempDir()

	// Create a regular file where sessionsDir points -- MkdirAll cannot create a child dir inside a file
	parentDir := t.TempDir()                                               // Writable temp dir for the blocking file
	blockingFile := filepath.Join(parentDir, "blocked")                    // Path for the file that will block dir creation
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil { // Create the blocking file
		t.Fatalf("failed to create blocking file: %v", err) // Bail if file creation fails
	}
	server.sessionsDir = blockingFile // Point sessionsDir at a file -- MkdirAll inside a file must fail

	err := server.prepareSessionDir("badid") // Must fail -- cannot create a directory inside a file
	if err == nil {                          // Nil error here means the error path is not covered
		t.Error("prepareSessionDir returned nil for an invalid sessions path; want error")
	}
}

// ── Integration tests ─────────────────────────────────────────────────────────

// TestIntegration_BadPassword verifies that the server rejects clients with wrong credentials.
// handleConn must fail at the SSH handshake and not produce any session directories.
func TestIntegration_BadPassword(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	port := findFreePort(t)                                 // Allocate a free port for the server
	server := newTestServer(t, port)                        // Build the test server
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context for clean shutdown
	defer cancel()                                          // Ensure server shuts down when the test ends
	go func() { _ = server.ListenAndServe(ctx) }()          // Start server in background goroutine

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	waitForListening(t, addr) // Wait for the server to be ready before connecting

	badCfg := &gossh.ClientConfig{
		User:            "testuser",                                      // Correct username
		Auth:            []gossh.AuthMethod{gossh.Password("wrongpass")}, // Wrong password -- should be rejected
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),                   // Accept any host key
		Timeout:         3 * time.Second,                                 // Fail fast if no response
	}
	_, err := gossh.Dial("tcp", addr, badCfg) // Must fail -- server rejects the wrong password
	if err == nil {                           // A nil error means auth was unexpectedly accepted
		t.Error("Dial succeeded with wrong password; expected authentication failure")
	}
}

// TestIntegration_FullSessionLifecycle exercises the complete SSH path:
// ListenAndServe -> acceptLoop -> handleConn -> prepareSessionDir -> handleChannel ->
// processChannelRequests (pty-req, shell) -> runMenuSession (exit via EOF).
func TestIntegration_FullSessionLifecycle(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	port := findFreePort(t)                                 // Allocate a free port
	server := newTestServer(t, port)                        // Build the test server
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable server context
	done := make(chan error, 1)                             // Receives the return value of ListenAndServe
	go func() { done <- server.ListenAndServe(ctx) }()      // Start server in background goroutine

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	waitForListening(t, addr) // Synchronise with the server goroutine

	// Connect as an authenticated SSH client -- triggers handleConn
	client, err := gossh.Dial("tcp", addr, clientConfig())
	if err != nil {
		t.Fatalf("gossh.Dial failed: %v", err) // Bail if connection fails unexpectedly
	}

	// Open a session channel -- triggers handleChannel accepting a "session" type
	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		t.Fatalf("NewSession failed: %v", err) // Bail if session open fails
	}

	// Request PTY -- exercises the "pty-req" branch in processChannelRequests
	if err := session.RequestPty("xterm", 24, 80, gossh.TerminalModes{}); err != nil {
		_ = session.Close()
		_ = client.Close()
		t.Fatalf("RequestPty failed: %v", err) // PTY request must be accepted by the server
	}

	// Get stdin pipe so we can close it to send EOF to the menu dispatcher
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		t.Fatalf("StdinPipe failed: %v", err) // Stdin pipe must be available for an open session
	}

	// Start shell -- exercises the "shell" branch, triggers runMenuSession
	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		t.Fatalf("Shell failed: %v", err) // Shell request must be accepted by the server
	}

	// Close stdin immediately -- sends EOF to the server's bufio.Reader,
	// causing the menu dispatcher to exit cleanly (Run returns nil on EOF).
	if err := stdin.Close(); err != nil {
		t.Logf("stdin.Close: %v (non-fatal)", err) // Log but don't fail -- session may already be closing
	}

	// Wait for the session to end with a timeout guard so the test cannot hang
	sessionDone := make(chan error, 1)            // Receives session.Wait() result
	go func() { sessionDone <- session.Wait() }() // Wait in a goroutine for timeout safety
	select {
	case <-sessionDone: // Session closed -- server finished runMenuSession
	case <-time.After(5 * time.Second): // Safety net -- fail if session never completes
		t.Error("SSH session did not complete within 5 seconds")
	}

	_ = client.Close() // Close SSH connection -- unblocks handleConn's channel loop
	cancel()           // Signal server shutdown
	select {
	case err := <-done: // ListenAndServe must return nil on clean shutdown
		if err != nil {
			t.Errorf("ListenAndServe returned error %v; want nil", err)
		}
	case <-time.After(5 * time.Second): // Safety net -- fail if server never shuts down
		t.Error("ListenAndServe did not return after context cancel")
	}
}

// TestIntegration_NonSessionChannelRejected verifies that handleChannel rejects
// channel types other than "session" with an error response.
func TestIntegration_NonSessionChannelRejected(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	port := findFreePort(t)                                 // Allocate a free port
	server := newTestServer(t, port)                        // Build the test server
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable server context
	defer cancel()                                          // Ensure clean server shutdown
	go func() { _ = server.ListenAndServe(ctx) }()          // Start server in background goroutine

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	waitForListening(t, addr) // Wait for the server to be ready

	client, err := gossh.Dial("tcp", addr, clientConfig()) // Authenticated connection
	if err != nil {
		t.Fatalf("gossh.Dial failed: %v", err) // Bail if connection fails
	}
	defer func() { _ = client.Close() }() // Ensure connection is released after the test

	// Attempt to open a non-session channel type -- handleChannel must reject it
	_, _, err = client.OpenChannel("x-unsupported-type", nil) // Not "session" -- must be rejected
	if err == nil {                                           // Nil error means the server incorrectly accepted it
		t.Error("OpenChannel with non-session type succeeded; want rejection")
	}
}

// TestIntegration_UnknownChannelRequest verifies the "default" branch in processChannelRequests --
// an unknown request type receives Reply(false) from the server.
func TestIntegration_UnknownChannelRequest(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	port := findFreePort(t)                                 // Allocate a free port
	server := newTestServer(t, port)                        // Build the test server
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable server context
	defer cancel()                                          // Ensure clean server shutdown
	go func() { _ = server.ListenAndServe(ctx) }()          // Start server in background goroutine

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	waitForListening(t, addr) // Wait for the server to be ready

	client, err := gossh.Dial("tcp", addr, clientConfig()) // Authenticated connection
	if err != nil {
		t.Fatalf("gossh.Dial failed: %v", err) // Bail if connection fails
	}
	defer func() { _ = client.Close() }() // Ensure connection is released after the test

	session, err := client.NewSession() // Open a session channel
	if err != nil {
		t.Fatalf("NewSession failed: %v", err) // Bail if session open fails
	}
	defer func() { _ = session.Close() }() // Ensure session is released

	// Send a custom channel request -- the server's default case must reply false
	ok, err := session.SendRequest("x-custom-unknown-req", true, nil) // WantReply=true so we get an answer
	if err != nil {                                                   // SendRequest error is unexpected (channel is open)
		t.Fatalf("SendRequest returned unexpected error: %v", err)
	}
	if ok { // Server must deny unknown requests
		t.Error("unknown channel request was accepted (ok=true); want rejection (ok=false)")
	}
}

// TestIntegration_WindowChange verifies that window-change requests are handled
// without error, exercising the "window-change" branch in processChannelRequests.
func TestIntegration_WindowChange(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	port := findFreePort(t)                                 // Allocate a free port
	server := newTestServer(t, port)                        // Build the test server
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable server context
	defer cancel()                                          // Ensure clean server shutdown
	go func() { _ = server.ListenAndServe(ctx) }()          // Start server in background goroutine

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	waitForListening(t, addr) // Wait for the server to be ready

	client, err := gossh.Dial("tcp", addr, clientConfig()) // Authenticated connection
	if err != nil {
		t.Fatalf("gossh.Dial failed: %v", err) // Bail if connection fails
	}
	defer func() { _ = client.Close() }() // Ensure connection is released

	session, err := client.NewSession() // Open a session channel
	if err != nil {
		t.Fatalf("NewSession failed: %v", err) // Bail if session open fails
	}
	defer func() { _ = session.Close() }() // Ensure session is released

	// WindowChange sends a "window-change" channel request -- must be handled without error
	if err := session.WindowChange(48, 160); err != nil { // Resize to 48 rows x 160 cols
		t.Errorf("WindowChange returned unexpected error: %v", err) // Report the error
	}
}
