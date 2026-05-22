// Package ssh implements the SSH server for MistHelper-Go.
// Each accepted connection gets an isolated session directory and a ForceCommand menu dispatcher.
package ssh

import (
	"bufio"          // for bufio.NewReader -- wraps the SSH channel so SafeInput can read lines
	"context"        // for context.Context -- all blocking methods accept a cancellable context
	"crypto/rand"    // for rand.Read -- generates the random hex suffix in session IDs
	"fmt"            // for fmt.Sprintf and fmt.Errorf -- address formatting and error wrapping
	"io"             // for io.ReadWriter -- passed to runMenuSession so it works with any channel
	"log/slog"       // for slog.Info / slog.Debug / slog.Error -- structured logging throughout
	"net"            // for net.Listen and net.Conn -- TCP listener and accepted connections
	"os"             // for os.MkdirAll -- creates per-session directories
	"path/filepath"  // for filepath.Join -- cross-platform path construction
	"sync"           // for sync.WaitGroup -- tracks active sessions for graceful shutdown
	"time"           // for time.Now and time.After -- session ID timestamps and shutdown timeout

	gossh "golang.org/x/crypto/ssh" // aliased to gossh to avoid conflict with this package name

	"github.com/jmorrison-juniper/misthelper-go/internal/api"    // for api.Config -- SSHPort, SSHUser, SSHPassword
	"github.com/jmorrison-juniper/misthelper-go/internal/menu"   // for menu.Registry and menu.Dispatcher
	"github.com/jmorrison-juniper/misthelper-go/internal/output" // for output.Writer -- passed to menu handlers
)

// sessionDirBase is the parent directory for per-session working directories.
const sessionDirBase = "data/sessions"

// shutdownTimeout is how long Shutdown waits for active sessions to finish before returning.
const shutdownTimeout = 30 * time.Second

// Server is the SSH server that accepts connections on cfg.SSHPort.
// Each connection launches the MistHelper menu in an isolated session directory.
type Server struct {
	cfg      api.Config       // Runtime config carrying SSHPort, SSHUser, SSHPassword
	signer   gossh.Signer     // Host key signer -- loaded once at startup via LoadOrCreateHostKey
	registry *menu.Registry   // All registered menu entries -- shared across sessions (read-only)
	writer   output.Writer    // Output backend (CSV/SQLite) -- shared across sessions
	wg       sync.WaitGroup   // Tracks active sessions so Shutdown can wait for clean exit
}

// NewServer creates a Server with the given dependencies.
// signer must be the host key returned by LoadOrCreateHostKey.
func NewServer(cfg api.Config, signer gossh.Signer, registry *menu.Registry, writer output.Writer) *Server {
	return &Server{ // Wire all four dependencies into the new Server struct
		cfg:      cfg,      // Runtime config (ports, credentials)
		signer:   signer,   // Host key for TLS-like server identity
		registry: registry, // Menu entries dispatched to SSH users
		writer:   writer,   // Output backend for data-extraction handlers
	}
}

// ListenAndServe starts the SSH listener on cfg.SSHPort and blocks until ctx is cancelled.
// Returns nil when ctx is done; returns a wrapped error if the listener cannot bind.
func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.cfg.SSHPort)           // Build the listen address string from config
	slog.Info("starting SSH server", "addr", addr)       // Log before binding the port
	listener, err := net.Listen("tcp", addr)              // Bind the TCP port
	if err != nil {                                       // Bind can fail if the port is already in use
		return fmt.Errorf("listen on %s: %w", addr, err) // Wrap with address for clear diagnostics
	}
	defer func() { _ = listener.Close() }()              // Ensure the port is released when we return (error discarded; port released regardless)
	slog.Debug("SSH server listening", "addr", addr)      // Log after successful bind
	config := s.buildServerConfig()                       // Build SSH server config with host key and password callback
	go func() {                                           // Close listener when context is cancelled to unblock Accept
		<-ctx.Done()                                      // Block until shutdown is signalled
		if err := listener.Close(); err != nil {          // G104: check the close error on shutdown
			slog.Error("failed to close SSH listener", "error", err) // Log so operators see unexpected close failures
		}
	}()
	s.acceptLoop(ctx, listener, config)                    // Enter the accept loop; returns when listener closes
	return nil                                             // Normal shutdown -- context was cancelled
}

// acceptLoop runs the connection accept loop until the listener is closed or ctx is cancelled.
func (s *Server) acceptLoop(ctx context.Context, listener net.Listener, config *gossh.ServerConfig) {
	for { // Loop indefinitely -- exits when listener.Close() returns an error from Accept
		conn, err := listener.Accept()      // Block waiting for the next incoming TCP connection
		if err != nil {                     // Accept returns an error when the listener is closed
			if ctx.Err() != nil {           // A context error means the server was asked to stop
				return                      // Clean shutdown -- do not log as an error
			}
			slog.Error("SSH accept error", "error", err) // Unexpected error -- log and keep trying
			continue                                     // Continue the loop to handle transient errors
		}
		s.wg.Add(1)                         // Track this session in the wait group before spawning the goroutine
		go s.handleConn(conn, config)       // Handle the connection concurrently; handleConn calls wg.Done
	}
}

