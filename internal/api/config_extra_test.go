// Package api -- additional unit tests for config helpers not covered by config_test.go.
package api

import (
	"os"      // for os.Unsetenv -- clears env vars between test cases
	"testing" // for testing.T -- standard Go test runner
)

// TestEnvStr_ReturnsDefault verifies that envStr returns the default when the env var is not set.
func TestEnvStr_ReturnsDefault(t *testing.T) {
	t.Parallel()                              // Safe to run concurrently with other tests
	_ = os.Unsetenv("_MISTHELPER_TEST_ENVSTR_A") // Ensure the variable is unset for a clean test
	result := envStr("_MISTHELPER_TEST_ENVSTR_A", "my-default") // Call with absent variable
	if result != "my-default" {               // Must return the compiled default
		t.Errorf("expected %q, got %q", "my-default", result) // Report the mismatch for debugging
	}
}

// TestEnvStr_ReturnsEnvValue verifies that envStr returns the env var value when set.
func TestEnvStr_ReturnsEnvValue(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("_MISTHELPER_TEST_ENVSTR_B", "actual-value") // Set the variable for this test
	result := envStr("_MISTHELPER_TEST_ENVSTR_B", "default") // Call with variable present
	if result != "actual-value" {             // Must return the environment value, not the default
		t.Errorf("expected %q, got %q", "actual-value", result) // Report the mismatch
	}
}

// TestEnvStr_EmptyEnvVarFallsBack verifies that an explicitly empty env var returns the default.
func TestEnvStr_EmptyEnvVarFallsBack(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("_MISTHELPER_TEST_ENVSTR_C", "")              // Set to empty string (treated as absent)
	result := envStr("_MISTHELPER_TEST_ENVSTR_C", "fallback") // Call with empty variable
	if result != "fallback" {                               // Empty string is treated as absent
		t.Errorf("expected %q, got %q", "fallback", result) // Report the mismatch
	}
}

// TestEnvInt_ReturnsDefault verifies that envInt returns the default when the env var is not set.
func TestEnvInt_ReturnsDefault(t *testing.T) {
	t.Parallel()                             // Safe to run concurrently
	_ = os.Unsetenv("_MISTHELPER_TEST_ENVINT_A") // Ensure the variable is absent
	result := envInt("_MISTHELPER_TEST_ENVINT_A", 42) // Call with absent variable
	if result != 42 {                        // Must return the default integer
		t.Errorf("expected 42, got %d", result) // Report actual value for debugging
	}
}

// TestEnvInt_ReturnsEnvValue verifies that envInt parses and returns a valid integer env var.
func TestEnvInt_ReturnsEnvValue(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("_MISTHELPER_TEST_ENVINT_B", "99")    // Set a valid integer env var
	result := envInt("_MISTHELPER_TEST_ENVINT_B", 0) // Call with a valid integer variable
	if result != 99 {                               // Must return the parsed integer
		t.Errorf("expected 99, got %d", result) // Report actual value for debugging
	}
}

// TestEnvInt_InvalidValueFallsBack verifies that envInt returns the default for a non-integer value.
func TestEnvInt_InvalidValueFallsBack(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("_MISTHELPER_TEST_ENVINT_C", "not-a-number")    // Set a non-numeric env var
	result := envInt("_MISTHELPER_TEST_ENVINT_C", 7)         // Call with a malformed variable
	if result != 7 {                                          // Must fall back to the default
		t.Errorf("expected default 7, got %d", result) // Report actual value for debugging
	}
}

// TestEnvInt_EmptyValueFallsBack verifies that an empty env var returns the default.
func TestEnvInt_EmptyValueFallsBack(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("_MISTHELPER_TEST_ENVINT_D", "")        // Set an empty env var
	result := envInt("_MISTHELPER_TEST_ENVINT_D", 5) // Call with an empty variable
	if result != 5 {                                  // Empty string is treated as absent
		t.Errorf("expected default 5, got %d", result) // Report actual value for debugging
	}
}

