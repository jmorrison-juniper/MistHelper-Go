package output

import (
	"context"       // for context.Background() in Write calls
	"database/sql"  // for sql.Open to verify DB state in TestSQLiteWriter
	"fmt"           // for fmt.Sprintf to build test strings
	"os"            // for os.ReadFile to read CSV output in verification helper
	"path/filepath" // for filepath.Join and filepath.Glob in helper functions
	"strings"       // for strings.Count to count lines in CSV helper
	"testing"       // for *testing.T and test registration

	"github.com/jmorrison-juniper/misthelper-go/internal/api" // for api.Config struct
)

// testRecords returns two deterministic records for use in writer tests.
// Using a factory keeps test data in one place so any schema change is trivial to update.
func testRecords() []map[string]any {
	return []map[string]any{ // Two records matching the listOrgSites natural-PK strategy
		{"id": "site-1", "name": "Alpha", "org_id": "org-99"}, // First site record
		{"id": "site-2", "name": "Beta", "org_id": "org-99"},  // Second site record
	}
}

// TestCSVWriter_WritesFile verifies that Write creates exactly one CSV file
// with a header row plus one data row per record.
func TestCSVWriter_WritesFile(t *testing.T) {
	dir := t.TempDir() // Create a throwaway temp directory that is auto-cleaned after the test

	cfg := api.Config{OutputFormat: "csv"} // Configure the writer to use the CSV backend
	w, err := NewWriter(cfg, dir)          // Create the writer using the factory
	if err != nil {                        // NewWriter must succeed for a valid CSV format
		t.Fatalf("NewWriter: %v", err) // Fatal so subsequent nil-deref is avoided
	}
	defer func() { _ = w.Close() }() // Ensure writer resources are released even if assertions fail (best-effort cleanup)

	if err := w.Write(context.Background(), "testEndpoint", testRecords()); err != nil { // Write test records
		t.Fatalf("Write: %v", err) // Fatal: no point verifying the file if the write failed
	}

	pattern := filepath.Join(dir, "testEndpoint_*.csv") // Glob pattern to find the timestamped output file
	files, err := filepath.Glob(pattern)                // List files matching the pattern
	if err != nil {                                     // Glob only errors on malformed patterns
		t.Fatalf("Glob: %v", err) // Fatal -- malformed pattern is a test bug
	}
	if len(files) != 1 { // Exactly one file should be created per Write call
		t.Fatalf("expected 1 CSV file, got %d (pattern=%s)", len(files), pattern) // Report count and pattern
	}

	verifyCSVLines(t, files[0], 3) // Expect header line + 2 data lines = 3 total lines
}

// TestSQLiteWriter_WritesDB verifies that Write creates mist_data.db with a table
// containing the correct number of rows for the "listOrgSites" endpoint.
func TestSQLiteWriter_WritesDB(t *testing.T) {
	dir := t.TempDir() // Create a throwaway temp directory that is auto-cleaned after the test

	cfg := api.Config{OutputFormat: "sqlite"} // Configure the writer to use the SQLite backend
	w, err := NewWriter(cfg, dir)             // Open the database (creates mist_data.db in dir)
	if err != nil {                           // Must succeed -- SQLite driver is registered at package init
		t.Fatalf("NewWriter: %v", err) // Fatal so subsequent nil-deref is avoided
	}

	if err := w.Write(context.Background(), "listOrgSites", testRecords()); err != nil { // Write 2 rows
		t.Fatalf("Write: %v", err) // Fatal: no point verifying the DB if the write failed
	}
	if err := w.Close(); err != nil { // Close before re-opening so all WAL pages are flushed
		t.Logf("w.Close: %v", err) // Log but do not fatal -- DB verify step will catch real problems
	}

	dbPath := filepath.Join(dir, "mist_data.db") // Canonical database path used by newSQLiteWriter
	verifyDBRows(t, dbPath, "listOrgSites", 2)   // Expect exactly 2 rows in the listOrgSites table
}

// TestNewWriter_InvalidFormat verifies that NewWriter returns a non-nil error
// when given an unsupported output format string.
func TestNewWriter_InvalidFormat(t *testing.T) {
	dir := t.TempDir() // Create a throwaway temp directory for the test

	cfg := api.Config{OutputFormat: "parquet"} // Use an unsupported format to trigger the error path
	_, err := NewWriter(cfg, dir)              // Should fail with a descriptive error
	if err == nil {                            // A nil error here means the guard clause is missing
		t.Error("expected error for unsupported format, got nil") // Report missing validation
	}
}

// ── Test helpers ─────────────────────────────────────────────────────────────

// verifyCSVLines reads the CSV file at path and fails the test if the number
// of non-empty lines does not equal want.
func verifyCSVLines(t *testing.T, path string, want int) {
	t.Helper()                        // Mark as helper so failure lines point to the caller
	data, err := os.ReadFile(path)    // Read the entire CSV file into memory
	if err != nil {                   // File must exist -- if not the write step failed silently
		t.Fatalf("read csv %s: %v", path, err) // Report the failing path and error
	}
	content := string(data)                      // Convert bytes to string for line counting
	got := strings.Count(content, "\n")          // Count newline characters as a proxy for row count
	if got != want {                             // Row count must match the expected value
		t.Errorf("expected %d lines in %s, got %d\n%s", want, path, got, content) // Show file content for debugging
	}
}

// verifyDBRows opens the SQLite database at dbPath and fails the test if
// SELECT COUNT(*) FROM table does not return want.
func verifyDBRows(t *testing.T, dbPath, table string, want int) {
	t.Helper()                                   // Mark as helper so failure lines point to the caller
	db, err := sql.Open("sqlite", dbPath)        // Open the database with the modernc driver
	if err != nil {                              // Must succeed -- file was created by the writer
		t.Fatalf("open db %s: %v", dbPath, err) // Report the failing path and error
	}
	defer func() { _ = db.Close() }() // Release the connection after the assertion (best-effort; test cleanup)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %q", table) // Count rows in the named table
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil { // Execute the count query
		t.Fatalf("count rows in %q: %v", table, err) // Fail if the table doesn't exist or query fails
	}
	if got != want { // Row count must match the number of records written
		t.Errorf("expected %d rows in table %q, got %d", want, table, got) // Report count mismatch
	}
}