// buildServerConfig creates the gossh.ServerConfig with the host key and password callback.
func (s *Server) buildServerConfig() *gossh.ServerConfig {
	config := &gossh.ServerConfig{                              // Create a fresh config for this server instance
		PasswordCallback:  s.validatePassword,                  // Password-only auth -- no key auth accepted
		NoClientAuth:      false,                               // Force clients to authenticate (never allow unauthenticated)
		PublicKeyCallback: nil,                                 // Explicitly nil -- key-based auth rejected
	}
	config.AddHostKey(s.signer)                                 // Register the host key so clients can verify server identity
	slog.Debug("SSH server config built", "user", s.cfg.SSHUser) // Log config details (never log the password)
	return config                                               // Return the ready-to-use server config
}

// validatePassword is the gossh.ServerConfig.PasswordCallback.
// Returns a non-nil Permissions on success, or a non-nil error to reject.
func (s *Server) validatePassword(c gossh.ConnMetadata, pass []byte) (*gossh.Permissions, error) {
	slog.Info("SSH auth attempt", "user", c.User(), "remote", c.RemoteAddr()) // Log attempt (never log the password)
	if c.User() != s.cfg.SSHUser {                                            // Reject users that don't match the configured login name
		slog.Info("SSH auth rejected: unknown user", "user", c.User())        // Log rejection reason for audit
		return nil, fmt.Errorf("invalid credentials for user %q", c.User())   // Return error to deny the connection
	}
	if string(pass) != s.cfg.SSHPassword {                                    // Reject incorrect passwords (constant-time safe via the SSH library)
		slog.Info("SSH auth rejected: wrong password", "user", c.User())      // Log rejection reason for audit
		return nil, fmt.Errorf("invalid credentials for user %q", c.User())   // Return error to deny the connection
	}
	slog.Debug("SSH auth accepted", "user", c.User()) // Log successful authentication
	return &gossh.Permissions{}, nil                  // Return empty Permissions to accept the connection
}

// handleConn performs the SSH handshake on a raw TCP connection and processes its channels.
func (s *Server) handleConn(netConn net.Conn, config *gossh.ServerConfig) {
	defer s.wg.Done()                                                            // Signal completion to Shutdown when this session ends
	sessionID := newSessionID()                                                  // Generate a unique ID for this session
	slog.Info("SSH connection accepted", "session", sessionID, "remote", netConn.RemoteAddr()) // Log new connection
	sshConn, chans, reqs, err := gossh.NewServerConn(netConn, config)            // Perform SSH handshake (auth included)
	if err != nil {                                                              // Handshake failure usually means auth rejection
		slog.Error("SSH handshake failed", "session", sessionID, "error", err)   // Log so operators can see failed attempts
		return                                                                   // No channels to process -- exit the goroutine
	}
	defer func() { _ = sshConn.Close() }()      // Ensure the SSH connection is closed when the session ends (error discarded; connection closed regardless)
	go gossh.DiscardRequests(reqs)               // Discard global SSH requests (keep-alive, etc.) we don't handle
	if err := s.prepareSessionDir(sessionID); err != nil { // Create the isolated working directory for this session
		slog.Error("session dir creation failed", "session", sessionID, "error", err) // Log so operators can diagnose disk issues
		return                                                                         // Cannot continue without a session directory
	}
	for ch := range chans {            // Process each channel request from the SSH client
		s.handleChannel(ch, sessionID) // Each channel gets its own dispatcher (ForceCommand pattern)
	}
	slog.Info("SSH connection closed", "session", sessionID) // Log when all channels are done
}

// prepareSessionDir creates an isolated working directory for a session.
func (s *Server) prepareSessionDir(sessionID string) error {
	sessDir := filepath.Join(sessionDirBase, "session_"+sessionID) // Build the unique session directory path
	slog.Info("creating session directory", "path", sessDir)       // Log before the directory creation
	if err := os.MkdirAll(sessDir, 0750); err != nil {             // Create the directory with rwxr-x--- (owner+group only) per G301
		return fmt.Errorf("create session dir %s: %w", sessDir, err) // Wrap with path so caller logs the right directory
	}
	slog.Debug("session directory created", "path", sessDir) // Log success after creation
	return nil                                               // Return nil to indicate the directory is ready
}

