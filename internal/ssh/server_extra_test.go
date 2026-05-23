// Package ssh -- additional unit tests for newSessionID and buildServerConfig.
package ssh

import (
	"context" // for context.WithCancel -- test Shutdown ctx-cancelled and acceptLoop paths
	"errors"  // for errors.Is -- compare net.ErrClosed in fake listener
	"fmt"     // for fmt.Errorf -- construct a transient accept error and fake channel errors
	"net"     // for net.Conn, net.Listener, net.TCPAddr, net.ErrClosed -- fake listener types
	"regexp"  // for regexp.MustCompile -- validate session ID format
	"sync"    // for sync.Once -- ensure first-call logic runs exactly once in fake listener
	"testing" // for testing.T -- standard Go test runner
	"time"    // for time.Sleep -- give acceptLoop time to process the transient error

	gossh "golang.org/x/crypto/ssh" // for gossh.NewChannel, gossh.Request -- fake SSH channel types

	"github.com/jmorrison-juniper/misthelper-go/internal/api"  // for api.Config -- builds test server config
	"github.com/jmorrison-juniper/misthelper-go/internal/menu" // for menu.NewRegistry -- stub registry
)

// TestNewSessionID_NonEmpty verifies that newSessionID always returns a non-empty string.
// An empty session ID would cause the session directory to be non-unique, leading to collisions.
func TestNewSessionID_NonEmpty(t *testing.T) {
	t.Parallel()         // Safe to run concurrently
	id := newSessionID() // Call the function under test
	if id == "" {        // Session ID must never be empty
		t.Error("newSessionID returned empty string") // Report empty ID as a bug
	}
}

// TestNewSessionID_Unique verifies that two consecutive calls return different session IDs.
// Collision-free IDs are required for per-session directory isolation.
func TestNewSessionID_Unique(t *testing.T) {
	t.Parallel()          // Safe to run concurrently
	id1 := newSessionID() // First session ID
	id2 := newSessionID() // Second session ID -- random suffix makes collisions astronomically unlikely
	if id1 == id2 {       // Two IDs must be distinct with overwhelming probability
		t.Errorf("newSessionID returned same value twice: %q", id1) // Report the duplicate (would indicate broken rand)
	}
}

// TestNewSessionID_Format verifies that the session ID matches the expected timestamp_hex format.
// The format is "YYYYMMDD_HHMMSS_XXXX" where XXXX is 4 hex characters.
// Consistent format is required so per-session directories have predictable, sortable names.
func TestNewSessionID_Format(t *testing.T) {
	t.Parallel()                                               // Safe to run concurrently
	pattern := regexp.MustCompile(`^\d{8}_\d{6}_[0-9a-f]{4}$`) // YYYYMMDD_HHMMSS_4hexchars
	id := newSessionID()                                       // Generate an ID to validate
	if !pattern.MatchString(id) {                              // Must match the documented format
		t.Errorf("newSessionID %q does not match pattern %s", id, pattern.String()) // Report the malformed ID
	}
}

// TestBuildServerConfig_NotNil verifies that buildServerConfig returns a non-nil ServerConfig.
// A nil config would cause a panic when the SSH listener tries to accept connections.
func TestBuildServerConfig_NotNil(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	dir := t.TempDir()                      // Create a temp dir for the host key
	signer, err := LoadOrCreateHostKey(dir) // Generate a test host key
	if err != nil {                         // Key generation must succeed before testing config
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail if key generation fails
	}
	cfg := api.Config{ // Minimal config for building the SSH server config
		SSHUser:     "testuser", // Username to embed in the password callback
		SSHPassword: "testpass", // Password to embed in the password callback
	}
	server := NewServer(cfg, signer, menu.NewRegistry(), nil) // Build server with test dependencies
	config := server.buildServerConfig()                      // Call the function under test
	if config == nil {                                        // Config must never be nil
		t.Fatal("buildServerConfig returned nil") // Report nil config as a critical bug
	}
}

// TestBuildServerConfig_NoPublicKeyCallback verifies that the server config rejects public key auth.
// MistHelper uses password-only auth; public key callbacks must be nil so the library rejects them.
func TestBuildServerConfig_NoPublicKeyCallback(t *testing.T) {
	t.Parallel() // Safe to run concurrently

	dir := t.TempDir()                      // Temp dir for host key
	signer, err := LoadOrCreateHostKey(dir) // Generate a test host key
	if err != nil {                         // Key generation must succeed before testing
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail if key generation fails
	}
	cfg := api.Config{SSHUser: "u", SSHPassword: "p"} // Minimal config
	server := NewServer(cfg, signer, menu.NewRegistry(), nil)
	config := server.buildServerConfig() // Build the config under test
	if config.PublicKeyCallback != nil { // Public key auth must be explicitly disabled
		t.Error("PublicKeyCallback must be nil") // Report if it's accidentally set
	}
	if config.NoClientAuth { // Client auth must be required (not bypassed)
		t.Error("NoClientAuth must be false -- authentication is required") // Report the security misconfiguration
	}
}

