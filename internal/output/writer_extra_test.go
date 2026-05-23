// Package output -- extra unit tests covering uncovered paths in writer.go and strategies.go.
// Focuses on: empty-record paths, default Get() path, insertConflictClause branches,
// buildArgs missing-key path, and pure helper functions.
package output

import (
	"context"       // for context.Background() passed to Write methods
	"fmt"           // for fmt.Sprintf -- builds the trigger DDL in the insert-rows error test
	"os"            // for os.WriteFile, os.Mkdir -- creates blocking files/dirs in error-path tests
	"path/filepath" // for filepath.Join -- cross-platform path construction
	"testing"       // for testing.T -- standard test runner

	"github.com/jmorrison-juniper/misthelper-go/internal/api" // for api.Config in NewWriter calls
)

// ── Get() default path ────────────────────────────────────────────────────────

// TestGet_UnknownEndpointReturnsDefault verifies the fallback branch in Get:
// an endpoint that is not in Strategies must return a safe auto-increment strategy.
func TestGet_UnknownEndpointReturnsDefault(t *testing.T) {
	t.Parallel()                                                    // Independent of all other tests
	strategy := Get("xNonExistentEndpoint_DefinitelyNotRegistered") // Look up an endpoint that has no entry
	if strategy.Type != PKTypeAutoIncrement {                       // Default must be auto-increment
		t.Errorf("Get(unknown).Type = %q; want %q", strategy.Type, PKTypeAutoIncrement) // Report wrong type
	}
	if len(strategy.PrimaryKey) == 0 { // Must have a placeholder primary key
		t.Error("Get(unknown).PrimaryKey is empty; want at least one element") // Report missing PK
	}
	if strategy.PrimaryKey[0] != "misthelper_internal_id" { // Must use the virtual PK column name
		t.Errorf("Get(unknown).PrimaryKey[0] = %q; want %q", strategy.PrimaryKey[0], "misthelper_internal_id")
	}
}

// ── insertConflictClause branches ────────────────────────────────────────────

// TestInsertConflictClause_Natural verifies PKTypeNatural produces "OR REPLACE".
// Natural-PK endpoints must upsert so that a re-import refreshes stale rows.
func TestInsertConflictClause_Natural(t *testing.T) {
	t.Parallel()                                                       // Independent of all other tests
	got := insertConflictClause(EndpointStrategy{Type: PKTypeNatural}) // Natural PK uses upsert semantics
	if got != "OR REPLACE" {                                           // Must be OR REPLACE to enable upsert
		t.Errorf("insertConflictClause(natural) = %q; want %q", got, "OR REPLACE") // Report wrong clause
	}
}

// TestInsertConflictClause_Composite verifies PKTypeComposite produces "OR REPLACE".
// Composite-PK endpoints (time-series) must also upsert to avoid duplicate rows.
func TestInsertConflictClause_Composite(t *testing.T) {
	t.Parallel()                                                         // Independent of all other tests
	got := insertConflictClause(EndpointStrategy{Type: PKTypeComposite}) // Composite PK also uses upsert
	if got != "OR REPLACE" {                                             // Must be OR REPLACE so re-imports deduplicate
		t.Errorf("insertConflictClause(composite) = %q; want %q", got, "OR REPLACE") // Report wrong clause
	}
}

// TestInsertConflictClause_AutoIncrement verifies the default branch produces "OR IGNORE".
// Auto-increment endpoints append rows and must never overwrite existing data.
func TestInsertConflictClause_AutoIncrement(t *testing.T) {
	t.Parallel()                                                             // Independent of all other tests
	got := insertConflictClause(EndpointStrategy{Type: PKTypeAutoIncrement}) // Auto-increment uses append-only semantics
	if got != "OR IGNORE" {                                                  // Must be OR IGNORE so rows are never overwritten
		t.Errorf("insertConflictClause(auto_increment) = %q; want %q", got, "OR IGNORE") // Report wrong clause
	}
}

