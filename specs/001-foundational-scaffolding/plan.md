# Implementation Plan: Foundational Go Scaffolding

**Feature Branch**: `feat/001-foundational-scaffolding`
**Spec**: [spec.md](spec.md)
**Created**: 2026-05-22
**Status**: Ready for Implementation

---

## Technical Context

### Current State
- `cmd/misthelper/main.go` — skeleton that loads `.env`, validates `MIST_API_TOKEN`/`MIST_ORG_ID`, and exits.
- `cmd/misthelper/main_test.go` — placeholder test file.
- `go.mod` — module `github.com/jmorrison-juniper/misthelper-go`, Go 1.24, only `godotenv v1.5.1` pinned.

### Target State
Five new internal packages and a fully wired `main.go` that compiles, passes all quality gates, and satisfies all user stories in the spec.

### New Dependencies Required
| Package | Version | Reason |
| - | - | - |
| `modernc.org/sqlite` | latest stable | CGO-free SQLite driver; preserves scratch-image container build |
| `golang.org/x/crypto` | latest stable | `ssh` sub-package for server-side SSH handshake |

Both are added to `go.mod` via `go get` before writing any implementation code.

### Architecture Summary
```
cmd/misthelper/main.go
  ├── internal/api/        Mist API client (wraps mistapi-go, retry, pagination)
  ├── internal/output/     CSV + SQLite writers (strategy map, upsert semantics)
  ├── internal/menu/       Menu dispatcher (register/dispatch, safeInput, EOF handling)
  ├── internal/ssh/        SSH server port 2200 (ForceCommand, session isolation)
  └── internal/web/        HTTP server port 8055 (health, version routes)
```

### Config Entity (single source of truth)
`Config` struct in `internal/api/config.go` is populated at startup from env + CLI flags and passed by value to all package constructors. No global variables.

| Field | Env Var | CLI Flag | Default |
| - | - | - | - |
| `APIToken` | `MIST_API_TOKEN` | — | (required) |
| `OrgID` | `MIST_ORG_ID` | — | (required) |
| `OutputFormat` | `OUTPUT_FORMAT` | `--format` | `csv` |
| `RateLimitMs` | `API_RATE_LIMIT_MS` | — | `200` |
| `SSHPort` | `SSH_PORT` | — | `2200` |
| `SSHUser` | `SSH_USER` | — | `misthelper` |
| `SSHPassword` | `SSH_PASSWORD` | — | `misthelper123!` |
| `WebPort` | `WEB_PORT` | — | `8055` |

### Graceful Shutdown Order (FR-031)
```
OS signal (SIGTERM/SIGINT)
  → cancel root context
  → SSH server: stop accepting; drain active sessions via WaitGroup (max 30s)
  → Web server: http.Server.Shutdown with 5s timeout
  → API client context cancel
Total max: 35 seconds
```

---

## Constitution Check

*No `.specify/memory/constitution.md` found in this repository — applying project coding
standards from `.github/copilot-instructions.md` directly.*

| Standard | Applied? | Notes |
| - | - | - |
| 5-Item Rule (≤5 files/pkg, ≤5 params, ≤25 lines/fn) | YES | Package file counts below all respect this |
| Inline comments on every executable line | YES | All generated code must follow |
| `slog.Info` before / `slog.Debug` after every action | YES | Enforced per package |
| `safeInput` for all stdin reads | YES | Implemented in `internal/menu/input.go` |
| Natural business keys in strategy map | YES | Full ENDPOINT_PRIMARY_KEY_STRATEGIES port required |
| `filepath.Join` not hardcoded separators | YES | All paths use `filepath` |
| ASCII-only in logs | YES | No Unicode or emoji in log output |
| `fmt.Errorf("context: %w", err)` for error wrapping | YES | All error paths |

---

## Phase 0: Research Resolutions

All NEEDS CLARIFICATION items were resolved in the spec's Clarifications section (2026-05-22). Decisions documented here for implementer reference.

### R-001: SQLite Driver Selection
- **Decision**: `modernc.org/sqlite` (pure Go, CGO-free)
- **Rationale**: Preserves the scratch-image container build pipeline. CGO-based drivers (`mattn/go-sqlite3`) require a C toolchain in the build stage and break the multi-stage Docker build.
- **Alternatives considered**: `mattn/go-sqlite3` (rejected: CGO required), `zombiezen.com/go/sqlite` (rejected: less widely known, same CGO consideration as modernc but with less documentation).

