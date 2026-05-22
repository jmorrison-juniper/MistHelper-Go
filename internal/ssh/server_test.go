// Package ssh contains tests for the SSH server components.
package ssh

import (
	"net"           // for net.Addr -- used in the mock ConnMetadata implementation
	"os"            // for os.Stat -- checks key file existence after LoadOrCreateHostKey
	"path/filepath" // for filepath.Join -- constructs the expected key file path
	"testing"       // for testing.T and testing.TempDir -- standard Go test utilities

	gossh "golang.org/x/crypto/ssh" // aliased to gossh to avoid conflict with the package name

	"github.com/jmorrison-juniper/misthelper-go/internal/api"  // for api.Config -- builds test server config
	"github.com/jmorrison-juniper/misthelper-go/internal/menu" // for menu.NewRegistry -- stub registry
)

// mockConnMeta implements gossh.ConnMetadata for use in unit tests.
// All fields default to zero values; tests only need User() for password validation.
type mockConnMeta struct {
	user   string   // SSH username presented by the client
	remote net.Addr // Remote address (unused in password validation but required by interface)
}

// User returns the username presented during authentication.
func (m mockConnMeta) User() string { return m.user } // Return the configured test username

// SessionID returns an empty byte slice (not used in password validation tests).
func (m mockConnMeta) SessionID() []byte { return nil } // Not needed for password callback testing

// ClientVersion returns an empty byte slice (not used in password validation tests).
func (m mockConnMeta) ClientVersion() []byte { return nil } // Not needed for password callback testing

// ServerVersion returns an empty byte slice (not used in password validation tests).
func (m mockConnMeta) ServerVersion() []byte { return nil } // Not needed for password callback testing

// RemoteAddr returns a nil net.Addr (not used in password validation tests).
func (m mockConnMeta) RemoteAddr() net.Addr { return m.remote } // Not needed for password callback testing

// LocalAddr returns a nil net.Addr (not used in password validation tests).
func (m mockConnMeta) LocalAddr() net.Addr { return nil } // Not needed for password callback testing

// TestLoadOrCreateHostKey_CreatesKey verifies that LoadOrCreateHostKey writes a key file and
// returns a non-nil signer when called on a directory that contains no key.
func TestLoadOrCreateHostKey_CreatesKey(t *testing.T) {
	t.Parallel() // Run independently of other tests to avoid shared-state interference

	dir := t.TempDir()                      // Create a fresh temporary directory for this test
	signer, err := LoadOrCreateHostKey(dir) // Call the function under test on the empty directory
	if err != nil {                         // Any error here means the function failed unexpectedly
		t.Fatalf("LoadOrCreateHostKey returned unexpected error: %v", err) // Fail fast with the error detail
	}
	if signer == nil { // The returned signer must be ready to use
		t.Fatal("expected non-nil signer, got nil") // Nil signer means the key was not parsed
	}

	keyPath := filepath.Join(dir, keyFileName)  // Build the expected key file path
	if _, err := os.Stat(keyPath); err != nil { // The key file must exist on disk after the call
		t.Fatalf("key file not found at %s: %v", keyPath, err) // Fail with the path if the file is missing
	}
}

// TestLoadOrCreateHostKey_LoadsExisting verifies that calling LoadOrCreateHostKey twice on the
// same directory returns a signer with an identical public key fingerprint both times.
func TestLoadOrCreateHostKey_LoadsExisting(t *testing.T) {
	t.Parallel() // Run independently of other tests

	dir := t.TempDir() // Fresh directory so this test is isolated from other test runs

	signer1, err := LoadOrCreateHostKey(dir) // First call -- generates and saves the key
	if err != nil {                          // First call must succeed before we can test the second
		t.Fatalf("first LoadOrCreateHostKey failed: %v", err) // Bail early if generation failed
	}

	signer2, err := LoadOrCreateHostKey(dir) // Second call -- must load the existing key, not generate a new one
	if err != nil {                          // Loading an existing key must not produce an error
		t.Fatalf("second LoadOrCreateHostKey failed: %v", err) // Bail if the load step failed
	}

	fp1 := gossh.FingerprintSHA256(signer1.PublicKey()) // Compute fingerprint of the first signer's public key
	fp2 := gossh.FingerprintSHA256(signer2.PublicKey()) // Compute fingerprint of the second signer's public key
	if fp1 != fp2 {                                     // Both calls must yield the same key -- different fingerprints mean a new key was generated
		t.Errorf("fingerprint mismatch: first=%s second=%s", fp1, fp2) // Report both fingerprints for debugging
	}
}