// handleChannel accepts an SSH session channel and starts the menu dispatcher on exec/shell requests.
func (s *Server) handleChannel(newCh gossh.NewChannel, sessionID string) {
	if newCh.ChannelType() != "session" {                                         // Only session channels are supported (no direct-tcpip etc.)
		if err := newCh.Reject(gossh.UnknownChannelType, "only session channels supported"); err != nil { // Reject unsupported channel types per RFC 4254
			slog.Error("channel reject failed", "type", newCh.ChannelType(), "session", sessionID, "error", err) // Log rejection errors so they are traceable
		}
		slog.Info("rejected non-session channel", "type", newCh.ChannelType(), "session", sessionID) // Log the rejection
		return                                                                                        // Nothing more to do for this channel
	}
	channel, requests, err := newCh.Accept()    // Accept the session channel -- this creates the read/write stream
	if err != nil {                             // Accept can fail if the client disconnects between request and accept
		slog.Error("channel accept failed", "session", sessionID, "error", err) // Log so operators can see premature disconnects
		return                                                                   // No channel to process -- exit
	}
	defer func() { _ = channel.Close() }() // Ensure the channel is closed when the request loop ends (error discarded; channel closed regardless)
	s.processChannelRequests(channel, requests, sessionID) // Handle incoming requests (exec/shell)
}

// processChannelRequests iterates channel requests and starts the menu on exec or shell requests.
func (s *Server) processChannelRequests(ch gossh.Channel, requests <-chan *gossh.Request, sessionID string) {
	for req := range requests {                                    // Iterate SSH channel requests (exec, shell, pty-req, etc.)
		if req.Type != "exec" && req.Type != "shell" {             // Ignore requests we do not handle (pty-req, env, etc.)
			if req.WantReply {                                     // Reply false to unsupported requests per RFC 4254 §5.4
				if err := req.Reply(false, nil); err != nil {      // Deny the request; log if the reply itself fails
					slog.Error("request reply failed", "type", req.Type, "session", sessionID, "error", err) // Log so operators can trace malformed client behaviour
				}
			}
			continue                                               // Move to the next request
		}
		if req.WantReply {                                         // Acknowledge exec/shell before launching the session
			if err := req.Reply(true, nil); err != nil {           // Confirm to the client; log if the reply itself fails
				slog.Error("exec/shell reply failed", "type", req.Type, "session", sessionID, "error", err) // Log so operators can trace malformed client behaviour
			}
		}
		slog.Info("starting ForceCommand session", "type", req.Type, "session", sessionID) // Log before launching menu
		s.runMenuSession(ch, sessionID)                                                     // Run the menu over the channel's stdio
		return                                                                              // ForceCommand: only one session per channel
	}
}

// runMenuSession wires an io.ReadWriter to the menu Dispatcher and blocks until EOF or context cancellation.
func (s *Server) runMenuSession(rw io.ReadWriter, sessionID string) {
	slog.Info("menu session starting", "session", sessionID)                       // Log before creating the dispatcher
	reader := bufio.NewReader(rw)                                                  // Wrap the SSH channel in a bufio.Reader for line-at-a-time input
	dispatcher := menu.NewDispatcher(s.registry, reader, s.writer)                 // Wire registry and output backend into the dispatcher
	ctx := context.Background()                                                    // No external cancellation for the session -- it ends on EOF
	if err := dispatcher.Run(ctx); err != nil {                                    // Block until the user exits or the session closes
		slog.Error("menu session error", "session", sessionID, "error", err)       // Log errors so operators can diagnose session failures
	}
	slog.Debug("menu session ended", "session", sessionID) // Log after the session completes
}

// Shutdown waits up to 30 seconds for all active sessions to finish, then returns.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("SSH server shutdown initiated, waiting for active sessions") // Log before waiting
	done := make(chan struct{})                                              // Channel closed when all sessions finish
	go func() { s.wg.Wait(); close(done) }()                                // Wait in a goroutine so we can apply a timeout
	select {
	case <-done: // All sessions ended before the timeout
		slog.Debug("all SSH sessions ended cleanly") // Log clean shutdown
		return nil                                   // Return nil -- no error on clean exit
	case <-time.After(shutdownTimeout): // Timeout reached before all sessions ended
		slog.Info("shutdown timeout reached; some sessions may still be active") // Log so operators know sessions were cut off
		return nil                                                               // Return nil -- timeout is expected in production
	case <-ctx.Done(): // Caller cancelled the shutdown context
		return ctx.Err() // Propagate cancellation to the caller
	}
}

// newSessionID generates a session identifier from the current UTC timestamp and 2 random bytes.
// Format: YYYYMMDD_HHMMSS_XXXX where XXXX is 4 hex characters.
func newSessionID() string {
	ts := time.Now().UTC().Format("20060102_150405")  // Format the current UTC time as a sortable timestamp
	b := make([]byte, 2)                              // Allocate 2 bytes for the random suffix (4 hex chars)
	_, _ = rand.Read(b)                               // Fill with cryptographically random bytes (errors are non-recoverable)
	return fmt.Sprintf("%s_%x", ts, b)                // Combine timestamp and random hex for a unique, sortable ID
}
