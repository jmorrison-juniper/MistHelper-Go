package output

import (
	"context"       // for context.Context passed to every Write call
	"database/sql"  // for sql.DB -- SQLite connection type
	"encoding/csv"  // for csv.Writer -- RFC 4180-compliant CSV output
	"fmt"           // for fmt.Errorf with %w for error chain wrapping
	"log/slog"      // for structured logging (Go 1.21+ standard library)
	"os"            // for os.Create and os.MkdirAll
	"path/filepath" // for filepath.Join -- cross-platform path construction
	"regexp"        // for regexp.MustCompile -- table name sanitisation
	"slices"        // for slices.Sort -- deterministic column ordering (Go 1.21+)
	"strings"       // for strings.Join -- SQL statement construction
	"time"          // for time.Now -- unique timestamp suffix in CSV filenames

	"github.com/jmorrison-juniper/misthelper-go/internal/api" // for api.Config struct
	_ "modernc.org/sqlite"                                    // register SQLite driver under name "sqlite"
)

// nonAlphanumericRE matches any character that is not a letter or digit.
// Pre-compiled at package init for efficient reuse in sanitizeTableName.
var nonAlphanumericRE = regexp.MustCompile(`[^a-zA-Z0-9]`) // Match non-alphanumeric chars for table name cleaning

// Writer is the output backend interface.
// All callers depend only on this interface -- never on a concrete type.
type Writer interface {
	// Write persists records to the configured output backend.
	// endpoint is the API endpoint name used to look up the PK strategy.
	Write(ctx context.Context, endpoint string, records []map[string]any) error
	// Close flushes and releases any resources held by the writer.
	Close() error
}

// ── CSV writer ───────────────────────────────────────────────────────────────

// csvWriter writes one CSV file per Write call into dataDir.
type csvWriter struct {
	dataDir string // Directory where CSV output files are created
}

// newCSVWriter constructs a csvWriter targeting the given directory.
func newCSVWriter(dataDir string) *csvWriter {
	return &csvWriter{dataDir: dataDir} // Construct with the output directory path
}

// Write creates a CSV file named {endpoint}_{timestamp}.csv in dataDir
// and writes all records to it.  Each record is flattened before writing.
func (w *csvWriter) Write(ctx context.Context, endpoint string, records []map[string]any) error {
	_ = ctx // Context reserved for future cancellation support; unused in file I/O today
	if len(records) == 0 { // Skip file creation entirely if there is nothing to write
		slog.Debug("CSV write skipped: no records", "endpoint", endpoint) // Log the skip so operators aren't confused
		return nil                                                         // Return nil -- empty is not an error
	}
	slog.Info("Writing CSV output", "endpoint", endpoint, "records", len(records)) // Log before starting file I/O

	ts := time.Now().Format("20060102_150405")          // Format timestamp for a unique filename suffix
	csvName := endpoint + "_" + ts + ".csv"             // Relative filename within dataDir (no traversal possible via os.OpenRoot)

	root, err := os.OpenRoot(w.dataDir)                  // Scope CSV creation within dataDir to prevent G304 path traversal
	if err != nil {                                      // OpenRoot fails if dataDir was removed after startup
		return fmt.Errorf("open data dir %s: %w", w.dataDir, err) // Wrap with dir for diagnosis
	}
	defer func() { _ = root.Close() }()                 // Release root fd on return (best-effort cleanup)

	file, err := root.Create(csvName)                    // Create or truncate the CSV file within the scoped root
	if err != nil {                                      // File creation may fail on permissions or disk full
		return fmt.Errorf("create csv file %s: %w", csvName, err) // Wrap with filename context for caller
	}
	defer func() { _ = file.Close() }() // Ensure the OS file handle is released even if writeRecords fails (best-effort; write errors reported above)

	if err := w.writeRecords(file, records); err != nil { // Delegate actual CSV row writing
		return fmt.Errorf("write csv records to %s: %w", csvName, err) // Wrap with filename context
	}

	slog.Debug("CSV write complete", "endpoint", endpoint, "file", csvName, "rows", len(records)) // Log after file I/O
	return nil                                                                                       // Signal success to the caller
}

