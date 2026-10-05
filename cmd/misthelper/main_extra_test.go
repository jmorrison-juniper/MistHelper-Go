// Package main -- extra tests covering registerStubs, makeStubHandler,
// runOrDispatch, and shutdown without invoking main() or os.Exit.
package main

import (
	"bufio"         // for bufio.NewReader -- wraps stdin substitutes for Dispatcher creation
	"context"       // for context.WithCancel and context.Background in all tests
	"errors"        // for sentinel error assertions in handler tests
	"io"            // for io.Discard -- throwaway terminal writer in dispatcher construction
	"os"            // for os.WriteFile -- creates a file named "data" to trigger MkdirAll error
	"path/filepath" // for filepath.Join -- builds cross-platform paths in initPackages tests
	"strings"       // for strings.NewReader -- provides empty stdin in dispatcher tests
	"testing"       // for testing.T -- standard test runner
	"time"          // for time.AfterFunc -- safety timeouts in goroutine-based tests

	"github.com/jmorrison-juniper/misthelper-go/internal/api"            // for api.Config used by newTestAppPackages helper
	mmenu "github.com/jmorrison-juniper/misthelper-go/internal/menu"     // aliased mmenu to avoid conflict if menu were in main package
	moutput "github.com/jmorrison-juniper/misthelper-go/internal/output" // aliased moutput to avoid conflict
	mssh "github.com/jmorrison-juniper/misthelper-go/internal/ssh"       // aliased mssh to match main.go conventions
	mweb "github.com/jmorrison-juniper/misthelper-go/internal/web"       // aliased mweb to match main.go conventions
)

// ── No-op writer ─────────────────────────────────────────────────────────────

// mainNoopWriter satisfies output.Writer with no side effects.
// Used in tests that don't want to write CSV or SQLite output.
type mainNoopWriter struct{}

// Write discards all records so tests produce no output artifacts.
func (mainNoopWriter) Write(_ context.Context, _ string, _ []map[string]any) error {
	return nil // Discard all records -- output correctness is not tested here
}

// Close is a no-op -- no real backend was opened.
func (mainNoopWriter) Close() error {
	return nil // Nothing to close -- no real backend was opened
}

// compile-time check that mainNoopWriter satisfies the Writer interface.
var _ moutput.Writer = mainNoopWriter{}

// testInventoryClient is a controllable stub for option-26 handler tests.
type testInventoryClient struct {
	records []map[string]any // Inventory records returned by GetOrgInventory on success
	err     error            // Optional error returned by GetOrgInventory
}

// GetOrgInventory satisfies inventoryClient for handler tests.
func (c testInventoryClient) GetOrgInventory(_ context.Context) ([]map[string]any, error) {
	if c.err != nil { // Simulate API-layer failure when configured by test
		return nil, c.err // Propagate configured error
	}
	return c.records, nil // Return configured inventory records for success path assertions
}

// recordingWriter captures endpoint keys and row counts for assertion in handler tests.
type recordingWriter struct {
	endpoint string // Last endpoint key passed to Write
	rows     int    // Number of rows passed to Write
	err      error  // Optional write error injected by test
}

// Write records endpoint usage and returns configured error when set.
func (w *recordingWriter) Write(_ context.Context, endpoint string, records []map[string]any) error {
	w.endpoint = endpoint // Capture endpoint key for strategy-routing assertions
	w.rows = len(records) // Capture row count for success-path assertions
	return w.err          // Return injected error when test needs writer-failure path
}

// Close satisfies output.Writer with no-op behavior for tests.
func (w *recordingWriter) Close() error {
	return nil // No resources to release in recording writer
}

// ── Test helpers ──────────────────────────────────────────────────────────────

// newTestDispatcher creates a minimal Dispatcher wired with an empty reader and
// discard writer. No operations are pre-registered; callers may register stubs.
func newTestDispatcher(r *mmenu.Registry) *mmenu.Dispatcher {
	return mmenu.NewDispatcher( // Wire a Dispatcher with test-safe I/O
		r,                                      // Registry provided by caller -- allows pre-registration of stubs
		bufio.NewReader(strings.NewReader("")), // Empty reader simulates EOF immediately -- no blocking
		io.Discard,                             // Terminal writer discards all output -- tests need not inspect it
		mainNoopWriter{},                       // No-op output writer -- tests produce no CSV/SQLite files
	)
}