// TestInsertConflictClause_UnknownType verifies an unrecognised Type falls to "OR IGNORE".
// Any unregistered type must default to the safest behaviour (append-only) to avoid data loss.
func TestInsertConflictClause_UnknownType(t *testing.T) {
	t.Parallel()                                                            // Independent of all other tests
	got := insertConflictClause(EndpointStrategy{Type: "some_future_type"}) // Unknown type hits the default case
	if got != "OR IGNORE" {                                                 // Default must be OR IGNORE for safety
		t.Errorf("insertConflictClause(unknown) = %q; want %q", got, "OR IGNORE") // Report wrong clause
	}
}

// ── buildArgs missing-key path ────────────────────────────────────────────────

// TestBuildArgs_PresentKeys verifies buildArgs extracts values in header order.
func TestBuildArgs_PresentKeys(t *testing.T) {
	t.Parallel()                                  // Independent of all other tests
	flat := map[string]any{"a": "alpha", "b": 42} // Flat record with two known keys
	args := buildArgs(flat, []string{"b", "a"})   // Request columns in reversed order
	if len(args) != 2 {                           // Must have one slot per header
		t.Fatalf("buildArgs returned %d args; want 2", len(args)) // Bail if length is wrong
	}
	if args[0] != 42 { // First slot must be the value for "b"
		t.Errorf("buildArgs[0] = %v; want 42", args[0]) // Report wrong value
	}
	if args[1] != "alpha" { // Second slot must be the value for "a"
		t.Errorf("buildArgs[1] = %v; want \"alpha\"", args[1]) // Report wrong value
	}
}

// TestBuildArgs_MissingKey verifies that a header not present in the flat map
// produces an empty-string placeholder rather than nil (avoids NULL in SQLite).
func TestBuildArgs_MissingKey(t *testing.T) {
	t.Parallel()                                // Independent of all other tests
	flat := map[string]any{"a": "alpha"}        // Flat map contains only "a", not "b"
	args := buildArgs(flat, []string{"a", "b"}) // "b" is in headers but missing from flat
	if len(args) != 2 {                         // Must still have one slot per header
		t.Fatalf("buildArgs returned %d args; want 2", len(args)) // Bail if length is wrong
	}
	if args[0] != "alpha" { // First slot: "a" is present -- use its value
		t.Errorf("buildArgs[0] = %v; want \"alpha\"", args[0]) // Report wrong value
	}
	if args[1] != "" { // Second slot: "b" is missing -- must be empty string
		t.Errorf("buildArgs[1] = %v (%T); want \"\" (string)", args[1], args[1]) // Report non-empty value
	}
}

// ── buildRow missing-key path ─────────────────────────────────────────────────

// TestBuildRow_MissingKey verifies that a header not in the flat map yields an empty CSV cell.
// Missing columns must produce empty strings, not "map[]" or panics.
func TestBuildRow_MissingKey(t *testing.T) {
	t.Parallel()                              // Independent of all other tests
	flat := map[string]any{"x": "hello"}      // Flat map has only "x", not "y"
	row := buildRow(flat, []string{"x", "y"}) // "y" is in headers but absent in flat
	if row[0] != "hello" {                    // First cell: "x" is present
		t.Errorf("buildRow[0] = %q; want \"hello\"", row[0]) // Report wrong value
	}
	if row[1] != "" { // Second cell: "y" is absent -- must be empty
		t.Errorf("buildRow[1] = %q; want empty string for missing key", row[1]) // Report non-empty value
	}
}

// ── collectSortedKeys ─────────────────────────────────────────────────────────

// TestCollectSortedKeys_Alphabetical verifies that keys are sorted alphabetically.
// Deterministic column order makes CSV diffs and SQLite DDL stable across runs.
func TestCollectSortedKeys_Alphabetical(t *testing.T) {
	t.Parallel()                                              // Independent of all other tests
	m := map[string]any{"charlie": 3, "alpha": 1, "bravo": 2} // Unsorted map with three keys
	keys := collectSortedKeys(m)                              // Must return keys in alphabetical order
	want := []string{"alpha", "bravo", "charlie"}             // Expected alphabetical order
	if len(keys) != len(want) {                               // Must have the same count
		t.Fatalf("collectSortedKeys returned %d keys; want %d", len(keys), len(want)) // Bail if count differs
	}
	for i, w := range want { // Verify each position in order
		if keys[i] != w { // Each key must be in the right position
			t.Errorf("collectSortedKeys[%d] = %q; want %q", i, keys[i], w) // Report wrong key at this position
		}
	}
}