// TestNewSessionID_MultipleFormatsValid verifies that multiple generated IDs all match the expected format.
// This strengthens confidence that the format is always valid, not just for a single lucky call.
func TestNewSessionID_MultipleFormatsValid(t *testing.T) {
	t.Parallel()                                               // Safe to run concurrently
	pattern := regexp.MustCompile(`^\d{8}_\d{6}_[0-9a-f]{4}$`) // Same pattern as in TestNewSessionID_Format
	for i := 0; i < 10; i++ {                                  // Generate 10 IDs to increase statistical confidence
		id := newSessionID()          // Generate one session ID
		if !pattern.MatchString(id) { // Each ID must match the format
			t.Errorf("iteration %d: ID %q does not match pattern", i, id) // Report the malformed ID with iteration number
		}
	}
}

// ── errOnFirstListener ────────────────────────────────────────────────────────
// errOnFirstListener is a fake net.Listener used in acceptLoop tests.
// The first Accept() call returns a transient (non-net.Error) error to exercise
// the slog.Error + continue path in acceptLoop.  Subsequent calls block until
// Close() is called, at which point they return net.ErrClosed.
type errOnFirstListener struct {
	once   sync.Once     // Ensures the first-call error is returned exactly once
	closed chan struct{} // Closed by Close() to unblock subsequent Accept() calls
}

// newErrOnFirstListener creates an errOnFirstListener ready for use in tests.
func newErrOnFirstListener() *errOnFirstListener {
	return &errOnFirstListener{closed: make(chan struct{})} // Buffered channel not needed -- Close blocks briefly
}

// Accept implements net.Listener.Accept.
func (l *errOnFirstListener) Accept() (net.Conn, error) {
	var isFirst bool                     // Tracks whether this is the first invocation
	l.once.Do(func() { isFirst = true }) // Only the first goroutine that reaches Do gets isFirst=true
	if isFirst {                         // First call returns a transient error to trigger slog.Error
		return nil, fmt.Errorf("transient accept error") // Non-net.Error: ctx.Err() check comes BEFORE slog.Error
	}
	<-l.closed                // Subsequent calls block until Close() is called
	return nil, net.ErrClosed // Return ErrClosed so acceptLoop can exit when ctx is done
}

// Close implements net.Listener.Close.
func (l *errOnFirstListener) Close() error {
	select {
	case <-l.closed: // Already closed -- idempotent
	default:
		close(l.closed) // Unblock all waiting Accept() calls
	}
	return nil // Close itself never fails in this fake
}

// Addr implements net.Listener.Addr.
func (l *errOnFirstListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0} // Fake address -- never used in acceptLoop
}

// ── acceptLoop tests ──────────────────────────────────────────────────────────

// TestAcceptLoop_TransientErrorLogged verifies that acceptLoop logs an error via slog.Error
// and continues the loop when listener.Accept() returns a non-net.Error while the context
// is still active. This is the "transient accept error" path -- not caused by shutdown.
func TestAcceptLoop_TransientErrorLogged(t *testing.T) {
	t.Parallel() // Independent of all other tests

	ln := newErrOnFirstListener()                           // First Accept returns transient error, then blocks
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context for clean exit

	s := &Server{}                                          // Minimal server -- acceptLoop uses s.wg and s.handleConn
	done := make(chan struct{})                             // Signal test when acceptLoop returns
	go func() { s.acceptLoop(ctx, ln, nil); close(done) }() // Run acceptLoop; nil config safe -- no real connections

	// Give acceptLoop time to: receive transient error, check ctx.Err()==nil,
	// call slog.Error, continue, and block on the second Accept call.
	time.Sleep(50 * time.Millisecond) // 50ms is orders of magnitude more than acceptLoop needs

	cancel()       // Set ctx.Err() to non-nil -- acceptLoop will exit after the next Accept error
	_ = ln.Close() // Unblock the blocking Accept() call -- returns net.ErrClosed

	select {
	case <-done: // acceptLoop returned cleanly
	case <-time.After(3 * time.Second): // Safety net: acceptLoop must not hang
		t.Error("acceptLoop did not return within 3 seconds after context cancel + listener close")
	}
}

// ── Shutdown tests ────────────────────────────────────────────────────────────

// TestShutdown_ContextCancelledReturnsError verifies that Shutdown returns ctx.Err() when
// the context is cancelled before all active sessions finish. This covers the
// "case <-ctx.Done(): return ctx.Err()" path that is not exercised by the normal shutdown test.
func TestShutdown_ContextCancelledReturnsError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	s := &Server{}    // Minimal server -- only wg and once are used by Shutdown
	s.wg.Add(1)       // Simulate one active session so wg.Wait() blocks indefinitely
	defer s.wg.Done() // Release the fake session when the test exits (cleanup)

	ctx, cancel := context.WithCancel(context.Background()) // Context we will cancel to trigger the error path
	cancel()                                                // Pre-cancel: ctx.Err() is non-nil before Shutdown is called

	err := s.Shutdown(ctx) // Shutdown must return ctx.Err() because ctx is already done
	if err == nil {        // Must NOT return nil -- the wg blocks the normal exit
		t.Error("Shutdown with cancelled context and active session must return non-nil error") // Report missing error
	}
	if !errors.Is(err, context.Canceled) { // The error must be context.Canceled specifically
		t.Errorf("expected context.Canceled; got %v", err) // Report the wrong error type
	}
}

