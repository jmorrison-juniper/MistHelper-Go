// Package api provides the Mist API client, configuration, pagination, and retry logic.
// All interaction with the mistapi-go SDK happens through the Client interface defined here.
package api

import (
	"fmt"     // for descriptive error messages
	"os"      // for reading environment variables
	"strconv" // for parsing integer env vars
)

// Config holds all runtime configuration for the application.
// Passed by value to every package constructor -- no global state, fully testable.
type Config struct {
	APIToken     string // Mist API bearer token -- never logged
	OrgID        string // Target organisation UUID from MIST_ORG_ID
	OutputFormat string // "csv" or "sqlite" -- set by env var or --format flag
	RateLimitMs  int    // Milliseconds to sleep between API pages (default 200)
	SSHPort      int    // TCP port for the SSH server (default 2200)
	SSHUser      string // SSH login username (default "misthelper")
	SSHPassword  string // SSH login password -- never logged
	WebPort      int    // TCP port for the HTTP server (default 8055)
}

// LoadConfig reads environment variables and applies the CLI format override.
// format is the value supplied by the --format flag ("" means no override).
// Returns a descriptive error if MIST_API_TOKEN or MIST_ORG_ID are missing.
func LoadConfig(format string) (Config, error) {
	token := os.Getenv("MIST_API_TOKEN") // Read bearer token -- required, never logged
	orgID := os.Getenv("MIST_ORG_ID")   // Read target org UUID -- required

	if token == "" { // Token is mandatory before any API call can succeed
		return Config{}, fmt.Errorf("MIST_API_TOKEN environment variable is not set -- add it to .env")
	}
	if orgID == "" { // OrgID scopes every API call; missing means we can't target any resource
		return Config{}, fmt.Errorf("MIST_ORG_ID environment variable is not set -- add it to .env")
	}

	outputFmt := resolveOutputFormat(format) // Apply CLI flag precedence over env var

	return Config{
		APIToken:     token,                         // Validated above
		OrgID:        orgID,                         // Validated above
		OutputFormat: outputFmt,                     // CLI flag > env var > default "csv"
		RateLimitMs:  envInt("API_RATE_LIMIT_MS", 200), // Fixed delay between pages (default 200ms)
		SSHPort:      envInt("SSH_PORT", 2200),         // SSH server listen port
		SSHUser:      envStr("SSH_USER", "misthelper"), // SSH login username
		SSHPassword:  envStr("SSH_PASSWORD", "misthelper123!"), // SSH login password -- never logged
		WebPort:      envInt("WEB_PORT", 8055),         // HTTP server listen port
	}, nil
}

// resolveOutputFormat returns the effective output format given the CLI flag value.
// CLI flag (non-empty) takes precedence; otherwise OUTPUT_FORMAT env var; default "csv".
func resolveOutputFormat(flagValue string) string {
	if flagValue != "" { // --format flag was explicitly supplied on the command line
		return flagValue // CLI flag wins over all other sources
	}
	if env := os.Getenv("OUTPUT_FORMAT"); env != "" { // Check env var as secondary source
		return env // Env var wins over compiled default
	}
	return "csv" // Compiled default -- CSV is always safe and requires no extra setup
}

// envInt reads an environment variable and returns its integer value.
// Falls back to defaultVal if the variable is unset or not a valid integer.
func envInt(key string, defaultVal int) int {
	raw := os.Getenv(key) // Read raw string value from environment
	if raw == "" {        // Unset variable -- use the compiled default
		return defaultVal
	}
	v, err := strconv.Atoi(raw) // Parse string to int
	if err != nil {             // Malformed value -- warn and fall back to default
		return defaultVal // Safer than crashing on a misconfigured env var
	}
	return v // Valid integer from environment
}

// envStr reads an environment variable and returns its string value.
// Falls back to defaultVal if the variable is unset or empty.
func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" { // Non-empty env var takes precedence
		return v
	}
	return defaultVal // Use compiled default when variable is absent
}