// TestCollectSortedKeys_Empty verifies that an empty map returns an empty slice.
func TestCollectSortedKeys_Empty(t *testing.T) {
	t.Parallel()                                // Independent of all other tests
	keys := collectSortedKeys(map[string]any{}) // Empty input must produce empty output
	if len(keys) != 0 {                         // Must return an empty slice, not nil with elements
		t.Errorf("collectSortedKeys(empty) returned %d keys; want 0", len(keys)) // Report unexpected keys
	}
}

// ── sanitizeTableName ─────────────────────────────────────────────────────────

// TestSanitizeTableName_ReplacesSpecialChars verifies that non-alphanumeric characters
// are converted to underscores so the table name is a valid SQL identifier.
func TestSanitizeTableName_ReplacesSpecialChars(t *testing.T) {
	t.Parallel()                                  // Independent of all other tests
	got := sanitizeTableName("list-org-sites/v1") // Input with hyphens and slash
	want := "list_org_sites_v1"                   // All special chars replaced with underscores
	if got != want {                              // Must match expected safe identifier
		t.Errorf("sanitizeTableName(%q) = %q; want %q", "list-org-sites/v1", got, want) // Report the difference
	}
}

// TestSanitizeTableName_Alphanumeric verifies that an already-safe name is not modified.
func TestSanitizeTableName_Alphanumeric(t *testing.T) {
	t.Parallel()                    // Independent of all other tests
	input := "listOrgSites123"      // Already a valid identifier with no special chars
	got := sanitizeTableName(input) // Must pass through without modification
	if got != input {               // Unchanged input must produce unchanged output
		t.Errorf("sanitizeTableName(%q) = %q; want unchanged", input, got) // Report unexpected modification
	}
}

// ── buildColumnDefs ───────────────────────────────────────────────────────────

// TestBuildColumnDefs_Format verifies each definition is in `"col" TEXT` format.
// The DDL must use this exact format so SQLite accepts the CREATE TABLE statement.
func TestBuildColumnDefs_Format(t *testing.T) {
	t.Parallel()                                         // Independent of all other tests
	defs := buildColumnDefs([]string{"site_id", "name"}) // Build definitions for two columns
	if len(defs) != 2 {                                  // Must have one definition per column
		t.Fatalf("buildColumnDefs returned %d defs; want 2", len(defs)) // Bail if count is wrong
	}
	if defs[0] != `"site_id" TEXT` { // First definition must match exact format
		t.Errorf("buildColumnDefs[0] = %q; want %q", defs[0], `"site_id" TEXT`) // Report wrong format
	}
	if defs[1] != `"name" TEXT` { // Second definition must match exact format
		t.Errorf("buildColumnDefs[1] = %q; want %q", defs[1], `"name" TEXT`) // Report wrong format
	}
}

// ── buildPlaceholders ─────────────────────────────────────────────────────────

// TestBuildPlaceholders_Three verifies that 3 columns produce "?, ?, ?".
// Parameterised queries prevent SQL injection via API field values.
func TestBuildPlaceholders_Three(t *testing.T) {
	t.Parallel()                // Independent of all other tests
	got := buildPlaceholders(3) // Build placeholder string for 3 columns
	want := "?, ?, ?"           // Expected placeholder format
	if got != want {            // Must match exact SQL VALUES syntax
		t.Errorf("buildPlaceholders(3) = %q; want %q", got, want) // Report wrong format
	}
}

// TestBuildPlaceholders_One verifies that 1 column produces just "?".
func TestBuildPlaceholders_One(t *testing.T) {
	t.Parallel()                                 // Independent of all other tests
	if got := buildPlaceholders(1); got != "?" { // Single placeholder must have no commas
		t.Errorf("buildPlaceholders(1) = %q; want %q", got, "?") // Report wrong format
	}
}