// newTestAppPackages creates a minimal appPackages value for shutdown tests.
// The SSH server is initialised with a fresh host key stored in a temp directory.
func newTestAppPackages(t *testing.T) appPackages {
	t.Helper()                                      // Mark as helper for clean failure attribution
	keyDir := t.TempDir()                           // Isolated temp dir so each test has its own host key
	signer, err := mssh.LoadOrCreateHostKey(keyDir) // Generate a test-only RSA host key
	if err != nil {                                 // Key generation failure makes the test invalid
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail immediately with the error detail
	}
	cfg := api.Config{SSHPort: 0, WebPort: 0}                  // Zero ports -- servers are not started in these tests
	r := mmenu.NewRegistry()                                   // Empty registry -- no operations needed for shutdown tests
	sshSrv := mssh.NewServer(cfg, signer, r, mainNoopWriter{}) // SSH server to test shutdown
	webSrv := mweb.NewServer(cfg)                              // Web server to test shutdown
	return appPackages{                                        // Populate the appPackages struct
		cfg:       cfg,    // Minimal config (not used by shutdown directly)
		registry:  r,      // Empty registry (not used by shutdown)
		sshServer: sshSrv, // Real SSH server -- Shutdown waits on wg then returns nil
		webServer: webSrv, // Real web server -- Shutdown on unstarted server returns nil
	}
}

// ── registerStubs tests ───────────────────────────────────────────────────────

// TestRegisterStubs_RegistersAllOps verifies that registerStubs populates the registry
// with every operation from the stubOps table and that each entry is retrievable.
func TestRegisterStubs_RegistersAllOps(t *testing.T) {
	t.Parallel()             // Independent of all other tests
	r := mmenu.NewRegistry() // Empty registry to populate
	registerStubs(r, nil)    // Register stubs with no inventory client (all remain placeholders)

	sorted := r.Sorted()             // Retrieve all registered entries
	if len(sorted) != len(stubOps) { // Count must match the stubOps table
		t.Errorf("registry has %d entries; want %d (len(stubOps))", len(sorted), len(stubOps)) // Report mismatch
	}
	// Spot-check: the first stub op must be findable by its number
	if len(stubOps) > 0 { // Guard against an empty table (would be a configuration error)
		firstOp := stubOps[0]         // First entry in the stub table
		entry, ok := r.Get(firstOp.n) // Look up by operation number
		if !ok {                      // Entry must be present after registerStubs
			t.Errorf("entry %d (%s) not found in registry after registerStubs", firstOp.n, firstOp.name) // Report missing entry
		}
		if entry.Number != firstOp.n { // Entry number must match the stub table
			t.Errorf("entry.Number = %d; want %d", entry.Number, firstOp.n) // Report wrong number
		}
	}
}

// TestRegisterStubs_NoOps verifies that registerStubs does not panic when called
// multiple times on the same registry (idempotent re-registration path).
func TestRegisterStubs_NoOps(t *testing.T) {
	t.Parallel()             // Independent of all other tests
	r := mmenu.NewRegistry() // Fresh empty registry
	registerStubs(r, nil)    // First registration -- all entries added
	registerStubs(r, nil)    // Second registration -- must not panic or corrupt state
}

// ── makeStubHandler tests ─────────────────────────────────────────────────────

// TestMakeStubHandler_ReturnsNil verifies that a stub handler returns nil so the
// interactive menu loop continues after a stub operation is invoked.
func TestMakeStubHandler_ReturnsNil(t *testing.T) {
	t.Parallel()                                                                                               // Independent of all other tests
	handler := makeStubHandler(99, "Test Stub")                                                                // Create a stub for an arbitrary operation number
	err := handler(context.Background(), bufio.NewReader(strings.NewReader("")), io.Discard, mainNoopWriter{}) // Invoke the handler
	if err != nil {                                                                                            // Stub handlers must never return an error
		t.Errorf("makeStubHandler returned error: %v; want nil", err) // Report unexpected error
	}
}