// writeRecords writes the header row followed by one data row per record into file.
// Column order is determined alphabetically from the first record's flattened keys.
func (w *csvWriter) writeRecords(file *os.File, records []map[string]any) error {
	writer := csv.NewWriter(file) // Wrap the OS file in an RFC 4180 CSV encoder
	defer writer.Flush()         // Flush buffered CSV data to the OS file on return

	first := FlattenRecord(records[0])   // Flatten the first record to derive the column schema
	headers := collectSortedKeys(first)  // Sort column names for deterministic, diff-friendly output

	if err := writer.Write(headers); err != nil { // Write the header row as the first CSV line
		return fmt.Errorf("write csv header: %w", err) // Wrap error with context for debugging
	}

	for _, record := range records { // Write one data row for every record in the slice
		flat := FlattenRecord(record)      // Flatten nested fields to scalar string values
		row := buildRow(flat, headers)     // Extract values in the same order as headers
		if err := writer.Write(row); err != nil { // Write the row to the CSV encoder buffer
			return fmt.Errorf("write csv row: %w", err) // Wrap error with context for debugging
		}
	}
	return nil // All rows written successfully
}

// Close is a no-op for the CSV writer; files are closed immediately after each Write.
func (w *csvWriter) Close() error {
	return nil // CSV: no persistent resources to release between calls
}

// ── SQLite writer ────────────────────────────────────────────────────────────

// sqliteWriter writes records into a SQLite database at dataDir/mist_data.db.
type sqliteWriter struct {
	db *sql.DB // Open database connection -- must be closed via Close()
}

// newSQLiteWriter opens (or creates) the SQLite database file and verifies connectivity.
func newSQLiteWriter(dataDir string) (*sqliteWriter, error) {
	dbPath := filepath.Join(dataDir, "mist_data.db") // Canonical database file path inside data directory
	slog.Info("Opening SQLite database", "path", dbPath) // Log before opening so failures are traceable

	db, err := sql.Open("sqlite", dbPath) // Open or create the SQLite file using the modernc driver
	if err != nil {                        // sql.Open may fail if the driver is not registered
		return nil, fmt.Errorf("open sqlite db %s: %w", dbPath, err) // Wrap with path context
	}

	if err := db.Ping(); err != nil { // Verify the connection is live and the file is writable
		return nil, fmt.Errorf("ping sqlite db %s: %w", dbPath, err) // Wrap with path context
	}

	slog.Debug("SQLite database opened", "path", dbPath) // Log after successful open
	return &sqliteWriter{db: db}, nil                    // Return the live writer to the caller
}

// Write flattens records, ensures the table exists, then inserts all rows using
// the primary-key strategy registered for endpoint.
func (w *sqliteWriter) Write(ctx context.Context, endpoint string, records []map[string]any) error {
	if len(records) == 0 { // Skip DDL and DML entirely when there is nothing to write
		slog.Debug("SQLite write skipped: no records", "endpoint", endpoint) // Log the skip
		return nil                                                            // Empty is not an error
	}
	slog.Info("Writing SQLite output", "endpoint", endpoint, "records", len(records)) // Log before any DB work

	first := FlattenRecord(records[0])  // Flatten first record to determine the column schema
	headers := collectSortedKeys(first) // Sort columns for deterministic DDL and INSERT order
	table := sanitizeTableName(endpoint) // Convert endpoint name to a safe SQL identifier
	strategy := Get(endpoint)            // Look up primary-key strategy for upsert vs insert logic

	if err := w.ensureTable(ctx, table, headers); err != nil { // Create the table if it does not exist
		return fmt.Errorf("ensure table %s: %w", table, err) // Wrap with table name for context
	}

	written, err := w.insertRows(ctx, table, headers, records, strategy) // Insert all rows using the strategy
	if err != nil {                                                        // Any row insert failure aborts the batch
		return fmt.Errorf("insert rows into %s: %w", table, err) // Wrap with table name for context
	}

	slog.Debug("SQLite write complete", "endpoint", endpoint, "table", table, "rows", written) // Log after batch insert
	return nil                                                                                   // Signal success to the caller
}

// ensureTable runs CREATE TABLE IF NOT EXISTS for table with a TEXT column per entry in cols.
// All columns are TEXT -- SQLite's flexible typing means no schema migration is needed for the scaffold.
func (w *sqliteWriter) ensureTable(ctx context.Context, table string, cols []string) error {
	colDefs := buildColumnDefs(cols) // Build quoted "col" TEXT definitions for the DDL statement
	ddl := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %q (%s)", // Table name is double-quoted for safety
		table,
		strings.Join(colDefs, ", "), // Join column definitions with comma separator
	)
	slog.Info("Ensuring SQLite table exists", "table", table, "columns", len(cols)) // Log DDL intent
	if _, err := w.db.ExecContext(ctx, ddl); err != nil {                           // Execute CREATE TABLE
		return fmt.Errorf("create table %q: %w", table, err) // Wrap with table name context
	}
	slog.Debug("SQLite table ready", "table", table) // Log after successful DDL
	return nil                                        // Table exists and is ready for inserts
}