// ── quoteColumns ─────────────────────────────────────────────────────────────

// TestQuoteColumns_ProducesDoubleQuotes verifies column names are double-quoted
// for safe use as SQL identifiers in the INSERT column list.
func TestQuoteColumns_ProducesDoubleQuotes(t *testing.T) {
	t.Parallel()                                        // Independent of all other tests
	quoted := quoteColumns([]string{"site id", "name"}) // Column names with space and plain name
	if len(quoted) != 2 {                               // Must have one quoted name per input
		t.Fatalf("quoteColumns returned %d elements; want 2", len(quoted)) // Bail if count is wrong
	}
	if quoted[0] != `"site id"` { // Column with space must be double-quoted
		t.Errorf("quoteColumns[0] = %q; want %q", quoted[0], `"site id"`) // Report wrong quoting
	}
	if quoted[1] != `"name"` { // Plain column must also be double-quoted
		t.Errorf("quoteColumns[1] = %q; want %q", quoted[1], `"name"`) // Report wrong quoting
	}
}

// ── buildInsertSQL ────────────────────────────────────────────────────────────

// TestBuildInsertSQL_NaturalPK verifies the INSERT statement for a natural-PK strategy
// uses "OR REPLACE" and includes the expected structural keywords.
func TestBuildInsertSQL_NaturalPK(t *testing.T) {
	t.Parallel()                                                                            // Independent of all other tests
	strategy := EndpointStrategy{Type: PKTypeNatural}                                       // Natural PK strategy drives OR REPLACE
	sql := buildInsertSQL("myTable", []string{"id", "name"}, strategy)                      // Build the INSERT statement
	for _, want := range []string{"INSERT", "OR REPLACE", `"myTable"`, "id", "name", "?"} { // All expected substrings
		if !containsStr(sql, want) { // Each substring must appear
			t.Errorf("buildInsertSQL result missing %q: got %q", want, sql) // Report the missing part
		}
	}
}

// TestBuildInsertSQL_AutoIncrement verifies the INSERT statement for auto-increment uses "OR IGNORE".
func TestBuildInsertSQL_AutoIncrement(t *testing.T) {
	t.Parallel()                                                 // Independent of all other tests
	strategy := EndpointStrategy{Type: PKTypeAutoIncrement}      // Auto-increment uses OR IGNORE
	sql := buildInsertSQL("aggTable", []string{"val"}, strategy) // Build the INSERT statement
	if !containsStr(sql, "OR IGNORE") {                          // Must use OR IGNORE for append-only tables
		t.Errorf("buildInsertSQL(auto-increment) missing OR IGNORE: got %q", sql) // Report missing clause
	}
}

// containsStr is a local helper that checks if substr appears in s.
// Used to verify SQL statement substrings without exact-format coupling.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && func() bool { // Trivial O(n) contains check for test use only
		for i := 0; i <= len(s)-len(substr); i++ { // Slide a window of substr length over s
			if s[i:i+len(substr)] == substr { // Compare window to substr
				return true // Found a match
			}
		}
		return false // No match found
	}()
}

// ── Empty-record paths ────────────────────────────────────────────────────────

// TestCSVWriter_EmptyRecords verifies that Write returns nil without creating a file
// when the records slice is empty. Operators must not see an empty CSV artifact.
func TestCSVWriter_EmptyRecords(t *testing.T) {
	t.Parallel()                                                              // Independent of all other tests
	dir := t.TempDir()                                                        // Throwaway dir -- no CSV file should appear here
	w := newCSVWriter(dir)                                                    // Construct CSV writer directly (no error to check)
	err := w.Write(context.Background(), "emptyEndpoint", []map[string]any{}) // Write empty slice
	if err != nil {                                                           // Empty write must return nil, not an error
		t.Errorf("csvWriter.Write(empty) returned error: %v; want nil", err) // Report unexpected error
	}
	// Verify no CSV file was created -- empty writes must not produce artifacts
	files, _ := filepath.Glob(filepath.Join(dir, "emptyEndpoint_*.csv")) // Glob for any CSV with this prefix
	if len(files) != 0 {                                                 // Must be zero -- empty writes create no files
		t.Errorf("csvWriter.Write(empty) created %d file(s); want 0", len(files)) // Report unexpected file
	}
}