// TestMakeStubHandler_DifferentNumbers verifies that two different stub handlers can be
// created without conflict (closure captures are per-handler, not shared).
func TestMakeStubHandler_DifferentNumbers(t *testing.T) {
	t.Parallel()                                                                                           // Independent of all other tests
	h1 := makeStubHandler(1, "Op One")                                                                     // Handler for operation 1
	h2 := makeStubHandler(2, "Op Two")                                                                     // Handler for operation 2 (closure must capture independently)
	err1 := h1(context.Background(), bufio.NewReader(strings.NewReader("")), io.Discard, mainNoopWriter{}) // Invoke h1
	err2 := h2(context.Background(), bufio.NewReader(strings.NewReader("")), io.Discard, mainNoopWriter{}) // Invoke h2
	if err1 != nil || err2 != nil {                                                                        // Both handlers must return nil
		t.Errorf("stub handlers returned errors: h1=%v h2=%v; want both nil", err1, err2) // Report any error
	}
}

// ── runOrDispatch tests ───────────────────────────────────────────────────────

// TestRunOrDispatch_ZeroQuits verifies that menuNum == 0 returns nil immediately
// without touching the dispatcher (the explicit quit / smoke-test invocation).
func TestRunOrDispatch_ZeroQuits(t *testing.T) {
	t.Parallel()                                       // Independent of all other tests
	err := runOrDispatch(context.Background(), nil, 0) // menuNum==0 must not dereference dispatcher
	if err != nil {                                    // Clean quit must return nil
		t.Errorf("runOrDispatch(0) returned error: %v; want nil", err) // Report unexpected error
	}
}

// TestRunOrDispatch_NegativeWithCancelledContext verifies that menuNum < 0 returns nil
// when the context is already cancelled (covers the headless <-ctx.Done() path).
// In the test runner stdin may or may not be a terminal, but both paths return nil
// when the context is cancelled before any blocking read can occur.
func TestRunOrDispatch_NegativeWithCancelledContext(t *testing.T) {
	t.Parallel()                                            // Independent of all other tests
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context for clean exit
	cancel()                                                // Pre-cancel so neither d.Run nor <-ctx.Done blocks

	r := mmenu.NewRegistry()                          // Empty registry -- no operations needed
	d := newTestDispatcher(r)                         // Dispatcher with empty stdin (EOF on first read)
	done := make(chan error, 1)                       // Buffered so the goroutine never blocks on send
	go func() { done <- runOrDispatch(ctx, d, -1) }() // Run in goroutine -- might block briefly on terminal check

	select {
	case err := <-done: // runOrDispatch must complete in under 3 seconds
		if err != nil { // Must return nil (clean headless exit)
			t.Errorf("runOrDispatch(-1) returned error: %v; want nil", err) // Report unexpected error
		}
	case <-time.After(3 * time.Second): // Safety net: fail if the function blocks
		t.Error("runOrDispatch(-1) did not return within 3 seconds; check terminal detection or context handling")
	}
}

// TestRunOrDispatch_PositiveDispatchKnownOp verifies that menuNum > 0 dispatches a known
// operation. After registerStubs the first stubOp is registered, so Dispatch must succeed.
func TestRunOrDispatch_PositiveDispatchKnownOp(t *testing.T) {
	t.Parallel()           // Independent of all other tests
	if len(stubOps) == 0 { // Guard against an empty stub table (config error)
		t.Skip("stubOps is empty; skipping dispatch test") // Skip rather than fail -- not the test's fault
	}
	r := mmenu.NewRegistry()                             // Fresh registry to populate
	registerStubs(r, nil)                                // Register all stubs so the target operation is present
	d := newTestDispatcher(r)                            // Dispatcher wired to empty reader and discard writer
	opNum := stubOps[0].n                                // Use the first registered stub op (reliable to exist)
	err := runOrDispatch(context.Background(), d, opNum) // Dispatch the known operation
	if err != nil {                                      // Stub handlers return nil so dispatch must succeed
		t.Errorf("runOrDispatch(%d) returned error: %v; want nil", opNum, err) // Report unexpected error
	}
}