// buildColumnDefs returns a slice of `"col" TEXT` strings for use in a CREATE TABLE statement.
// Every column is TEXT so the schema matches any API response without migration.
func buildColumnDefs(cols []string) []string {
	defs := make([]string, len(cols)) // Allocate output slice with one slot per column
	for i, col := range cols {        // Iterate column names to build definition strings
		defs[i] = fmt.Sprintf("%q TEXT", col) // Quote column name and append TEXT type
	}
	return defs // Return the complete list of column definitions
}

// insertRows iterates records and executes one INSERT per row using the given strategy.
// Returns the count of successfully inserted rows for logging.
func (w *sqliteWriter) insertRows(ctx context.Context, table string, headers []string, records []map[string]any, strategy EndpointStrategy) (int, error) {
	insertSQL := buildInsertSQL(table, headers, strategy) // Build the INSERT OR REPLACE / OR IGNORE statement once
	written := 0                                          // Track the successful row count for the debug log

	for _, record := range records { // Insert one row per record in the batch
		flat := FlattenRecord(record)      // Flatten nested fields to scalars before binding
		args := buildArgs(flat, headers)   // Extract values in header column order for parameter binding
		if _, err := w.db.ExecContext(ctx, insertSQL, args...); err != nil { // Execute parameterised INSERT
			return written, fmt.Errorf("insert row: %w", err) // Return partial count plus error context
		}
		written++ // Increment after each successful insert to keep count accurate
	}
	return written, nil // Return total rows written and nil error on full success
}

// Close flushes the WAL and releases the SQLite database connection.
func (w *sqliteWriter) Close() error {
	slog.Info("Closing SQLite database connection") // Log before close so the shutdown is traceable
	if err := w.db.Close(); err != nil {            // Close flushes WAL and releases the file lock
		return fmt.Errorf("close sqlite db: %w", err) // Wrap error -- callers should log and report
	}
	slog.Debug("SQLite database connection closed") // Log after successful close
	return nil                                       // Signal clean shutdown to the caller
}

// ── Constructor ──────────────────────────────────────────────────────────────

// NewWriter creates and returns a Writer implementation matching cfg.OutputFormat.
// The output directory is created if it does not already exist.
// Supported formats: "csv" (default when empty), "sqlite".
func NewWriter(cfg api.Config, dataDir string) (Writer, error) {
	slog.Info("Creating output writer", "format", cfg.OutputFormat, "dataDir", dataDir) // Log intent

	if err := os.MkdirAll(dataDir, 0o750); err != nil { // Ensure the data directory exists before writing
		return nil, fmt.Errorf("create data directory %s: %w", dataDir, err) // Wrap with path context
	}

	switch cfg.OutputFormat { // Dispatch on format string -- empty string defaults to CSV
	case "sqlite": // SQLite backend: single persistent database file
		writer, err := newSQLiteWriter(dataDir) // Open or create the SQLite database
		if err != nil {                         // Fail early if the DB cannot be opened
			return nil, fmt.Errorf("create sqlite writer: %w", err) // Wrap for context
		}
		slog.Debug("SQLite writer ready", "dataDir", dataDir) // Log after successful writer creation
		return writer, nil                                     // Return the live SQLite writer

	case "csv", "": // CSV backend: one file per Write call (empty string is the default)
		slog.Debug("CSV writer ready", "dataDir", dataDir) // Log after construction
		return newCSVWriter(dataDir), nil                   // CSV writer needs no initialisation error

	default: // Unknown format string -- fail immediately so the misconfiguration is obvious
		return nil, fmt.Errorf("unknown output format %q -- supported: csv, sqlite", cfg.OutputFormat)
	}
}

// ── Shared helpers ───────────────────────────────────────────────────────────

