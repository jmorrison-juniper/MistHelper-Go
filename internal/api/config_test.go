package api

import (
	"os"
	"testing"
)

// TestLoadConfig_ValidEnv verifies that LoadConfig succeeds when both required env vars are present.
func TestLoadConfig_ValidEnv(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Inject valid token for this test
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Inject valid org ID for this test

	cfg, err := LoadConfig("") // No CLI format override
	if err != nil {            // Should succeed with valid env vars
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIToken != "tok-test-123" { // Token must be preserved exactly
		t.Errorf("APIToken = %q, want %q", cfg.APIToken, "tok-test-123")
	}
	if cfg.OrgID != "aaaa-bbbb-cccc" { // OrgID must be preserved exactly
		t.Errorf("OrgID = %q, want %q", cfg.OrgID, "aaaa-bbbb-cccc")
	}
}

// TestLoadConfig_MissingToken verifies that LoadConfig returns a descriptive error when MIST_API_TOKEN is absent.
func TestLoadConfig_MissingToken(t *testing.T) {
	if err := os.Unsetenv("MIST_API_TOKEN"); err != nil { // Ensure token is absent for this test
		t.Fatalf("os.Unsetenv: %v", err) // Fatal -- if we cannot clear the env the test is invalid
	}
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Org is present; token is the missing piece

	_, err := LoadConfig("") // Should fail with a descriptive error
	if err == nil {          // Missing required field must be an error
		t.Fatal("expected error for missing MIST_API_TOKEN, got nil")
	}
}

// TestLoadConfig_MissingOrgID verifies that LoadConfig returns a descriptive error when MIST_ORG_ID is absent.
func TestLoadConfig_MissingOrgID(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Token is present; org is the missing piece
	if err := os.Unsetenv("MIST_ORG_ID"); err != nil { // Ensure org is absent for this test
		t.Fatalf("os.Unsetenv: %v", err) // Fatal -- if we cannot clear the env the test is invalid
	}

	_, err := LoadConfig("") // Should fail with a descriptive error
	if err == nil {          // Missing required field must be an error
		t.Fatal("expected error for missing MIST_ORG_ID, got nil")
	}
}

// TestLoadConfig_CLIFormatOverridesEnv verifies that the --format flag takes precedence over OUTPUT_FORMAT env var.
func TestLoadConfig_CLIFormatOverridesEnv(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Required field
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Required field
	t.Setenv("OUTPUT_FORMAT", "sqlite")        // Env var says sqlite

	cfg, err := LoadConfig("csv") // CLI flag overrides with "csv"
	if err != nil {               // Config load should succeed
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OutputFormat != "csv" { // CLI flag must win over env var
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "csv")
	}
}

// TestLoadConfig_EnvFormatUsedWhenNoFlag verifies that OUTPUT_FORMAT env var is used when no CLI flag is given.
func TestLoadConfig_EnvFormatUsedWhenNoFlag(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Required field
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Required field
	t.Setenv("OUTPUT_FORMAT", "sqlite")        // Env var sets format

	cfg, err := LoadConfig("") // No CLI flag -- env var should be used
	if err != nil {            // Config load should succeed
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OutputFormat != "sqlite" { // Env var must be respected when no flag is given
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "sqlite")
	}
}

// TestLoadConfig_DefaultCSVWhenNoFlag verifies that "csv" is the default when neither env var nor flag is set.
func TestLoadConfig_DefaultCSVWhenNoFlag(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Required field
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Required field
	if err := os.Unsetenv("OUTPUT_FORMAT"); err != nil { // No env var -- should fall back to default
		t.Fatalf("os.Unsetenv: %v", err) // Fatal -- if we cannot clear the env the test result is undefined
	}

	cfg, err := LoadConfig("") // No CLI flag either
	if err != nil {            // Config load should succeed
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OutputFormat != "csv" { // "csv" is the compiled default
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "csv")
	}
}

// TestLoadConfig_RateLimitDefault verifies the 200ms rate limit default when API_RATE_LIMIT_MS is unset.
func TestLoadConfig_RateLimitDefault(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Required field
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Required field
	os.Unsetenv("API_RATE_LIMIT_MS")           // Ensure no override

	cfg, err := LoadConfig("") // Load with defaults
	if err != nil {            // Should succeed
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RateLimitMs != 200 { // Default must be 200ms per spec FR-028
		t.Errorf("RateLimitMs = %d, want 200", cfg.RateLimitMs)
	}
}

// TestLoadConfig_RateLimitFromEnv verifies API_RATE_LIMIT_MS env var is respected.
func TestLoadConfig_RateLimitFromEnv(t *testing.T) {
	t.Setenv("MIST_API_TOKEN", "tok-test-123") // Required field
	t.Setenv("MIST_ORG_ID", "aaaa-bbbb-cccc") // Required field
	t.Setenv("API_RATE_LIMIT_MS", "500")       // Override rate limit

	cfg, err := LoadConfig("") // Load config
	if err != nil {            // Should succeed
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RateLimitMs != 500 { // Env var override must be applied
		t.Errorf("RateLimitMs = %d, want 500", cfg.RateLimitMs)
	}
}