// TestRegisterStubs_Option26UsesInventoryHandler verifies that option 26 is wired to real logic when client exists.
func TestRegisterStubs_Option26UsesInventoryHandler(t *testing.T) {
	t.Parallel()                                                          // Safe to run concurrently with other in-memory tests
	registry := mmenu.NewRegistry()                                       // Fresh registry for this test only
	client := testInventoryClient{records: []map[string]any{{"id": "a"}}} // Stub API client returning one inventory row
	registerStubs(registry, client)                                       // Register handlers with option-26 override enabled
	entry, ok := registry.Get(26)                                         // Resolve operation 26 entry from registry
	if !ok {                                                              // Option 26 must exist in stubOps table
		t.Fatal("operation 26 not found in registry") // Fail fast if registration broke
	}
	writer := &recordingWriter{}                                                                           // Capture writer endpoint invoked by handler
	err := entry.Handler(context.Background(), bufio.NewReader(strings.NewReader("")), io.Discard, writer) // Execute handler directly for deterministic assertion
	if err != nil {                                                                                        // Success path should not return error
		t.Fatalf("option 26 handler returned unexpected error: %v", err) // Report unexpected handler failure
	}
	if writer.endpoint != "getOrgInventory" { // Endpoint key must match strategy registration requirement
		t.Errorf("option 26 writer endpoint = %q; want %q", writer.endpoint, "getOrgInventory") // Report endpoint routing regression
	}
	if writer.rows != 1 { // Writer should receive exactly one row from stub client response
		t.Errorf("option 26 writer rows = %d; want 1", writer.rows) // Report row-count mismatch
	}
}

// TestRegisterStubs_Option26PropagatesFetchError verifies deterministic failure propagation for direct mode.
func TestRegisterStubs_Option26PropagatesFetchError(t *testing.T) {
	t.Parallel()                                            // Safe to run concurrently with other in-memory tests
	registry := mmenu.NewRegistry()                         // Fresh registry for this test only
	want := errors.New("inventory fetch boom")              // Sentinel error for assertion
	registerStubs(registry, testInventoryClient{err: want}) // Wire handler with failing client
	entry, ok := registry.Get(26)                           // Resolve option 26 handler
	if !ok {                                                // Must be present for direct dispatch path
		t.Fatal("operation 26 not found in registry") // Fail fast if registration broke
	}
	err := entry.Handler(context.Background(), bufio.NewReader(strings.NewReader("")), io.Discard, &recordingWriter{}) // Execute handler directly
	if !errors.Is(err, want) {                                                                                         // Error chain must preserve API failure cause
		t.Errorf("option 26 handler error = %v; want sentinel %v", err, want) // Report mismatch for deterministic status propagation
	}
}

// TestRunOrDispatch_PositiveDispatchUnknownOp verifies that dispatching an unregistered
// operation number returns an error (exercises the Dispatch error-return path).
func TestRunOrDispatch_PositiveDispatchUnknownOp(t *testing.T) {
	t.Parallel()                                        // Independent of all other tests
	r := mmenu.NewRegistry()                            // Empty registry -- no operations registered
	d := newTestDispatcher(r)                           // Dispatcher with no known operations
	err := runOrDispatch(context.Background(), d, 9999) // Dispatch an unregistered operation number
	if err == nil {                                     // Must return an error -- operation is not in registry
		t.Error("runOrDispatch(9999) returned nil; want error for unregistered operation") // Report missing error
	}
}

// ── shutdown tests ────────────────────────────────────────────────────────────

// TestShutdown_CompletesWithoutPanic verifies that shutdown completes cleanly for
// initialised but not-yet-started SSH and web servers. Both Shutdown calls return
// nil when no sessions or HTTP requests are active.
func TestShutdown_CompletesWithoutPanic(t *testing.T) {
	t.Parallel()                                // Independent of all other tests
	pkgs := newTestAppPackages(t)               // Create initialised (but unstarted) servers
	done := make(chan struct{})                 // Channel to detect completion
	go func() { shutdown(pkgs); close(done) }() // Run shutdown in a goroutine for timeout safety

	select {
	case <-done: // Shutdown must complete within the timeout
	case <-time.After(10 * time.Second): // Safety net: SSH timeout is 30 s; web is 5 s -- give 10 s total
		t.Error("shutdown did not complete within 10 seconds; possible deadlock in Shutdown method")
	}
}

// ── initPackages tests ────────────────────────────────────────────────────────

// TestInitPackages_MkdirAllError verifies that initPackages returns an error when
// the "data" directory cannot be created because a file already exists with that name.
// This test must NOT be parallel because it changes the working directory.
func TestInitPackages_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()                                           // Isolated writable directory for this test
	t.Chdir(tmpDir)                                                 // Change CWD so "data" is relative to tmpDir
	dataPath := filepath.Join(tmpDir, "data")                       // Absolute path for the blocking file
	if err := os.WriteFile(dataPath, []byte{}, 0o600); err != nil { // Create a plain file named "data"
		t.Fatalf("WriteFile: %v", err) // Bail if we can't even set up the blocker
	}
	_, err := initPackages("") // MkdirAll("data") fails -- "data" is a file, not a dir
	if err == nil {            // Must return an error
		t.Error("initPackages succeeded; expected error because \"data\" is a regular file")
	}
}

