// Package ssh -- additional unit tests for newSessionID and buildServerConfig.
package ssh

import (
	"regexp"  // for regexp.MustCompile -- validate session ID format
	"testing" // for testing.T -- standard Go test runner

	"github.com/jmorrison-juniper/misthelper-go/internal/api"  // for api.Config -- builds test server config
	"github.com/jmorrison-juniper/misthelper-go/internal/menu" // for menu.NewRegistry -- stub registry
)

// TestNewSessionID_NonEmpty verifies that newSessionID always returns a non-empty string.
// An empty session ID would cause the session directory to be non-unique, leading to collisions.
func TestNewSessionID_NonEmpty(t *testing.T) {
	t.Parallel()             // Safe to run concurrently
	id := newSessionID()     // Call the function under test
	if id == "" {            // Session ID must never be empty
		t.Error("newSessionID returned empty string") // Report empty ID as a bug
	}
}

// TestNewSessionID_Unique verifies that two consecutive calls return different session IDs.
// Collision-free IDs are required for per-session directory isolation.
func TestNewSessionID_Unique(t *testing.T) {
	t.Parallel()            // Safe to run concurrently
	id1 := newSessionID()   // First session ID
	id2 := newSessionID()   // Second session ID -- random suffix makes collisions astronomically unlikely
	if id1 == id2 {         // Two IDs must be distinct with overwhelming probability
		t.Errorf("newSessionID returned same value twice: %q", id1) // Report the duplicate (would indicate broken rand)
	}
}

// TestNewSessionID_Format verifies that the session ID matches the expected timestamp_hex format.
// The format is "YYYYMMDD_HHMMSS_XXXX" where XXXX is 4 hex characters.
// Consistent format is required so per-session directories have predictable, sortable names.
func TestNewSessionID_Format(t *testing.T) {
	t.Parallel()                                                                 // Safe to run concurrently
	pattern := regexp.MustCompile(`^\d{8}_\d{6}_[0-9a-f]{4}$`)                 // YYYYMMDD_HHMMSS_4hexchars
	id := newSessionID()                                                         // Generate an ID to validate
	if !pattern.MatchString(id) {                                               // Must match the documented format
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
	config := server.buildServerConfig()           // Build the config under test
	if config.PublicKeyCallback != nil {           // Public key auth must be explicitly disabled
		t.Error("PublicKeyCallback must be nil") // Report if it's accidentally set
	}
	if config.NoClientAuth { // Client auth must be required (not bypassed)
		t.Error("NoClientAuth must be false -- authentication is required") // Report the security misconfiguration
	}
}

// TestNewSessionID_MultipleFormatsValid verifies that multiple generated IDs all match the expected format.
// This strengthens confidence that the format is always valid, not just for a single lucky call.
func TestNewSessionID_MultipleFormatsValid(t *testing.T) {
	t.Parallel()                                                    // Safe to run concurrently
	pattern := regexp.MustCompile(`^\d{8}_\d{6}_[0-9a-f]{4}$`)    // Same pattern as in TestNewSessionID_Format
	for i := 0; i < 10; i++ {                                      // Generate 10 IDs to increase statistical confidence
		id := newSessionID()                                         // Generate one session ID
		if !pattern.MatchString(id) {                               // Each ID must match the format
			t.Errorf("iteration %d: ID %q does not match pattern", i, id) // Report the malformed ID with iteration number
		}
	}
}
