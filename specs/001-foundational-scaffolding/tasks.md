# Tasks: Foundational Go Scaffolding

**Feature Branch**: `feat/001-foundational-scaffolding`
**Spec**: [spec.md](spec.md)
**Plan**: [plan.md](plan.md)
**Created**: 2026-05-22
**Status**: Ready for Implementation

---

## Overview

| Metric | Value |
| - | - |
| Total tasks | 28 |
| Phase 1 — Setup | 1 |
| Phase 2 — Foundational | 4 |
| Phase 3 — US1: API Client | 4 |
| Phase 4 — US2: Output Writing | 4 |
| Phase 5 — US3: Menu Dispatch | 5 |
| Phase 6 — US4: SSH Server | 4 |
| Phase 7 — US5: Web UI Skeleton | 3 |
| Phase 8 — Wiring | 1 |
| Phase 9 — Quality & Docs | 2 |

---

## Dependencies Graph

```text
T001 (deps)
  └── T002 (config) ──────────────────────────────────────────────────────────┐
        ├── T003 (retry)                                                       │
        │     └── T006 (client) ─────── T007 (paginate) ── [US1 complete]    │
        ├── T004 (strategies) ─── T010 (sqlite) ─── [US2 complete] ──────────┤
        ├── T005 (flatten) ──── T009 (csv)                                    │
        │                             └── T011 (writer)                       │
        ├── T012 (input) ─── T013 (dispatcher) ─── T014 (display)            │
        │                          └── T015 (loop) ── [US3 complete] ─────────┤
        ├── T016 (auth) ─── T018 (session) ─── T019 (handler) ─── T020 (ssh-server) ── [US4 complete] ──┤
        └── T021 (web-handlers) ─── T022 (routes) ─── T023 (web-server) ── [US5 complete] ──────────────┤
                                                                               └── T024 (main.go)
                                                                                     └── T025 (quality)
                                                                                           └── T026 (docs)
```

**Parallel opportunities**:

- After T002 is complete: US1 (T003→T006→T007), US2 (T004→T005→T009→T010→T011), US3 (T012→T013→T014→T015), US4 (T016→T018→T019→T020), and US5 (T021→T022→T023) can all proceed in parallel.
- Within US2: T009 (csv.go) and T010 (sqlite.go) can proceed in parallel after T004 and T005 are done.

**Suggested MVP scope**: Complete Phase 1, 2, 3, and 4 (US1 + US2) to achieve a working API client with data persistence. US3–US5 can follow in any order.

---

## Phase 1: Setup

- [ ] T001 Add `modernc.org/sqlite` and `golang.org/x/crypto` to `go.mod` via `go get modernc.org/sqlite && go get golang.org/x/crypto/ssh && go mod tidy`; verify `go build ./...` compiles with zero errors

---

## Phase 2: Foundational

*These four files are prerequisites for all user stories. Complete them before starting any US phase.*

- [ ] T002 Create `internal/api/config.go` with `Config` struct (fields: `APIToken`, `OrgID`, `OutputFormat`, `RateLimitMs`, `SSHPort`, `SSHUser`, `SSHPassword`, `WebPort`) and `LoadConfig(format string) (Config, error)` that reads env vars, applies CLI format-flag override, validates required fields, and returns a descriptive error for missing `MIST_API_TOKEN` or `MIST_ORG_ID`; add `TestLoadConfig_ValidEnv`, `TestLoadConfig_MissingToken`, `TestLoadConfig_MissingOrgID`, `TestLoadConfig_CLIFormatOverridesEnv`, `TestLoadConfig_Defaults` in `internal/api/config_test.go`

- [ ] T003 Create `internal/api/retry.go` with `RetryConfig` struct (max attempts, base delay, max delay) and `withRetry(ctx context.Context, op func() error, cfg RetryConfig) error` implementing exponential backoff with jitter for HTTP 429/5xx; log each retry at info level without logging response bodies; add `TestWithRetry_SuccessFirstAttempt`, `TestWithRetry_SuccessAfterRetry`, `TestWithRetry_ExhaustsRetries`, `TestWithRetry_NonRetryableError` in `internal/api/retry_test.go`

