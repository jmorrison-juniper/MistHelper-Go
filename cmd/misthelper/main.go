package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// version is the application version string, matching CHANGELOG YY.MM.DD.HH.MM format.
const version = "25.07.14.00.00"

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit") // --version flag
	flag.Parse()                                                          // Parse CLI flags before any action

	if *showVersion { // Handle --version before loading credentials
		slog.Info("MistHelper-Go", "version", version) // Log version to structured output
		os.Exit(0)                                      // Clean exit after printing version
	}

	loadConfig() // Load .env and validate required environment variables

	slog.Info("MistHelper-Go ready", "version", version, "org_id", os.Getenv("MIST_ORG_ID")) // Confirm startup
	slog.Info("No menu operations ported yet -- skeleton only. Run --version to confirm binary works.")
}

// loadConfig loads .env from the working directory and validates required variables.
// The container mounts .env at /app/.env; local dev uses the project root .env.
func loadConfig() {
	slog.Debug("Loading configuration from .env file") // Log intent before action
	if err := godotenv.Load(); err != nil {             // .env is optional; container may inject vars directly
		slog.Debug("No .env file found, relying on environment variables", "error", err)
	}
	slog.Debug("Configuration loaded") // Log completion of load step

	token := os.Getenv("MIST_API_TOKEN")  // Read Mist API bearer token from environment
	orgID := os.Getenv("MIST_ORG_ID")     // Read target org UUID from environment
	if token == "" || orgID == "" {        // Both are required before any API call is possible
		slog.Error("Required environment variables missing -- set MIST_API_TOKEN and MIST_ORG_ID in .env")
		os.Exit(1) // Non-zero exit so container restarts and CI fails fast on misconfiguration
	}
}