// TestInitPackages_LoadConfigError verifies that initPackages returns an error when
// MIST_API_TOKEN is not set. Covers the MkdirAll success path and LoadConfig failure.
// This test must NOT be parallel because it changes the working directory and env vars.
func TestInitPackages_LoadConfigError(t *testing.T) {
	t.Chdir(t.TempDir())           // Writable CWD so MkdirAll("data") succeeds
	t.Setenv("MIST_API_TOKEN", "") // Ensure token is missing -- triggers LoadConfig error
	t.Setenv("MIST_ORG_ID", "")    // Clear OrgID as well for cleanliness
	_, err := initPackages("")     // MkdirAll succeeds; LoadConfig fails -- no token
	if err == nil {                // Must fail with a descriptive error
		t.Error("initPackages succeeded; expected LoadConfig error with empty MIST_API_TOKEN")
	}
}

// TestInitPackages_FullSuccess verifies the complete happy path of initPackages.
// Uses a writable temp dir and fake (but structurally valid) credentials so every
// line of the function is exercised without touching the real Mist API.
// This test must NOT be parallel because it changes the working directory and env vars.
func TestInitPackages_FullSuccess(t *testing.T) {
	t.Chdir(t.TempDir())                                            // Writable CWD for MkdirAll and host key generation
	t.Setenv("MIST_API_TOKEN", "fake-token-for-testing")            // Required by LoadConfig (never used for real API calls)
	t.Setenv("MIST_ORG_ID", "00000000-0000-0000-0000-000000000000") // Valid UUID format satisfies validation
	t.Setenv("SSH_PASSWORD", "test-ssh-password")                   // Required by LoadConfig -- test value only
	t.Setenv("OUTPUT_FORMAT", "")                                   // Clear so --format "csv" flag takes precedence
	pkgs, err := initPackages("csv")                                // Full initialisation with CSV output
	if err != nil {                                                 // All steps should succeed with fake creds
		t.Fatalf("initPackages returned error: %v", err) // Fail fast with the exact error
	}
	if err := pkgs.writer.Close(); err != nil { // Clean up the output writer to flush any buffered state
		t.Errorf("writer.Close returned error: %v", err) // Report but don't fatal -- main cleanup path
	}
}

// ── startServers tests ────────────────────────────────────────────────────────

// TestStartServers_LaunchesAndStops verifies that startServers launches both background
// goroutines without panicking and that the servers stop cleanly after shutdown.
// Uses port 0 so the OS assigns free ports -- no hard-coded port conflicts.
// This test must NOT be parallel because it starts real network listeners.
func TestStartServers_LaunchesAndStops(t *testing.T) {
	pkgs := newTestAppPackages(t)                           // Create initialised servers on port 0 (OS-assigned)
	ctx, cancel := context.WithCancel(context.Background()) // Cancellable context to stop the SSH server

	startServers(ctx, pkgs)            // Must not panic -- launches two goroutines
	time.Sleep(100 * time.Millisecond) // Give goroutines time to reach their accept/listen loop

	cancel()                                                                              // Signal SSH server to stop via ctx.Done()
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second) // Budget for web server shutdown
	defer cleanupCancel()                                                                 // Release timeout resources when test ends
	_ = pkgs.webServer.Shutdown(cleanupCtx)                                               // Stop the web server so its goroutine exits cleanly
	time.Sleep(150 * time.Millisecond)                                                    // Wait for goroutines to exit before test cleanup
}