- [ ] T004 Create `internal/output/strategies.go` with `PKType` string type, `Strategy` struct (Type, PrimaryKey, Indexes, Description), `StrategyMap` package-level var containing all ~65 entries ported from `ENDPOINT_PRIMARY_KEY_STRATEGIES` in `MistHelper.py` (including `timeseries_pk` entries and a `"default"` auto-increment entry), and `StrategyFor(endpointName string) Strategy` returning the named strategy or default with a warn-level log; add `TestStrategyFor_KnownEndpoint`, `TestStrategyFor_UnknownEndpoint_FallsBackToDefault`, `TestStrategyMap_AllPKTypesRepresented` in `internal/output/strategies_test.go`

- [ ] T005 Create `internal/output/flatten.go` with `flattenRow(row map[string]any, prefix string) map[string]string` that recursively flattens nested maps using dot notation, converts non-map values with `fmt.Sprintf("%v", v)`, enforces a max recursion depth of 10 with a warn log on breach; add `TestFlattenRow_Flat`, `TestFlattenRow_Nested`, `TestFlattenRow_NilValue`, `TestFlattenRow_MaxDepth` in `internal/output/flatten_test.go`

---

## Phase 3: User Story 1 — API Client Initialization (P1)

*Goal*: The tool connects to Mist Cloud using `.env` credentials, validates the token and org ID, and confirms the session is ready before the main menu appears.

*Independent test criterion*: `go test ./internal/api/...` passes with a mocked transport — no live API calls required.

- [X] T006 [US1] Create `internal/api/client.go` with `Client` interface (methods: `ListOrgSites`, `ListSiteDevices`, `GetOrgInventory`, each taking `context.Context` and returning `([]map[string]any, error)`), `mistClient` concrete struct embedding the `mistapi-go` SDK client (unexported `sdk` field), and `New(cfg Config) (Client, error)` constructor that validates the token, calls `mistapi.NewClient()`, and returns a descriptive error for missing token; log every API call at debug level before the call and result row count at debug level after; never log the token value; add `TestNew_MissingToken`, `TestNew_ValidConfig` in `internal/api/client_test.go`

- [ ] T007 [US1] Create `internal/api/paginate.go` with `PageFunc[T]` type alias `func(page, limit int) ([]T, bool, error)`, `Paginate[T any](ctx context.Context, fn PageFunc[T], cfg Config) ([]T, error)` that loops calling `fn`, appends results, sleeps `cfg.RateLimitMs` milliseconds between pages via `waitRateLimit(ms int)` helper, and logs page number at debug level each iteration; add `TestPaginate_SinglePage`, `TestPaginate_MultiPage`, `TestPaginate_ErrorOnPage2` in `internal/api/paginate_test.go`

- [ ] T008 [US1] Implement `ListOrgSites`, `ListSiteDevices`, and `GetOrgInventory` on `mistClient` in `internal/api/client.go`: each method calls the corresponding `mistapi-go` SDK function wrapped in `withRetry`, flattens the SDK response via a `toRowSlice` helper, applies the fixed rate-limit delay via `Paginate` for multi-page calls, and logs the method name at info before and row count at debug after; add table-driven tests using a mock `Client` interface in `internal/api/client_test.go`

- [ ] T009 [P] [US1] Write acceptance scenario tests in `internal/api/client_test.go` asserting: (1) valid env constructs client with no error; (2) missing token returns descriptive error and exits non-zero; (3) paginated list returns merged result from all pages; (4) 429 response triggers retry with backoff; each test uses a mock HTTP transport (no live API calls)

---

## Phase 4: User Story 2 — Output Writing (P1)

*Goal*: Results are written to CSV or SQLite (selected by `OUTPUT_FORMAT`/`--format`) with upsert semantics for SQLite.

*Independent test criterion*: `go test ./internal/output/...` creates files in a temp dir and verifies row counts and upsert behavior.

- [ ] T010 [US2] Create `internal/output/csv.go` with `CSVWriter` struct and `Write(data []map[string]any, target, endpointName string) error`: creates `data/` directory with `os.MkdirAll` if absent, flattens each row via `flattenRow`, collects all unique keys sorted alphabetically for the header, writes via `encoding/csv`, flushes and closes the file, and logs the file path and row count at info level; add `TestCSVWriter_CreatesFile`, `TestCSVWriter_CorrectHeaders`, `TestCSVWriter_CreatesDataDir`, `TestCSVWriter_EmptyData` in `internal/output/csv_test.go`