// TestSQLiteWriter_EmptyRecords verifies that Write returns nil without touching the
// database when the records slice is empty.
func TestSQLiteWriter_EmptyRecords(t *testing.T) {
	t.Parallel()                    // Independent of all other tests
	dir := t.TempDir()              // Throwaway dir for the SQLite database
	sw, err := newSQLiteWriter(dir) // Create a real SQLite writer
	if err != nil {                 // newSQLiteWriter must succeed with a valid temp dir
		t.Fatalf("newSQLiteWriter: %v", err) // Bail if database init fails
	}
	defer func() { _ = sw.Close() }()                                   // Ensure the database is closed after the test
	err = sw.Write(context.Background(), "emptyOp", []map[string]any{}) // Write empty slice
	if err != nil {                                                     // Empty write must return nil, not an error
		t.Errorf("sqliteWriter.Write(empty) returned error: %v; want nil", err) // Report unexpected error
	}
}

// ── NewWriter MkdirAll error path ─────────────────────────────────────────────

// TestNewWriter_MkdirAllError verifies that NewWriter returns an error when
// the data directory cannot be created (parent path is a regular file).
func TestNewWriter_MkdirAllError(t *testing.T) {
	t.Parallel()                                                      // Independent of all other tests
	base := t.TempDir()                                               // Writable temp root
	blocker := filepath.Join(base, "blocker")                         // Path for the blocking regular file
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil { // Create a file where a dir would be needed
		t.Fatalf("failed to create blocker file: %v", err) // Bail if file creation itself fails
	}
	dataDir := filepath.Join(blocker, "subdir")                   // Attempt to create a dir inside a file -- must fail on all platforms
	_, err := NewWriter(api.Config{OutputFormat: "csv"}, dataDir) // NewWriter must call MkdirAll which must fail
	if err == nil {                                               // A nil error means MkdirAll silently succeeded (wrong)
		t.Error("NewWriter returned nil error for uncreateable dataDir; want error") // Report missing validation
	}
}

// ── newSQLiteWriter error path ────────────────────────────────────────────────

// TestNewSQLiteWriter_InvalidDataDir verifies that newSQLiteWriter returns an error
// when the database file cannot be created (parent directory does not exist).
// SQLite's db.Ping() fails when the OS cannot open the file for writing.
func TestNewSQLiteWriter_InvalidDataDir(t *testing.T) {
	t.Parallel()                                                                    // Independent of all other tests
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist", "deeply", "nested") // Path under a non-existent directory chain
	_, err := newSQLiteWriter(nonExistent)                                          // Must fail -- directory chain does not exist
	if err == nil {                                                                 // Nil error means the DB was created in a bad path (wrong)
		t.Error("newSQLiteWriter returned nil error for non-existent dataDir; want error") // Report missing validation
	}
}

// ── SQLite error-path tests ───────────────────────────────────────────────────

// TestNewWriter_SQLiteCreateError verifies that NewWriter returns an error when
// the sqlite format is requested but newSQLiteWriter fails (e.g. db path is a directory).
// This covers the "if err != nil { return nil, fmt.Errorf("create sqlite writer...")" path
// inside NewWriter's "sqlite" format switch case.
func TestNewWriter_SQLiteCreateError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	dir := t.TempDir()                                // Writable scratch directory
	dbBlocker := filepath.Join(dir, "mist_data.db")   // SQLite db file path
	if err := os.Mkdir(dbBlocker, 0755); err != nil { // Create a DIRECTORY where the db file should go -- Ping will fail
		t.Fatalf("setup: create blocking directory: %v", err) // Bail if fixture cannot be created
	}
	_, err := NewWriter(api.Config{OutputFormat: "sqlite"}, dir) // NewWriter must fail because db path is a directory
	if err == nil {                                              // Must NOT succeed when the db path is a directory
		t.Error("NewWriter(sqlite) returned nil error when db path is a directory; want error")
	}
}

