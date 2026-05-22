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

	cmd := exec.Command(exe, "run", ".", "-version")     // Run the current package with --version
	cmd.Dir = "."                                         // Run from the package directory
	cmd.Env = append(os.Environ(), "MIST_API_TOKEN=test", "MIST_ORG_ID=test") // Provide stub creds so loadConfig does not exit 1

	out, err := cmd.CombinedOutput()  // Capture stdout+stderr together
	if err != nil {                   // --version should always exit 0
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