// TestStartServers_GoroutineErrorPaths covers the slog.Error branches inside the
// background goroutines by starting servers with an invalid port (-1) that forces
// an immediate bind error. Both goroutines hit their error-logging path.
// This test must NOT be parallel because it starts real (failing) network listeners.
func TestStartServers_GoroutineErrorPaths(t *testing.T) {
	keyDir := t.TempDir()                           // Temp dir for host key generation
	signer, err := mssh.LoadOrCreateHostKey(keyDir) // Need a valid signer even for a failing server
	if err != nil {                                 // Generation failure makes the test setup invalid
		t.Fatalf("LoadOrCreateHostKey: %v", err) // Bail with the setup error
	}
	invalidCfg := api.Config{SSHPort: -1, WebPort: -1}                // Negative ports are always rejected by the kernel
	r := mmenu.NewRegistry()                                          // Empty registry -- stubs not needed for this test
	sshSrv := mssh.NewServer(invalidCfg, signer, r, mainNoopWriter{}) // Server that will fail to bind on port -1
	webSrv := mweb.NewServer(invalidCfg)                              // Web server that will fail to bind on port -1
	pkgs := appPackages{cfg: invalidCfg, registry: r, sshServer: sshSrv, webServer: webSrv}

	ctx, cancel := context.WithCancel(context.Background()) // Context for the SSH server goroutine
	defer cancel()                                          // Release context when test exits

	startServers(ctx, pkgs)            // Launch goroutines -- both will immediately hit bind error
	time.Sleep(200 * time.Millisecond) // Wait for goroutines to run, fail, and log the error
}

// TestInitPackages_InvalidFormatFails verifies that initPackages returns an error when
// an unsupported output format is specified. Covers the output.NewWriter error path.
// This test must NOT be parallel because it changes the working directory and env vars.
func TestInitPackages_InvalidFormatFails(t *testing.T) {
	t.Chdir(t.TempDir())                                            // Writable CWD so MkdirAll succeeds
	t.Setenv("MIST_API_TOKEN", "fake-token-for-testing")            // Required by LoadConfig
	t.Setenv("MIST_ORG_ID", "00000000-0000-0000-0000-000000000000") // Valid UUID format
	t.Setenv("SSH_PASSWORD", "test-ssh-password")                   // Required by LoadConfig -- test value only
	_, err := initPackages("unsupported_format")                    // NewWriter rejects unknown formats
	if err == nil {                                                 // Must return an error for the bad format
		t.Error("initPackages succeeded; expected error for unsupported output format")
	}
}

// ── runMain tests ────────────────────────────────────────────────────────────

// TestRunMain_Version verifies that --version exits cleanly without requiring env vars.
// The version flag should short-circuit before initPackages or any server startup.
func TestRunMain_Version(t *testing.T) {
	t.Parallel()                          // Safe to run concurrently; no shared state is modified
	err := runMain([]string{"--version"}) // Version branch should return nil immediately
	if err != nil {                       // Version must not fail or try to load config
		t.Errorf("runMain(--version) returned error: %v", err) // Report unexpected error
	}
}

// TestRunMain_Menu0Quit verifies that --menu 0 runs the full init path and exits cleanly.
// This covers the orchestration path in runMain without entering the interactive menu loop.
func TestRunMain_Menu0Quit(t *testing.T) {
	t.Chdir(t.TempDir())                                            // Isolate filesystem side effects under a scratch directory
	t.Setenv("MIST_API_TOKEN", "fake-token-for-testing")            // Satisfy LoadConfig without touching real credentials
	t.Setenv("MIST_ORG_ID", "00000000-0000-0000-0000-000000000000") // Valid UUID format keeps LoadConfig happy
	t.Setenv("SSH_PASSWORD", "test-ssh-password")                   // Required by LoadConfig -- test value only
	t.Setenv("SSH_PORT", "0")                                       // Let the OS pick an available port for SSH
	t.Setenv("WEB_PORT", "0")                                       // Let the OS pick an available port for HTTP
	err := runMain([]string{"--menu", "0"})                         // Clean-quit path should initialize and shut down without error
	if err != nil {                                                 // runMain should return nil on menu 0
		t.Errorf("runMain(--menu 0) returned error: %v", err) // Report unexpected error
	}
}

// TestRunMain_InitError verifies that runMain returns an error when configuration loading fails.
// This ensures the top-level error path is still observable after the refactor.
func TestRunMain_InitError(t *testing.T) {
	t.Chdir(t.TempDir())                    // Isolate filesystem side effects under a scratch directory
	t.Setenv("MIST_API_TOKEN", "")          // Missing token should trigger LoadConfig failure
	t.Setenv("MIST_ORG_ID", "")             // Missing org ID keeps the test focused on the init error path
	err := runMain([]string{"--menu", "0"}) // Any mode still hits initPackages before dispatch
	if err == nil {                         // Must return an error for missing configuration
		t.Error("runMain returned nil error with missing env vars; want init failure")
	}
}