// TestSQLiteWriter_WriteEnsureTableError verifies the error path in sqliteWriter.Write
// where ensureTable fails (e.g. because the database is already closed). This covers:
//   - ensureTable: "if _, err := w.db.ExecContext(...); err != nil { return fmt.Errorf(...) }"
//   - Write:       "if err := w.ensureTable(...); err != nil { return fmt.Errorf(...) }"
func TestSQLiteWriter_WriteEnsureTableError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	dir := t.TempDir()              // Scratch dir for SQLite database
	sw, err := newSQLiteWriter(dir) // Create a working SQLite writer
	if err != nil {                 // Bail if setup fails
		t.Fatalf("newSQLiteWriter: %v", err)
	}
	_ = sw.Close() // Close the database connection -- ExecContext will now fail

	// Write attempts ensureTable first; with the DB closed, ExecContext returns an error
	err = sw.Write(context.Background(), "testOp", []map[string]any{{"col1": "val1"}}) // Must fail because DB is closed
	if err == nil {                                                                    // Must NOT succeed when DB is closed
		t.Error("Write returned nil error with closed database; want error from ensureTable")
	}
}

// TestSQLiteWriter_WriteInsertRowsError verifies the error path in sqliteWriter.Write
// where insertRows fails after ensureTable succeeds. A BEFORE INSERT trigger that calls
// RAISE(ABORT, ...) is used to block all inserts while keeping the database open.
// This covers:
//   - insertRows: "if _, err := w.db.ExecContext(...); err != nil { return written, fmt.Errorf(...) }"
//   - Write:      "if err != nil { return fmt.Errorf("insert rows into %s: %w"...) }"
func TestSQLiteWriter_WriteInsertRowsError(t *testing.T) {
	t.Parallel() // Independent of all other tests

	dir := t.TempDir()              // Scratch dir for SQLite database
	sw, err := newSQLiteWriter(dir) // Create a working SQLite writer
	if err != nil {                 // Bail if setup fails
		t.Fatalf("newSQLiteWriter: %v", err)
	}
	defer func() { _ = sw.Close() }() // Ensure cleanup after test

	ctx := context.Background()                   // Background context for all DB operations
	endpoint := "blockedOp"                       // Arbitrary endpoint name for test table
	records := []map[string]any{{"col1": "val1"}} // One record -- enough to exercise the insert path

	// First Write: creates the table and inserts the record successfully
	if err = sw.Write(ctx, endpoint, records); err != nil { // This must succeed to create the table
		t.Fatalf("initial Write failed: %v", err)
	}

	// Install a BEFORE INSERT trigger that aborts all future inserts on this table.
	// The table exists (created above), so ensureTable (IF NOT EXISTS) will be a no-op.
	// The trigger fires BEFORE every INSERT, causing ExecContext in insertRows to fail.
	table := sanitizeTableName(endpoint) // Build the table name the same way Write does
	triggerSQL := fmt.Sprintf(           // DDL for the blocking trigger
		`CREATE TRIGGER IF NOT EXISTS block_insert BEFORE INSERT ON %q `+
			`BEGIN SELECT RAISE(ABORT, 'blocked by test trigger'); END`,
		table,
	)
	if _, err = sw.db.ExecContext(ctx, triggerSQL); err != nil { // Install the blocking trigger
		t.Fatalf("failed to create blocking trigger: %v", err) // Bail if trigger creation fails
	}

	// Second Write: ensureTable succeeds (IF NOT EXISTS + table already exists),
	// but insertRows fails because the trigger raises ABORT for every INSERT.
	err = sw.Write(ctx, endpoint, records) // Must fail because trigger blocks all inserts
	if err == nil {                        // Must NOT succeed when inserts are blocked
		t.Error("Write returned nil error when INSERT trigger blocks all inserts; want error")
	}
}
