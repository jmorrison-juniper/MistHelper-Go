// Package ssh -- additional error-path tests for LoadOrCreateHostKey, keyFileExists,
// generateAndSaveKey, and loadKeyFromFile. These cover the branches where OS calls
// fail so the main test file can focus on the happy path.
package ssh

import (
	"os"            // for os.WriteFile, os.Mkdir -- creates blocking files and directories for tests
	"path/filepath" // for filepath.Join -- builds test key paths cross-platform
	"testing"       // for testing.T -- standard Go test runner
)

// ── keyFileExists error-path tests ───────────────────────────────────────────

// TestKeyFileExists_StatError verifies that keyFileExists returns false and a non-nil error
// when os.Stat returns an error that is NOT os.IsNotExist (e.g. ENOTDIR).
// To trigger this deterministically across OSes, pass a path with a NUL byte, which
// os.Stat rejects as an invalid path (not an IsNotExist condition).
func TestKeyFileExists_StatError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	exists, err := keyFileExists("invalid\x00path") // Invalid path should force a non-IsNotExist stat error on all platforms
	if err == nil {                                 // Must return an error (not nil)
		t.Error("expected non-nil error from keyFileExists when parent is a regular file")
	}
	if exists { // Must return exists=false when the stat fails
		t.Error("expected exists=false when keyFileExists encounters a stat error")
	}
}

// ── generateAndSaveKey error-path tests ──────────────────────────────────────

// TestGenerateAndSaveKey_WriteError verifies that generateAndSaveKey returns a non-nil error
// when os.WriteFile fails. This is triggered by using a directory as the target key path --
// WriteFile on a directory returns EISDIR (Linux) or ERROR_ACCESS_DENIED (Windows).
func TestGenerateAndSaveKey_WriteError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	dir := t.TempDir()                                // Scratch directory for this test
	keyPath := filepath.Join(dir, "ssh_host_rsa_key") // The path where we will create an obstacle
	if err := os.Mkdir(keyPath, 0755); err != nil {   // Create a DIRECTORY at the key path (not a file)
		t.Fatalf("setup: create blocking directory: %v", err) // Bail if the obstacle cannot be created
	}
	// WriteFile on a directory returns EISDIR -- triggers the error return in generateAndSaveKey
	err := generateAndSaveKey(keyPath) // Must fail because keyPath is a directory, not a writable file
	if err == nil {                    // Must NOT succeed when writing to a directory
		t.Error("expected non-nil error from generateAndSaveKey when keyPath is a directory")
	}
}

// ── LoadOrCreateHostKey error-path tests ─────────────────────────────────────

// TestLoadOrCreateHostKey_StatError verifies that LoadOrCreateHostKey returns an error when
// keyFileExists returns a non-IsNotExist stat error. This covers the
// "if err != nil { return nil, fmt.Errorf("check host key ...") }" branch.
func TestLoadOrCreateHostKey_StatError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	parent := t.TempDir()                          // Scratch directory
	blockingFile := filepath.Join(parent, "block") // A regular file that will block directory stat
	if err := os.WriteFile(blockingFile, []byte("x"), 0644); err != nil {
		t.Fatalf("setup: write blocking file: %v", err) // Bail if the test fixture cannot be created
	}
	// dataDir = blockingFile (a regular file, not a directory)
	// keyPath = filepath.Join(blockingFile, "ssh_host_rsa_key") → e.g. "/tmp/x/block/ssh_host_rsa_key"
	// os.Stat on this returns ENOTDIR -- not IsNotExist -- so keyFileExists returns (false, err)
	_, err := LoadOrCreateHostKey(blockingFile) // Passes a file path as dataDir to trigger the error
	if err == nil {                             // Must return an error
		t.Error("expected non-nil error from LoadOrCreateHostKey when stat fails with ENOTDIR")
	}
}

// TestLoadOrCreateHostKey_GenerateError verifies that LoadOrCreateHostKey returns an error when
// generateAndSaveKey fails. This covers the
// "if genErr != nil { return nil, fmt.Errorf("generate host key ...") }" branch.
// Trigger: use a non-existent subdirectory as dataDir so that WriteFile fails with ENOENT.
func TestLoadOrCreateHostKey_GenerateError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	parent := t.TempDir()                            // Scratch directory
	nonExistentDir := filepath.Join(parent, "ghost") // This subdirectory is never created
	// keyPath = filepath.Join(nonExistentDir, "ssh_host_rsa_key")
	// keyFileExists → os.Stat returns ENOENT → IsNotExist=true → returns (false, nil)
	// generateAndSaveKey → os.WriteFile fails because "ghost" directory does not exist
	_, err := LoadOrCreateHostKey(nonExistentDir) // Must fail because the parent directory is missing
	if err == nil {                               // Must return an error
		t.Error("expected non-nil error from LoadOrCreateHostKey when key directory does not exist")
	}
}

// ── loadKeyFromFile error-path tests ─────────────────────────────────────────

// TestLoadKeyFromFile_OpenRootError verifies that loadKeyFromFile returns an error when
// os.OpenRoot fails. This triggers the "open root dir" error branch.
// Trigger: pass a key path whose parent directory does not exist.
func TestLoadKeyFromFile_OpenRootError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	parent := t.TempDir()                                        // Scratch directory
	nonExistentDir := filepath.Join(parent, "ghost")             // Never created
	keyPath := filepath.Join(nonExistentDir, "ssh_host_rsa_key") // Parent directory does not exist
	// os.OpenRoot(nonExistentDir) returns ENOENT because the directory was never created
	_, err := loadKeyFromFile(keyPath) // Must fail at the OpenRoot step
	if err == nil {                    // Must return an error
		t.Error("expected non-nil error from loadKeyFromFile when parent directory does not exist")
	}
}

// TestLoadKeyFromFile_ReadError verifies that loadKeyFromFile returns an error when the key
// file does not exist within the opened root. This triggers the "read host key" error branch.
// Trigger: pass a key filename that does not exist inside the directory.
func TestLoadKeyFromFile_ReadError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	dir := t.TempDir()                                // Scratch directory that DOES exist (OpenRoot succeeds)
	keyPath := filepath.Join(dir, "ssh_host_rsa_key") // File does NOT exist inside dir
	// os.OpenRoot(dir) succeeds; root.ReadFile("ssh_host_rsa_key") returns ENOENT
	_, err := loadKeyFromFile(keyPath) // Must fail at the ReadFile step
	if err == nil {                    // Must return an error
		t.Error("expected non-nil error from loadKeyFromFile when key file does not exist")
	}
}

// TestLoadKeyFromFile_ParseError verifies that loadKeyFromFile returns an error when the key
// file exists but contains invalid PEM data. This triggers the "parse host key" error branch.
func TestLoadKeyFromFile_ParseError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	dir := t.TempDir()                                // Scratch directory
	keyPath := filepath.Join(dir, "ssh_host_rsa_key") // Key file path
	if err := os.WriteFile(keyPath, []byte("not valid pem"), 0600); err != nil {
		t.Fatalf("setup: write invalid key file: %v", err) // Bail if the test fixture cannot be created
	}
	// OpenRoot succeeds; ReadFile returns "not valid pem"; ParsePrivateKey returns an error
	_, err := loadKeyFromFile(keyPath) // Must fail at the ParsePrivateKey step
	if err == nil {                    // Must return an error
		t.Error("expected non-nil error from loadKeyFromFile when key file contains invalid PEM")
	}
}