// collectSortedKeys extracts the keys from a flat map and returns them sorted alphabetically.
// Sorted headers produce deterministic CSV column order and stable SQLite DDL across runs.
func collectSortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m)) // Pre-allocate with capacity hint to avoid reallocations
	for k := range m {                // Collect every key from the flat map
		keys = append(keys, k) // Append key to the slice for sorting
	}
	slices.Sort(keys) // Sort alphabetically -- consistent order across calls makes diffs readable
	return keys       // Return the sorted key slice to the caller
}

// buildRow extracts column values from flat in the order specified by headers,
// converting each value to its string representation for CSV output.
// Missing keys produce an empty string so every row has the same column count as the header.
func buildRow(flat map[string]any, headers []string) []string {
	row := make([]string, len(headers)) // Allocate one string slot per column
	for i, header := range headers {   // Iterate in header order to maintain column alignment
		if v, ok := flat[header]; ok { // Only populate the slot if the key is present
			row[i] = fmt.Sprintf("%v", v) // Convert any scalar type to its string representation
		}
		// Missing key: row[i] stays "" (zero value) -- produces an empty CSV cell
	}
	return row // Return the fully populated row in header column order
}

// buildArgs extracts column values from flat in the order specified by headers,
// returning []any for use as SQLite bound parameters (preserves original types).
// Missing keys produce an empty string rather than nil to avoid NULL column values.
func buildArgs(flat map[string]any, headers []string) []any {
	args := make([]any, len(headers)) // Allocate one any slot per column for parameter binding
	for i, header := range headers {  // Iterate in header order to match INSERT column list
		if v, ok := flat[header]; ok { // Use the actual value when the key is present
			args[i] = v // Pass the original typed value -- SQLite will coerce to TEXT
		} else {
			args[i] = "" // Use empty string for missing columns -- avoids NULL ambiguity
		}
	}
	return args // Return args in the same column order as the INSERT statement
}

// sanitizeTableName replaces any non-alphanumeric character with "_" so the
// endpoint name is safe to use as a SQLite table identifier.
func sanitizeTableName(endpoint string) string {
	return nonAlphanumericRE.ReplaceAllString(endpoint, "_") // Replace special chars with underscores
}

// buildInsertSQL constructs an INSERT statement for table with the given column headers,
// using the conflict resolution clause determined by strategy.
func buildInsertSQL(table string, headers []string, strategy EndpointStrategy) string {
	conflict := insertConflictClause(strategy) // Determine OR REPLACE vs OR IGNORE based on PK type
	cols := quoteColumns(headers)              // Quote each column name to prevent injection via field names
	placeholders := buildPlaceholders(len(headers)) // Generate one ? per column for safe parameter binding
	return fmt.Sprintf(
		"INSERT %s INTO %q (%s) VALUES (%s)", // table name is double-quoted for safety
		conflict,
		table,
		strings.Join(cols, ", "),          // Join quoted column names with comma-space
		placeholders,                      // Join ? placeholders with comma-space
	)
}

// insertConflictClause returns "OR REPLACE" for natural/composite PKs (upsert semantics)
// and "OR IGNORE" for auto-increment PKs (append-only -- never overwrites existing rows).
func insertConflictClause(strategy EndpointStrategy) string {
	switch strategy.Type { // Dispatch on the registered PK strategy type
	case PKTypeNatural, PKTypeComposite:
		return "OR REPLACE" // Upsert: newer API data overwrites the stale row with the same PK
	default:
		return "OR IGNORE" // Auto-increment: never overwrite; just skip if a duplicate key appears
	}
}

// buildPlaceholders returns a comma-separated string of count "?" bind parameters
// for use in the VALUES clause of a parameterised INSERT statement.
func buildPlaceholders(count int) string {
	parts := make([]string, count) // Allocate one slot per column
	for i := range parts {         // Fill each slot with the SQL bind-parameter marker
		parts[i] = "?" // Each ? corresponds to one bound argument passed to ExecContext
	}
	return strings.Join(parts, ", ") // Join with comma-space to match the INSERT VALUES syntax
}

// quoteColumns returns a new slice where each column name is double-quoted
// using Go's %q verb, which produces valid SQLite identifier quoting.
func quoteColumns(headers []string) []string {
	quoted := make([]string, len(headers)) // Allocate output slice with the same length as headers
	for i, h := range headers {            // Iterate every header to apply identifier quoting
		quoted[i] = fmt.Sprintf("%q", h) // Double-quote each name -- safe for identifiers with spaces
	}
	return quoted // Return the fully-quoted column name slice
}
