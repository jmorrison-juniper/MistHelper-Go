package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestVersionFlag verifies the --version flag exits 0 and produces output.
// This is the canonical smoke test: if the binary compiles and runs, CI can proceed.
func TestVersionFlag(t *testing.T) {
	exe, err := exec.LookPath("go") // Find the go toolchain on PATH
	if err != nil {
		t.Fatalf("go toolchain not found: %v", err) // CI must have Go available
	}

	cmd := exec.Command(exe, "run", ".", "-version")                          // Run the current package with --version
	cmd.Dir = "."                                                             // Run from the package directory
	cmd.Env = append(os.Environ(), "MIST_API_TOKEN=test", "MIST_ORG_ID=test") // Provide stub creds so loadConfig does not exit 1

	out, err := cmd.CombinedOutput() // Capture stdout+stderr together
	if err != nil {                  // --version should always exit 0
		t.Fatalf("--version exited with error: %v\noutput: %s", err, out)
	}
	if len(out) == 0 { // Version output must be non-empty to confirm slog wrote something
		t.Fatal("--version produced no output")
	}
}

// TestVersionConstant ensures the version constant is non-empty and plausibly formatted.
// Guards against accidental blanking during future edits.
func TestVersionConstant(t *testing.T) {
	if version == "" { // version must always have a value
		t.Fatal("version constant is empty")
	}
	if len(version) < 8 { // YY.MM.DD.HH.MM is at minimum 8 chars (e.g. 25.1.1.0.0)
		t.Fatalf("version constant looks too short: %q", version)
	}
}

// TestMenu0Quit verifies that --menu 0 exits cleanly (exit code 0) without hanging.
// This smoke test exercises the full init path (config load, host key, writers, servers)
// and confirms the graceful shutdown sequence completes without errors.
func TestMenu0Quit(t *testing.T) {
	exe, err := exec.LookPath("go") // find the go toolchain on PATH; required for go run
	if err != nil {
		t.Fatalf("go toolchain not found: %v", err) // CI must have Go available
	}

	cmd := exec.Command(exe, "run", ".", "--menu", "0") // --menu 0 means: init, skip interactive loop, shut down cleanly
	cmd.Dir = "."                                       // run from the package directory so relative "data/" paths resolve correctly
	cmd.Env = append(os.Environ(),                      // start from the real environment to preserve PATH and GOPATH
		"MIST_API_TOKEN=test", // stub API token satisfies LoadConfig validation
		"MIST_ORG_ID=test",    // stub org ID (not a valid UUID but passes the non-empty check)
		"SSH_PORT=0",          // port 0 lets the OS assign a free port, avoiding bind conflicts in CI
		"WEB_PORT=0",          // same for the HTTP server
	)

	out, err := cmd.CombinedOutput() // capture stdout+stderr; cmd.Run sets err if exit code != 0
	if err != nil {                  // --menu 0 must always exit 0
		t.Fatalf("--menu 0 exited with error: %v\noutput: %s", err, out)
	}
}