- [ ] T011 [US2] Create `internal/output/sqlite.go` with `SQLiteWriter` struct and `Write(data []map[string]any, target, endpointName string) error`: opens `data/mist_data.db` via `database/sql` + `modernc.org/sqlite`, calls `ensureTable(db, tableName, cols, strategy)` to create the table if absent (adding `misthelper_internal_id INTEGER PRIMARY KEY AUTOINCREMENT` for auto-increment strategies), uses `INSERT OR REPLACE INTO` for `natural_pk` and `composite_pk` and plain `INSERT` for `auto_increment_with_unique`, sets SQLite busy-timeout to 5000ms, logs table name and row count at info; add `TestSQLiteWriter_CreatesTable`, `TestSQLiteWriter_NaturalPKUpsert`, `TestSQLiteWriter_CompositeUpsert`, `TestSQLiteWriter_AutoIncrementNoDuplicate` in `internal/output/sqlite_test.go`

- [ ] T012 [US2] Add `TestSQLiteWriter_1000RowsBenchmark` to `internal/output/sqlite_test.go` asserting that writing 1,000 rows with a natural-PK strategy completes in under 2 seconds (SC-005); run the benchmark and record the baseline duration in a comment

- [ ] T013 [US2] Create `internal/output/writer.go` with `Writer` interface exposing `Write(data []map[string]any, target, endpointName string) error` and `NewWriter(cfg api.Config) (Writer, error)` factory that switches on `cfg.OutputFormat`, logs a warn and falls back to CSV for unknown format values; add `TestNewWriter_CSV`, `TestNewWriter_SQLite`, `TestNewWriter_UnknownFormatFallsBackToCSV` in `internal/output/writer_test.go`

---

## Phase 5: User Story 3 — Interactive Menu Dispatch (P2)

*Goal*: Numbered menu appears, `Dispatch(1)` calls the registered handler, `Dispatch(0)` quits, EOF exits cleanly.

*Independent test criterion*: `go test ./internal/menu/...` with simulated stdin passes all dispatch, EOF, and unknown-option assertions.

- [ ] T014 [US3] Create `internal/menu/input.go` with sentinel errors `ErrEOF = errors.New("stdin closed")` and `ErrQuit = errors.New("user quit")`, `safeInput(reader *bufio.Reader, prompt, context string) (string, error)` printing the prompt and returning a trimmed string (or `ErrEOF` on `io.EOF` — no panic), and `parseSelection(s string) (int, error)` trimming whitespace and parsing via `strconv.Atoi`; add `TestSafeInput_Normal`, `TestSafeInput_EOF`, `TestSafeInput_Whitespace`, `TestParseSelection_Valid`, `TestParseSelection_Invalid` in `internal/menu/input_test.go`

- [ ] T015 [US3] Create `internal/menu/dispatcher.go` with `HandlerFunc` type `func(ctx context.Context) error`, `Dispatcher` interface (methods: `Register(n int, name string, handler HandlerFunc)`, `Dispatch(ctx context.Context, n int) error`, `RunInteractive(ctx context.Context) error`), and `menuDispatcher` struct holding `handlers map[int]entry`; `Register` panics on duplicate registration (programming error); `Dispatch` returns `ErrQuit` for n==0 and `ErrUnknown` for unregistered n; `New() Dispatcher` constructor; add `TestDispatch_ValidOption`, `TestDispatch_Quit`, `TestDispatch_Unknown`, `TestDispatch_HandlerError` in `internal/menu/dispatcher_test.go`

- [ ] T016 [US3] Create `internal/menu/display.go` with `printMenu(w io.Writer, handlers map[int]entry)` that prints a sorted table of registered options via `fmt.Fprintf(w, ...)` (not `fmt.Println`) and always appends `"  0. Exit"` last; add `TestPrintMenu_OutputContainsAllHandlers`, `TestPrintMenu_ZeroAlwaysLast` in `internal/menu/display_test.go`