// ── fakeNewChannel ────────────────────────────────────────────────────────────
// fakeNewChannel implements gossh.NewChannel to allow direct testing of handleChannel
// without a real SSH connection. Test code controls the channel type and Accept behaviour.
type fakeNewChannel struct {
	channelType string // Returned by ChannelType() -- set to "session" or something else
	acceptErr   error  // Returned by Accept(); nil means success
	rejected    bool   // Set to true by Reject() so tests can verify rejection occurred
}

// Accept implements gossh.NewChannel.Accept.
func (f *fakeNewChannel) Accept() (gossh.Channel, <-chan *gossh.Request, error) {
	if f.acceptErr != nil { // Return the configured error when set
		return nil, nil, f.acceptErr // nil channel and requests are safe -- handleChannel returns immediately
	}
	requests := make(chan *gossh.Request) // Create a request channel that will be closed immediately
	close(requests)                       // Closing forces processChannelRequests to exit its range loop
	return nil, requests, nil             // nil gossh.Channel is safe -- handleChannel defers Close on it but nil interface panics
}

// Reject implements gossh.NewChannel.Reject.
func (f *fakeNewChannel) Reject(reason gossh.RejectionReason, msg string) error {
	f.rejected = true // Record that Reject was called so tests can assert on it
	return nil        // Always succeed -- no real SSH wire to write to
}

// ChannelType implements gossh.NewChannel.ChannelType.
func (f *fakeNewChannel) ChannelType() string { return f.channelType } // Return configured type

// ExtraData implements gossh.NewChannel.ExtraData.
func (f *fakeNewChannel) ExtraData() []byte { return nil } // No extra data needed for these tests

// ── handleChannel tests ───────────────────────────────────────────────────────

// TestHandleChannel_RejectsNonSession verifies that handleChannel rejects incoming channels
// that are NOT of type "session" (e.g. direct-tcpip). This covers the rejection branch.
func TestHandleChannel_RejectsNonSession(t *testing.T) {
	t.Parallel() // Independent of all other tests

	s := &Server{}                                     // Minimal server -- handleChannel only needs s.processChannelRequests
	ch := &fakeNewChannel{channelType: "direct-tcpip"} // Non-session type triggers the rejection path
	s.handleChannel(ch, "test-session-id")             // Must reject without panic
	if !ch.rejected {                                  // Verify Reject() was called
		t.Error("expected non-session channel to be rejected via Reject()") // Report missing rejection
	}
}

// TestHandleChannel_AcceptError verifies that handleChannel gracefully handles the case where
// Accept() returns an error, logging the error and returning without panicking.
// This covers the "channel accept failed" error path.
func TestHandleChannel_AcceptError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	s := &Server{}         // Minimal server
	ch := &fakeNewChannel{ // Session channel type so it passes the ChannelType check
		channelType: "session",                                    // "session" is the only accepted type
		acceptErr:   fmt.Errorf("accept failed: simulated error"), // Accept() will return this error
	}
	s.handleChannel(ch, "test-session-id") // Must not panic; logs slog.Error and returns
}

// ── processChannelRequests tests ─────────────────────────────────────────────

// TestProcessChannelRequests_WindowChange verifies that processChannelRequests handles
// "window-change" requests without panicking. This covers the "window-change" case branch.
// WantReply is false so no Reply() call is made (which would require a real SSH connection).
func TestProcessChannelRequests_WindowChange(t *testing.T) {
	t.Parallel() // Independent of all other tests

	s := &Server{}                                                      // Minimal server
	requests := make(chan *gossh.Request, 1)                            // Buffered so the send doesn't block
	requests <- &gossh.Request{Type: "window-change", WantReply: false} // Send a window-change resize request
	close(requests)                                                     // Close so the range loop exits after processing

	// nil gossh.Channel is safe here -- processChannelRequests only uses ch in exec/shell case
	s.processChannelRequests(nil, requests, "test-session-id") // Must not panic
}

// TestProcessChannelRequests_UnknownType verifies that processChannelRequests handles
// unrecognised request types by reaching the default case, logging, and continuing.
// This covers the default switch branch (e.g. keepalive requests from OpenSSH).
func TestProcessChannelRequests_UnknownType(t *testing.T) {
	t.Parallel() // Independent of all other tests

	s := &Server{}                                                                     // Minimal server
	requests := make(chan *gossh.Request, 2)                                           // Buffered for two requests
	requests <- &gossh.Request{Type: "keepalive@openssh.com", WantReply: false}        // Standard keepalive -- unknown to this server
	requests <- &gossh.Request{Type: "no-more-sessions@openssh.com", WantReply: false} // Multiplexing hint -- also unknown
	close(requests)                                                                    // Close so the range loop exits after both requests

	// nil gossh.Channel is safe here -- only exec/shell case uses ch
	s.processChannelRequests(nil, requests, "test-session-id") // Must not panic and must log both rejections
}