// TestResolveOutputFormat_FlagWins verifies that a non-empty CLI flag takes precedence over env var.
func TestResolveOutputFormat_FlagWins(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("OUTPUT_FORMAT", "sqlite")                // Set env var -- should be overridden by flag
	result := resolveOutputFormat("csv")               // Flag value "csv" should win over env var
	if result != "csv" {                               // CLI flag must win over OUTPUT_FORMAT env var
		t.Errorf("expected %q (flag wins), got %q", "csv", result) // Report actual value
	}
}

// TestResolveOutputFormat_EnvVarFallback verifies that env var is used when no CLI flag is given.
func TestResolveOutputFormat_EnvVarFallback(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("OUTPUT_FORMAT", "sqlite")                // Set env var as secondary source
	result := resolveOutputFormat("")                  // No CLI flag -- should use env var
	if result != "sqlite" {                            // Env var must win when no flag is given
		t.Errorf("expected %q (env var), got %q", "sqlite", result) // Report actual value
	}
}

// TestResolveOutputFormat_DefaultCSV verifies that "csv" is returned when no flag and no env var are set.
func TestResolveOutputFormat_DefaultCSV(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("OUTPUT_FORMAT", "")           // Explicitly clear the env var
	result := resolveOutputFormat("")       // No flag, no env var -- should default to "csv"
	if result != "csv" {                    // "csv" is the compiled default output format
		t.Errorf("expected %q (default), got %q", "csv", result) // Report actual value
	}
}

// TestLoadConfig_EmptyToken verifies that LoadConfig returns an error when MIST_API_TOKEN is set to empty string.
func TestLoadConfig_EmptyToken(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("MIST_API_TOKEN", "")           // Set token to empty string -- treated as absent
	t.Setenv("MIST_ORG_ID", "some-org-id")  // Provide org ID so token check fires first
	_, err := LoadConfig("")                 // Attempt to load with empty token
	if err == nil {                          // Must return an error when token is empty
		t.Error("expected error for empty MIST_API_TOKEN, got nil") // Fail if no error returned
	}
}

// TestLoadConfig_EmptyOrgID verifies that LoadConfig returns an error when MIST_ORG_ID is set to empty string.
func TestLoadConfig_EmptyOrgID(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("MIST_API_TOKEN", "valid-token")     // Provide token so org ID check fires
	t.Setenv("MIST_ORG_ID", "")                  // Set org ID to empty string -- treated as absent
	_, err := LoadConfig("")                      // Attempt to load with empty org ID
	if err == nil {                               // Must return an error when org ID is empty
		t.Error("expected error for empty MIST_ORG_ID, got nil") // Fail if no error returned
	}
}

// TestLoadConfig_SuccessWithCLIFormat verifies that LoadConfig returns a Config with the CLI format applied.
func TestLoadConfig_SuccessWithCLIFormat(t *testing.T) {
	// NOT parallel -- t.Setenv modifies process-global env, incompatible with t.Parallel
	t.Setenv("MIST_API_TOKEN", "test-token-xyz")  // Set required token
	t.Setenv("MIST_ORG_ID", "test-org-abc")       // Set required org ID
	cfg, err := LoadConfig("sqlite")              // CLI flag "sqlite" overrides default
	if err != nil {                               // LoadConfig must succeed with valid inputs
		t.Fatalf("unexpected error: %v", err)    // Fail fast with error detail
	}
	if cfg.APIToken != "test-token-xyz" {          // Token must be copied from env var
		t.Errorf("expected APIToken=%q, got %q", "test-token-xyz", cfg.APIToken) // Report mismatch
	}
	if cfg.OrgID != "test-org-abc" {              // OrgID must be copied from env var
		t.Errorf("expected OrgID=%q, got %q", "test-org-abc", cfg.OrgID) // Report mismatch
	}
	if cfg.OutputFormat != "sqlite" {             // CLI flag format must override env var
		t.Errorf("expected OutputFormat=%q, got %q", "sqlite", cfg.OutputFormat) // Report mismatch
	}
}