- [ ] T017 [US3] Create `internal/menu/loop.go` with `RunInteractive(ctx context.Context) error` looping: check `ctx.Done()` → `printMenu` → `safeInput` → `parseSelection` → `Dispatch`; on `ErrEOF` log info and return nil; on `ErrQuit` log info and return nil; on `ErrUnknown` print user-friendly message and continue; add `TestRunInteractive_QuitOnZero`, `TestRunInteractive_EOFExitsCleanly`, `TestRunInteractive_UnknownOptionContinues` in `internal/menu/loop_test.go`

- [ ] T018 [P] [US3] Write acceptance scenario test in `internal/menu/dispatcher_test.go` asserting: (1) `--menu 11` direct invocation calls handler for 11 without showing menu; (2) unregistered number prints "unknown option" and does not exit; (3) EOF on simulated stdin exits cleanly with zero return code

---

## Phase 6: User Story 4 — SSH Server Access (P2)

*Goal*: SSH connection on port 2200 is accepted, isolated session directory created, ForceCommand invoked, directory cleaned up on disconnect.

*Independent test criterion*: `go test ./internal/ssh/...` asserts server starts, mock connection creates a session dir, and handler cleans up.

- [ ] T019 [US4] Create `internal/ssh/auth.go` with `loadOrGenerateHostKey(path string) (ssh.Signer, error)` reading an existing RSA PEM from `path` or generating RSA-2048, writing PEM to `path`, and logging the key fingerprint (not the key) at info level; `passwordCallback(user, password string, cfg api.Config) bool` comparing against `cfg.SSHUser`/`cfg.SSHPassword` without logging the password; key file path is `filepath.Join("data", "ssh_host_key")`; add `TestLoadOrGenerateHostKey_GeneratesIfMissing`, `TestLoadOrGenerateHostKey_LoadsExisting`, `TestPasswordCallback_Valid`, `TestPasswordCallback_Invalid` in `internal/ssh/auth_test.go`

- [ ] T020 [US4] Create `internal/ssh/session.go` with `Session` struct (fields: `ID string`, `ClientIP string`, `Dir string`, `StartTime time.Time`), `newSession(clientIP string) (Session, error)` generating an 8-byte hex ID via `crypto/rand` and creating `data/sessions/session_<ID>/` with `os.MkdirAll`, and `(s Session) Cleanup() error` calling `os.RemoveAll` and logging session ID, client IP, and duration at info level; add `TestNewSession_CreatesDirectory`, `TestSession_Cleanup_RemovesDirectory` in `internal/ssh/session_test.go`

- [ ] T021 [US4] Create `internal/ssh/handler.go` with `SessionHandler` type `func(ctx context.Context, sess Session)` and `forceCommandHandler(conn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request, sess Session, handler SessionHandler)` that rejects any channel type other than `session` and any request type granting shell access (`shell`, `exec` for arbitrary commands, `pty-req`), logs rejected shell attempts at warn level with session ID and client IP, and invokes `handler(ctx, sess)` for the ForceCommand flow; drains `reqs` and `chans` to prevent goroutine leaks; add `TestForceCommandHandler_RejectsShell`, `TestForceCommandHandler_InvokesHandler` in `internal/ssh/handler_test.go`

- [ ] T022 [US4] Create `internal/ssh/server.go` with `Server` struct, `New(cfg api.Config, handler SessionHandler) (*Server, error)` calling `loadOrGenerateHostKey`, building `ssh.ServerConfig` with the password callback, and `Start(ctx context.Context, wg *sync.WaitGroup) error` calling `net.Listen("tcp", addr)` (returning the error immediately if the port is already in use), accepting connections in a goroutine per connection via `wg.Add(1)` / `defer wg.Done()`, and stopping the listener on ctx cancel; log port at info on start; add `TestServer_StartsOnPort`, `TestServer_RejectsInvalidPassword`, `TestServer_SessionIsolation` in `internal/ssh/server_test.go`

---

## Phase 7: User Story 5 — Web UI Skeleton (P3)

*Goal*: `GET /` returns 200 with app name and version; `GET /health` returns `{"status":"ok"}`; unregistered paths return 404.

*Independent test criterion*: `go test ./internal/web/...` asserts all three routes without starting a real TCP listener.

