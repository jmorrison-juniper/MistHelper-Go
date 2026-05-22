// Package api tests for the Mist API client wrapper.
package api

import (
	"testing" // testing.T for table-driven tests
)

// TestNewClient_ValidConfig verifies that a properly configured Client is returned
// when the APIToken is non-empty. No real network calls are made; the SDK does not
// validate the token at construction time.
func TestNewClient_ValidConfig(t *testing.T) {
	cfg := Config{             // stub config with required fields populated
		APIToken: "test-token", // non-empty token satisfies the NewClient guard
		OrgID:    "00000000-0000-0000-0000-000000000000", // valid UUID format; avoids parse error in ListSites
	}

	client, err := NewClient(cfg)   // call the constructor under test
	if err != nil {                 // constructor must not error on valid config
		t.Fatalf("NewClient with valid config returned unexpected error: %v", err)
	}
	if client == nil {              // constructor must return a non-nil client pointer
		t.Fatal("NewClient returned nil client for valid config")
	}
}

// TestNewClient_InvalidConfig verifies that an error is returned — not a panic —
// when APIToken is empty. The guard in NewClient must catch this before touching the SDK.
func TestNewClient_InvalidConfig(t *testing.T) {
	cfg := Config{             // stub config missing the required API token
		APIToken: "",          // empty token must trigger an error
		OrgID:    "00000000-0000-0000-0000-000000000000",
	}

	client, err := NewClient(cfg)  // call the constructor under test
	if err == nil {                // constructor must return an error for empty token
		t.Fatal("NewClient with empty APIToken should return an error, but returned nil")
	}
	if client != nil {             // constructor must not return a client when it errors
		t.Fatal("NewClient returned non-nil client despite error")
	}
}