// TestNewServer_NotNil verifies that NewServer returns a non-nil *Server for valid inputs.
func TestNewServer_NotNil(t *testing.T) {
	t.Parallel() // Run independently

	dir := t.TempDir()                      // Need a real host key for NewServer (signer must be non-nil)
	signer, err := LoadOrCreateHostKey(dir) // Generate a throw-away key for this test
	if err != nil {                         // Key generation must succeed to proceed
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail if we cannot generate the test key
	}

	cfg := api.Config{ // Minimal config -- only SSHUser and SSHPassword are checked in this test
		SSHPort:     2200,       // Standard MistHelper SSH port
		SSHUser:     "testuser", // Arbitrary username for the test server
		SSHPassword: "testpass", // Arbitrary password for the test server
	}
	registry := menu.NewRegistry()                  // Empty registry is valid -- no menu entries needed for this test
	server := NewServer(cfg, signer, registry, nil) // nil writer is acceptable when no menu handlers run
	if server == nil {                              // NewServer must never return nil
		t.Fatal("expected non-nil *Server, got nil") // Fail fast if the constructor returned nil
	}
}

// TestServer_PasswordAuth verifies the password callback accepts correct credentials and rejects incorrect ones.
// This tests the validatePassword method directly rather than via a full TCP connection.
func TestServer_PasswordAuth(t *testing.T) {
	t.Parallel() // Run independently

	const validUser = "admin"   // Username that the server is configured to accept
	const validPass = "s3cret!" // Password that the server is configured to accept

	dir := t.TempDir()                      // Temp dir for host key generation
	signer, err := LoadOrCreateHostKey(dir) // Generate a throw-away host key
	if err != nil {                         // Key generation must succeed before testing auth
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail if key generation fails
	}

	cfg := api.Config{ // Server config with the expected credentials
		SSHUser:     validUser, // The only username the server accepts
		SSHPassword: validPass, // The only password the server accepts
	}
	server := NewServer(cfg, signer, menu.NewRegistry(), nil) // Build server with the test credentials

	tests := []struct { // Table-driven sub-tests for all credential combinations
		name   string // Descriptive test name shown on failure
		user   string // Username to present to the callback
		pass   string // Password to present to the callback
		wantOK bool   // Whether the callback should return non-nil Permissions
	}{
		{"correct credentials", validUser, validPass, true}, // Should be accepted
		{"wrong password", validUser, "wrongpass", false},   // Should be rejected
		{"wrong username", "nobody", validPass, false},      // Should be rejected
		{"both wrong", "hacker", "letmein", false},          // Should be rejected
		{"empty username", "", validPass, false},            // Should be rejected
		{"empty password", validUser, "", false},            // Should be rejected
	}

	for _, tc := range tests { // Iterate through each credential combination
		t.Run(tc.name, func(t *testing.T) { // Run each combination as a named sub-test
			t.Parallel()                                                 // Sub-tests can run concurrently
			meta := mockConnMeta{user: tc.user}                          // Wrap the test username in the mock ConnMetadata
			perms, err := server.validatePassword(meta, []byte(tc.pass)) // Call the password callback directly
			if tc.wantOK && (perms == nil || err != nil) {               // Expected acceptance but got rejection
				t.Errorf("expected acceptance but got perms=%v err=%v", perms, err) // Report both values for debugging
			}
			if !tc.wantOK && (perms != nil || err == nil) { // Expected rejection but got acceptance
				t.Errorf("expected rejection but got perms=%v err=%v", perms, err) // Report both values for debugging
			}
		})
	}
}