- [ ] T023 [US5] Create `internal/web/handlers.go` with `handleRoot(version string) http.HandlerFunc` returning a closure writing `"MistHelper-Go v{version}"` with status 200, `handleHealth() http.HandlerFunc` writing `{"status":"ok"}` with `Content-Type: application/json` and status 200, and `handleNotFound() http.HandlerFunc` writing a plain text 404 message; log each request method and path at debug level; add `TestHandleRoot_Returns200WithVersion`, `TestHandleHealth_ReturnsJSON`, `TestHandleNotFound_Returns404` in `internal/web/handlers_test.go`

- [ ] T024 [US5] Create `internal/web/routes.go` with `registerRoutes(mux *http.ServeMux, version string)` wiring `/` to `handleRoot`, `/health` to `handleHealth`, and the catch-all `""` pattern to `handleNotFound`; pure routing logic only — no handler implementation in this file

- [ ] T025 [US5] Create `internal/web/server.go` with `Server` struct, `New(cfg api.Config, version string) *Server` creating an `http.ServeMux`, calling `registerRoutes`, and wrapping in `http.Server`, and `Start(ctx context.Context) error` calling `ListenAndServe` in a goroutine then calling `srv.Shutdown` with a 5-second `context.WithTimeout` on ctx cancel; log start and stop at info level; add `TestServer_StartsAndResponds`, `TestServer_GracefulShutdown` in `internal/web/server_test.go`

---

## Phase 8: Wiring

*Depends on*: T006 (Client), T013 (Writer), T015 (Dispatcher), T022 (SSH Server), T025 (Web Server).

- [ ] T026 Update `cmd/misthelper/main.go` to: add `--menu` (int), `--format` (string), and `--version` CLI flags; replace the existing `loadConfig()` stub with `api.LoadConfig(formatFlag)`; wire all five packages in order (`initPackages` → `registerStubs` → `startServers` → `runOrDispatch`); use `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` for the root context; implement graceful shutdown in fixed order per FR-031 — SSH `wg.Wait` (max 30s) → web `srv.Shutdown` (5s timeout) → API context cancel; `registerStubs` registers stub handlers for operations 1–89 via a `[]struct{n int; name string}` table; keep `main()` ≤25 lines by extracting `initPackages`, `startServers`, `runOrDispatch`, `registerStubs`, and `shutdown` as helper functions; update `cmd/misthelper/main_test.go` with smoke tests for `--version` flag and `--menu 0` (quit) invocation

---

## Phase 9: Quality Gates & Documentation

- [ ] T027 Run all quality gates in sequence and fix every reported issue before creating the PR: (1) `go build ./...` — zero compile errors; (2) `go vet ./...` — zero vet warnings; (3) `golangci-lint run ./...` — zero lint issues; (4) `go test ./... -race -cover` — all tests pass with race detector enabled; (5) `go test ./internal/... -cover -coverprofile=coverage.out && go tool cover -func=coverage.out` — each internal package must show ≥70% coverage; resolve any failure before proceeding

- [ ] T028 Update `CHANGELOG.md` with an entry under `[Unreleased]` describing the five new internal packages, two new dependencies (`modernc.org/sqlite`, `golang.org/x/crypto`), and updated `main.go` wiring; update `README.md` to document that stub handlers for operations 1–89 are registered and confirm SSH port 2200 and web port 8055 are active; verify operation count in README matches the 89 stubs registered

---

## Acceptance Criteria Reference

| Criterion | Satisfied by |
| - | - |
| SC-001: `go build ./internal/...` zero errors | T026, T027 |
| SC-002: coverage ≥70% per package | T009, T012, T018, all *_test.go tasks, T027 |
| SC-003: `golangci-lint` zero issues | T027 |
| SC-004: SSH + menu flow < 5s | T022, T026 |
| SC-005: 1,000 SQLite rows < 2s | T012 |
| SC-006: one test per exported function | all *_test.go tasks |
| SC-007: StrategyMap 100% endpoint coverage | T004 |
| FR-028: fixed rate-limit delay 200ms default | T007 |
| FR-029: password auth with configurable creds | T019 |
| FR-030: `--format` flag overrides `OUTPUT_FORMAT` | T002, T026 |
| FR-031: graceful shutdown order (SSH→web→API) | T026 |