### R-002: SSH Library Selection
- **Decision**: `golang.org/x/crypto/ssh`
- **Rationale**: Standard extended library; no additional third-party dependency; well-documented server-side API; used in production Go SSH tools.
- **Alternatives considered**: `gliderlabs/ssh` (a wrapper around x/crypto/ssh — adds convenience but also a dependency and abstraction layer we don't need for a simple ForceCommand server).

### R-003: HTTP Framework Selection
- **Decision**: `net/http` standard library mux
- **Rationale**: The spec explicitly requires standard library only for this scaffold stage. The web UI is a skeleton and does not require routing middleware.

### R-004: Config Distribution Pattern
- **Decision**: Pass `Config` struct by value to each package constructor; no global variables.
- **Rationale**: Avoids race conditions during testing (multiple tests can construct independent instances). Aligns with Go idiomatic dependency injection.

### R-005: SSH Host Key Persistence
- **Decision**: Generate RSA-2048 host key on first boot; persist to `data/ssh_host_key`; reload on subsequent starts.
- **Rationale**: A new host key on every restart would break `known_hosts` for NOC engineers. Persisting to `data/` (which is a mounted volume) survives container restarts.

### R-006: Strategy Map Scope for SC-007
- **Decision**: Port all ~65 entries currently in `ENDPOINT_PRIMARY_KEY_STRATEGIES` in `MistHelper.py` (line 3488–4200). Include `timeseries_pk` type even though Redis backend is out of scope — the strategy map must be complete so future Redis backend wiring requires only the backend implementation, not map updates.
- **Rationale**: SC-007 requires 100% coverage of endpoint names at porting time; partial port would fail the success criterion.

---

## Phase 1: Design

### 1.1 Data Model

#### `Config` (internal/api/config.go)
```go
type Config struct {
    APIToken     string  // MIST_API_TOKEN (required)
    OrgID        string  // MIST_ORG_ID (required)
    OutputFormat string  // "csv" | "sqlite" (default: "csv")
    RateLimitMs  int     // API_RATE_LIMIT_MS (default: 200)
    SSHPort      int     // SSH_PORT (default: 2200)
    SSHUser      string  // SSH_USER (default: "misthelper")
    SSHPassword  string  // SSH_PASSWORD (default: "misthelper123!")
    WebPort      int     // WEB_PORT (default: 8055)
}
```

#### `Strategy` (internal/output/strategies.go)
```go
type PKType string  // "natural_pk" | "composite_pk" | "auto_increment_with_unique" | "timeseries_pk" | "default"

type Strategy struct {
    Type        PKType   // Primary key strategy type
    PrimaryKey  []string // Column names forming the primary key
    Indexes     []string // Additional columns to index
    Description string   // Human-readable description for logging
}
```

#### `HandlerFunc` (internal/menu/dispatcher.go)
```go
type HandlerFunc func(ctx context.Context) error
```

#### `Session` (internal/ssh/session.go)
```go
type Session struct {
    ID        string    // UUID for this connection
    ClientIP  string    // Remote address for logging
    Dir       string    // Absolute path to data/sessions/session_<ID>/
    StartTime time.Time // For duration logging on disconnect
}
```

### 1.2 Interface Contracts

#### `internal/api` — Client interface
```go
// Client abstracts the mistapi-go SDK for testability.
type Client interface {
    // ListOrgSites returns all sites for the configured org.
    ListOrgSites(ctx context.Context) ([]map[string]any, error)

    // ListSiteDevices returns all devices (type=all) at the given site.
    ListSiteDevices(ctx context.Context, siteID string) ([]map[string]any, error)

    // GetOrgInventory returns the full device inventory for the org.
    GetOrgInventory(ctx context.Context) ([]map[string]any, error)
}

// New constructs an authenticated Client from cfg.
// Returns an error if APIToken or OrgID is empty.
func New(cfg Config) (Client, error)
```

#### `internal/output` — Writer interface
```go
// Writer abstracts CSV and SQLite output backends.
type Writer interface {
    // Write persists data rows to the target file or table.
    // endpointName is looked up in StrategyMap for upsert key selection.
    Write(data []map[string]any, target, endpointName string) error
}

// NewWriter returns a CSVWriter or SQLiteWriter based on cfg.OutputFormat.
func NewWriter(cfg api.Config) (Writer, error)
```

#### `internal/menu` — Dispatcher interface
```go
// Dispatcher maps integer menu option numbers to handler functions.
type Dispatcher interface {
    // Register binds handler to menu option n.
    Register(n int, name string, handler HandlerFunc)

    // Dispatch calls the handler registered for n.
    // Returns ErrQuit when n==0, ErrUnknown when n is unregistered.
    Dispatch(ctx context.Context, n int) error

    // RunInteractive displays the menu and loops until quit or EOF.
    RunInteractive(ctx context.Context) error
}

// New constructs a Dispatcher with no registered handlers.
func New() Dispatcher
```

#### `internal/ssh` — Server
```go
// Server manages the SSH listener and per-session goroutines.
type Server struct { /* unexported fields */ }

// New constructs a Server using cfg for port and credentials.
func New(cfg api.Config, handler SessionHandler) (*Server, error)

// Start begins accepting connections. Blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) error

// SessionHandler is called in a goroutine for each accepted connection.
type SessionHandler func(ctx context.Context, sess Session)
```

#### `internal/web` — Server
```go
// Server wraps net/http with graceful shutdown support.
type Server struct { /* unexported fields */ }

// New constructs a Server on cfg.WebPort with version embedded.
func New(cfg api.Config, version string) *Server

// Start begins serving HTTP. Blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error
```

### 1.3 Package File Structure

Each package must have ≤5 source files (5-Item Rule).

#### `internal/api/` (4 files)
| File | Responsibility |
| - | - |
| `config.go` | `Config` struct, `LoadConfig(flags)`, env validation |
| `client.go` | `Client` interface, `mistClient` concrete struct, `New()` |
| `paginate.go` | `Paginate()` generic helper, rate limit delay |
| `retry.go` | `withRetry()`, exponential backoff + jitter logic |

#### `internal/output/` (5 files)
| File | Responsibility |
| - | - |
| `writer.go` | `Writer` interface, `NewWriter()` factory |
| `csv.go` | `CSVWriter` struct and `Write()` implementation |
| `sqlite.go` | `SQLiteWriter` struct, `Write()`, `ensureTable()`, upsert logic |
| `strategies.go` | `Strategy` struct, `StrategyMap` (all ~65 endpoint entries) |
| `flatten.go` | `flattenRow()` helper — converts `map[string]any` with nested maps to flat string map |

#### `internal/menu/` (4 files)
| File | Responsibility |
| - | - |
| `dispatcher.go` | `Dispatcher` interface, `menuDispatcher` struct, `Register()`, `Dispatch()` |
| `display.go` | `printMenu()` — formats and prints the numbered menu table |
| `input.go` | `safeInput()` — EOF-safe stdin read; `parseSelection()` |
| `loop.go` | `RunInteractive()` — the menu loop combining display + input + dispatch |

#### `internal/ssh/` (4 files)
| File | Responsibility |
| - | - |
| `server.go` | `Server` struct, `New()`, `Start()`, listener lifecycle |
| `session.go` | `Session` struct, `newSession()`, directory creation/cleanup |
| `auth.go` | `passwordCallback` — password auth handler; host key generation/loading |
| `handler.go` | `forceCommandHandler` — rejects non-ForceCommand requests; invokes `SessionHandler` |

#### `internal/web/` (3 files)
| File | Responsibility |
| - | - |
| `server.go` | `Server` struct, `New()`, `Start()`, graceful shutdown |
| `handlers.go` | `handleRoot()`, `handleHealth()`, `handleNotFound()` |
| `routes.go` | `registerRoutes()` — wires handlers to mux patterns |

### 1.4 Updated `cmd/misthelper/main.go` Structure

The existing skeleton is replaced with a fully wired startup sequence:

```
main()
  ├── parseCLIFlags()           → menuNum int, format string, showVersion bool
  ├── handleVersionFlag()       → exit if --version
  ├── loadEnvAndBuildConfig()   → Config (env + CLI override)
  ├── initPackages()            → Client, Writer, Dispatcher, SSHServer, WebServer
  ├── registerMenuHandlers()    → stubs for operations 1–89
  ├── startServers()            → SSH + Web in goroutines with WaitGroup
  └── runOrDispatch()           → interactive loop OR --menu N direct dispatch
```

Each of these helpers is ≤25 lines. Graceful shutdown is wired with `signal.NotifyContext`.

---

## Phase 2: Implementation Tasks

Tasks are ordered by dependency. Complete each task and run quality gates before
proceeding to the next.

### Task 0: Add Dependencies

**Files changed**: `go.mod`, `go.sum`

```powershell
cd "C:\Users\jmorrison\OneDrive - Hewlett Packard Enterprise\Code\MistHelper-Go"
go get modernc.org/sqlite
go get golang.org/x/crypto/ssh
go mod tidy
```

**Verify**: `go build ./...` still compiles with zero errors.

---

### Task 1: `internal/api/config.go`

**New file**. Defines `Config` struct and `LoadConfig` function.

Key implementation notes:
- `LoadConfig` reads env vars via `os.Getenv`, applies defaults, validates required fields.
- It accepts a `format` string parameter (from CLI flag) and applies CLI precedence over env.
- Returns `(Config, error)` — never panics on missing values.
- `RateLimitMs` parses `API_RATE_LIMIT_MS` as int with `strconv.Atoi`; falls back to 200 on parse error.
- Every executable line gets an inline comment.

**Test coverage target**: `TestLoadConfig_ValidEnv`, `TestLoadConfig_MissingToken`, `TestLoadConfig_MissingOrgID`, `TestLoadConfig_CLIFormatOverridesEnv`, `TestLoadConfig_Defaults`.

---

### Task 2: `internal/api/client.go`

**New file**. Defines the `Client` interface and `mistClient` concrete struct.

Key implementation notes:
- `mistClient` embeds the `mistapi-go` SDK client (unexported `sdk` field).
- `New(cfg Config) (Client, error)` calls `mistapi.NewClient()` with the API token.
- `ListOrgSites`, `ListSiteDevices`, `GetOrgInventory` call the corresponding SDK methods, convert results to `[]map[string]any` via a `toRowSlice` helper, and apply rate limiting.
- `slog.Info("calling API", "method", "ListOrgSites")` before each SDK call; `slog.Debug("API call complete", "rows", len(result))` after.
- Token is never logged.

**Test coverage target**: `TestNew_MissingToken`, `TestNew_ValidConfig`, tests using a mock `Client` interface (table-driven).

---

### Task 3: `internal/api/paginate.go`

**New file**. Implements pagination and the fixed-delay rate limiter.

Key implementation notes:
- `Paginate` is a generic helper `func Paginate[T any](ctx context.Context, fn PageFunc[T], cfg Config) ([]T, error)`.
- `PageFunc[T]` signature: `func(page, limit int) ([]T, bool, error)` — returns (rows, hasMore, error).
- Loop: call `fn`, append results, sleep `cfg.RateLimitMs` milliseconds, repeat until `!hasMore`.
- Log page number at debug level on each iteration.
- Max 25 lines; extract the sleep-and-log into a `waitRateLimit(ms int)` helper if needed.

**Test coverage target**: `TestPaginate_SinglePage`, `TestPaginate_MultiPage`, `TestPaginate_ErrorOnPage2`.

---

### Task 4: `internal/api/retry.go`

**New file**. Implements exponential backoff with jitter for transient errors.

Key implementation notes:
- `withRetry(ctx context.Context, op func() error, cfg RetryConfig) error`
- `RetryConfig`: max attempts (default 3), base delay (default 500ms), max delay (default 10s).
- Retry on HTTP 429 and 5xx; surface all other errors immediately.
- Use `math/rand` for jitter: `delay = base * 2^attempt + rand.Intn(base)`.
- Log each retry attempt: `slog.Info("retrying after transient error", "attempt", n, "delay_ms", delay)`.
- Never log the response body (may contain PII).

**Test coverage target**: `TestWithRetry_SuccessFirstAttempt`, `TestWithRetry_SuccessAfterRetry`, `TestWithRetry_ExhaustsRetries`, `TestWithRetry_NonRetryableError`.

---

### Task 5: `internal/output/strategies.go`

**New file**. Defines `Strategy`, `PKType`, and `StrategyMap`.

Key implementation notes:
- Port **all ~65 entries** from `ENDPOINT_PRIMARY_KEY_STRATEGIES` in `MistHelper.py` (line 3488).
- Include `timeseries_pk` entries even though the Redis backend is deferred — strategy map completeness is required by SC-007.
- Include a `"default"` entry: `{Type: "auto_increment_with_unique", PrimaryKey: []string{"misthelper_internal_id"}}`.
- `StrategyFor(endpointName string) Strategy` returns the named strategy or `"default"` if not found; logs the fallback at warn level.
- The map is a `var StrategyMap = map[string]Strategy{...}` package-level variable (read-only after init; safe for concurrent reads without a mutex).

**Test coverage target**: `TestStrategyFor_KnownEndpoint`, `TestStrategyFor_UnknownEndpoint_FallsBackToDefault`, `TestStrategyMap_AllPKTypesRepresented`.

---

### Task 6: `internal/output/flatten.go`

**New file**. Converts nested `map[string]any` API responses to flat `map[string]string` rows.

Key implementation notes:
- `flattenRow(row map[string]any, prefix string) map[string]string`
- Recursively flattens nested maps with dot notation: `{"a": {"b": 1}}` → `{"a.b": "1"}`.
- Non-map values are converted to string with `fmt.Sprintf("%v", v)`.
- Max recursion depth is 10 (prevents stack overflow on pathological inputs); log a warning if exceeded.

**Test coverage target**: `TestFlattenRow_Flat`, `TestFlattenRow_Nested`, `TestFlattenRow_NilValue`, `TestFlattenRow_MaxDepth`.

---

### Task 7: `internal/output/csv.go`

**New file**. Implements `CSVWriter`.

Key implementation notes:
- `CSVWriter.Write(data []map[string]any, target, endpointName string) error`
- Creates `data/` directory if absent (`os.MkdirAll`).
- Flattens each row via `flattenRow`.
- Collects all unique keys across all rows for the header (sorted alphabetically for deterministic output).
- Uses `encoding/csv` writer; flushes and closes file on return.
- Logs file path and row count: `slog.Info("CSV write complete", "file", path, "rows", n)`.

**Test coverage target**: `TestCSVWriter_CreatesFile`, `TestCSVWriter_CorrectHeaders`, `TestCSVWriter_CreatesDataDir`, `TestCSVWriter_EmptyData`.

---

### Task 8: `internal/output/sqlite.go`

**New file**. Implements `SQLiteWriter`.

Key implementation notes:
- Opens `data/mist_data.db` via `database/sql` + `modernc.org/sqlite`.
- `ensureTable(db, tableName, cols []string, strategy Strategy) error` — creates table if not exists; adds `misthelper_internal_id INTEGER PRIMARY KEY AUTOINCREMENT` for auto-increment strategies.
- Upsert: `INSERT OR REPLACE INTO ...` for `natural_pk` and `composite_pk`; plain `INSERT` for `auto_increment_with_unique`.
- SQLite busy-timeout set to 5000ms on `db.Open` to handle concurrent write retries (FR edge case).
- Logs table name and row count on success.

**Test coverage target**: `TestSQLiteWriter_CreatesTable`, `TestSQLiteWriter_NaturalPKUpsert`, `TestSQLiteWriter_CompositeUpsert`, `TestSQLiteWriter_AutoIncrementNoDuplicate`, `TestSQLiteWriter_1000RowsBenchmark` (must complete < 2s per SC-005).

---

### Task 9: `internal/output/writer.go`

**New file**. Defines the `Writer` interface and `NewWriter` factory.

Key implementation notes:
- `NewWriter(cfg api.Config) (Writer, error)` switches on `cfg.OutputFormat`.
- Unknown format: log warning and fall back to CSV (FR edge case in spec).
- Returns the concrete writer wrapped in the interface; callers never see the concrete type.

**Test coverage target**: `TestNewWriter_CSV`, `TestNewWriter_SQLite`, `TestNewWriter_UnknownFormatFallsBackToCSV`.

---

### Task 10: `internal/menu/input.go`

**New file**. Implements `safeInput` and `parseSelection`.

Key implementation notes:
- `safeInput(reader *bufio.Reader, prompt, context string) (string, error)` — prints prompt, reads until newline, returns trimmed string; on `io.EOF` returns `("", ErrEOF)` (not a panic).
- `parseSelection(s string) (int, error)` — trims whitespace, parses to int with `strconv.Atoi`.
- Define sentinel `ErrEOF = errors.New("stdin closed")` and `ErrQuit = errors.New("user quit")`.
- Both helpers are ≤25 lines.

**Test coverage target**: `TestSafeInput_Normal`, `TestSafeInput_EOF`, `TestSafeInput_Whitespace`, `TestParseSelection_Valid`, `TestParseSelection_Invalid`.

---

### Task 11: `internal/menu/dispatcher.go`

**New file**. Implements `Dispatcher` interface and `menuDispatcher` struct.

Key implementation notes:
- `menuDispatcher` holds `handlers map[int]entry` where `entry` is `{name string, fn HandlerFunc}`.
- `Register(n int, name string, handler HandlerFunc)` adds to the map; panics if n is already registered (programming error, not runtime error).
- `Dispatch(ctx, n)` returns `ErrQuit` for n==0, `ErrUnknown` for unregistered n (after printing "unknown option"), else calls the handler.
- `ErrUnknown = errors.New("unknown menu option")`.

**Test coverage target**: `TestDispatch_ValidOption`, `TestDispatch_Quit`, `TestDispatch_Unknown`, `TestDispatch_HandlerError`.

---

### Task 12: `internal/menu/display.go`

**New file**. Implements `printMenu`.

Key implementation notes:
- `printMenu(w io.Writer, handlers map[int]entry)` — prints a sorted table of registered options.
- Uses `fmt.Fprintf(w, ...)` rather than `fmt.Println` so tests can capture output.
- Sorted by option number (ascending); use `slices.Sorted` (Go 1.23+) or manual sort.
- Always prints `"  0. Exit"` as the last entry.

**Test coverage target**: `TestPrintMenu_OutputContainsAllHandlers`, `TestPrintMenu_ZeroAlwaysLast`.

---

### Task 13: `internal/menu/loop.go`

**New file**. Implements `RunInteractive`.

Key implementation notes:
- `RunInteractive(ctx context.Context) error` — loop: `printMenu` → `safeInput` → `parseSelection` → `Dispatch`.
- On `ErrEOF`: log `slog.Info("menu: stdin closed, exiting")` and return nil.
- On `ErrQuit`: log `slog.Info("menu: user quit")` and return nil.
- On `ErrUnknown`: print message, continue loop (do not exit).
- Respects context cancellation: check `ctx.Done()` at top of each iteration.

**Test coverage target**: `TestRunInteractive_QuitOnZero`, `TestRunInteractive_EOFExitsCleanly`, `TestRunInteractive_UnknownOptionContinues`.

---

### Task 14: `internal/ssh/auth.go`

**New file**. Implements SSH host key generation and password authentication.

Key implementation notes:
- `loadOrGenerateHostKey(path string) (ssh.Signer, error)` — reads PEM from `path`; if not found, generates RSA-2048, writes PEM to `path`, returns signer.
- `passwordCallback(user, password string, cfg api.Config) bool` — compares against `cfg.SSHUser`/`cfg.SSHPassword`; never logs the password value.
- Key file path: `filepath.Join("data", "ssh_host_key")`.
- Log host key fingerprint (not the key itself) at info level on load or generation.

**Test coverage target**: `TestLoadOrGenerateHostKey_GeneratesIfMissing`, `TestLoadOrGenerateHostKey_LoadsExisting`, `TestPasswordCallback_Valid`, `TestPasswordCallback_Invalid`.

---

### Task 15: `internal/ssh/session.go`

**New file**. Implements `Session` lifecycle.

Key implementation notes:
- `newSession(clientIP string) (Session, error)` — generates UUID (use `crypto/rand` to build a simple 8-byte hex ID, no third-party UUID library), creates `data/sessions/session_<ID>/` with `os.MkdirAll`.
- `(s Session) Cleanup() error` — removes the session directory with `os.RemoveAll`; logs duration.
- Log session creation and cleanup at info level including `session_id` and `client_ip`.

**Test coverage target**: `TestNewSession_CreatesDirectory`, `TestSession_Cleanup_RemovesDirectory`.

---

### Task 16: `internal/ssh/handler.go`

**New file**. Implements the ForceCommand handler.

Key implementation notes:
- `forceCommandHandler(conn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request, sess Session, handler SessionHandler)` — rejects any channel type other than `session`; within a session channel, rejects any request type that would grant shell access (`shell`, `exec`, `pty-req` except for the ForceCommand flow); invokes `handler(ctx, sess)`.
- Log rejected shell attempts at warn level: `slog.Warn("SSH: rejected shell request", "session_id", sess.ID, "client_ip", sess.ClientIP)`.
- Must drain `reqs` and `chans` channels even when the connection is being rejected (prevent goroutine leaks).

**Test coverage target**: `TestForceCommandHandler_RejectsShell`, `TestForceCommandHandler_InvokesHandler`.

---

### Task 17: `internal/ssh/server.go`

**New file**. Implements the SSH server.

Key implementation notes:
- `New(cfg api.Config, handler SessionHandler) (*Server, error)` — calls `loadOrGenerateHostKey`, builds `ssh.ServerConfig` with password auth callback, stores listener address.
- `Start(ctx context.Context, wg *sync.WaitGroup) error` — calls `net.Listen("tcp", addr)`, logs port; then loops `Accept()` in a goroutine per connection; on ctx cancel, closes the listener (which unblocks Accept).
- Each accepted connection: `wg.Add(1)`, `go func() { defer wg.Done(); handleConn(...) }()`.
- Port-already-in-use error is detected at `net.Listen` and returned immediately (FR edge case).

**Test coverage target**: `TestServer_StartsOnPort`, `TestServer_RejectsInvalidPassword`, `TestServer_SessionIsolation` (two mock connections get distinct session dirs).

---

### Task 18: `internal/web/handlers.go`

**New file**. Implements HTTP handler functions.

Key implementation notes:
- `handleRoot(version string) http.HandlerFunc` — returns a closure; writes `"MistHelper-Go v{version}"` with status 200.
- `handleHealth() http.HandlerFunc` — writes `{"status":"ok"}` with `Content-Type: application/json` and status 200.
- `handleNotFound() http.HandlerFunc` — writes plain text 404 message.
- Log each request at debug level: `slog.Debug("HTTP request", "method", r.Method, "path", r.URL.Path)`.

**Test coverage target**: `TestHandleRoot_Returns200WithVersion`, `TestHandleHealth_ReturnsJSON`, `TestHandleNotFound_Returns404`.

---

### Task 19: `internal/web/routes.go`

**New file**. Wires handlers to mux.

Key implementation notes:
- `registerRoutes(mux *http.ServeMux, version string)` — registers `/`, `/health`, and the catch-all `""` pattern for 404.
- Pure routing logic; no handler implementation here.

**Test coverage target**: covered by handler tests + integration test in `server_test.go`.

---

### Task 20: `internal/web/server.go`

**New file**. Implements the HTTP server with graceful shutdown.

Key implementation notes:
- `New(cfg api.Config, version string) *Server` — creates `http.ServeMux`, calls `registerRoutes`, wraps in `http.Server`.
- `Start(ctx context.Context) error` — calls `ListenAndServe` in a goroutine; on ctx cancel, calls `srv.Shutdown(shutdownCtx)` with a 5-second `context.WithTimeout`.
- Log start and stop at info level.

**Test coverage target**: `TestServer_StartsAndResponds`, `TestServer_GracefulShutdown`.

---

### Task 21: Update `cmd/misthelper/main.go`

**Modify existing file**. Wire all five packages into the startup sequence.

Key implementation notes:
- Add `--menu` (int) and `--format` (string) flags alongside the existing `--version` flag.
- Replace `loadConfig()` with `api.LoadConfig(formatFlag)` returning `(api.Config, error)`.
- Construct `Client`, `Writer`, `Dispatcher` in sequence; return on any error.
- Call `registerStubs(d)` — registers stubs for operations 1–89 (a separate helper function that calls `d.Register(n, name, stubHandler)` in a loop over a `[]struct{int, string}` table).
- Start SSH and web servers in goroutines with `sync.WaitGroup`; use `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` for the root context.
- Implement the fixed shutdown order (FR-031): SSH drain → web shutdown → API context cancel.
- After all goroutines join, exit with code 0.

**5-Item Rule compliance**: `main()` must be ≤25 lines; extract `initPackages`, `startServers`, `runOrDispatch`, `registerStubs`, `shutdown` as helpers.

---

### Task 22: Quality Gates

Run **in this exact order** and fix all issues before proceeding to PR.

```powershell
cd "C:\Users\jmorrison\OneDrive - Hewlett Packard Enterprise\Code\MistHelper-Go"

# 1. Compile check
go build ./...

# 2. Vet
go vet ./...

# 3. Lint
golangci-lint run ./...

# 4. Tests with race detector and coverage
go test ./... -race -cover

# 5. Coverage check — each internal package must reach ≥70%
go test ./internal/... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v "100.0%"
```

All gates must pass before creating the PR.

---

### Task 23: Update go.mod, CHANGELOG.md, README.md

**Files changed**: `go.mod`, `go.sum`, `CHANGELOG.md`, `README.md`

- `go.mod`/`go.sum`: already updated in Task 0.
- `CHANGELOG.md`: add entry under `[Unreleased]` noting the five new packages, new dependencies, and updated wiring in `main.go`.
- `README.md`: update any operation count if stubs register all 89 menu operations; confirm SSH and web ports are documented.

---

## Ordered Delivery Sequence

```
Task 0  → go.mod deps (go get + tidy)
Task 1  → internal/api/config.go
Task 2  → internal/api/client.go
Task 3  → internal/api/paginate.go
Task 4  → internal/api/retry.go
Task 5  → internal/output/strategies.go      ← SC-007 critical path
Task 6  → internal/output/flatten.go
Task 7  → internal/output/csv.go
Task 8  → internal/output/sqlite.go
Task 9  → internal/output/writer.go
Task 10 → internal/menu/input.go
Task 11 → internal/menu/dispatcher.go
Task 12 → internal/menu/display.go
Task 13 → internal/menu/loop.go
Task 14 → internal/ssh/auth.go
Task 15 → internal/ssh/session.go
Task 16 → internal/ssh/handler.go
Task 17 → internal/ssh/server.go
Task 18 → internal/web/handlers.go
Task 19 → internal/web/routes.go
Task 20 → internal/web/server.go
Task 21 → cmd/misthelper/main.go (update)
Task 22 → Quality gates (all must pass)
Task 23 → CHANGELOG.md + README.md
```

---

## Success Criteria Mapping

| SC | Task(s) |
| - | - |
| SC-001: `go build ./internal/...` zero errors | Tasks 1–20, 22 |
| SC-002: coverage ≥70% per package | Tasks 1–20, 22 |
| SC-003: `golangci-lint` zero issues | Task 22 |
| SC-004: SSH + menu flow completes < 5s | Tasks 14–17, 21 |
| SC-005: 1,000 SQLite rows < 2s | Task 8 benchmark test |
| SC-006: one test per exported function | Tasks 1–20 (test targets listed per task) |
| SC-007: StrategyMap 100% complete | Task 5 |

---

## Risk Register

| Risk | Likelihood | Mitigation |
| - | - | - |
| `modernc.org/sqlite` CGO-free driver has different SQL dialect quirks than mattn | Low | `INSERT OR REPLACE` is ANSI SQL and works in modernc.org/sqlite; verified in documentation |
| SSH host key file permissions — `data/` volume mount may have wrong perms (known container issue) | Medium | Use `os.Chmod(keyPath, 0600)` after writing; document in deploy notes |
| `x/crypto/ssh` server API changes across minor versions | Low | Pin to latest stable via `go get`; the server API has been stable since Go 1.11 |
| 5-Item Rule violation in StrategyMap (~65 entries in one file) | Low | A single large `var` declaration with many entries is one top-level construct, not 65; the rule applies to file count and function body line count, not map literal size |
| `--menu N` with unregistered N when stubs are not yet registered | Low | `Dispatch` returns `ErrUnknown` and exits non-zero per spec edge case |
