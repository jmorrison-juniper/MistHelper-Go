# Feature Specification: Foundational Go Scaffolding

**Feature Branch**: `feat/001-foundational-scaffolding`
**Created**: 2026-05-22
**Status**: Ready for Planning
**Spec Directory**: `specs/001-foundational-scaffolding`

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — API Client Initialization (Priority: P1)

A NOC engineer starts MistHelper-Go in interactive mode. The tool connects to the Mist
Cloud using credentials from `.env`, validates the token and org ID, and confirms the
session is ready before the main menu appears.

**Why this priority**: All 187 menu operations depend on an authenticated Mist API
session. Nothing else works without this foundation.

**Independent Test**: Set valid `MIST_API_TOKEN` and `MIST_ORG_ID` in `.env`, run
`go test ./internal/api/...` — tests pass if the client can be constructed, configured,
and reports authentication status without making live API calls (mocked transport).

**Acceptance Scenarios**:

1. **Given** valid `MIST_API_TOKEN` and `MIST_ORG_ID` in `.env`, **When** the API client
   is initialized, **Then** it holds an authenticated SDK client and the org ID with no errors.
2. **Given** missing `MIST_API_TOKEN`, **When** the API client is initialized, **Then** it
   returns a descriptive error and the process exits non-zero.
3. **Given** the API client is initialized, **When** a paginated list call is made,
   **Then** the client fetches all pages and returns the merged result.
4. **Given** a transient 429 or 5xx response, **When** the client retries, **Then** it
   backs off with jitter and succeeds on the next attempt, logging each retry at info level.

---

### User Story 2 — Output Writing (Priority: P1)

A NOC engineer runs a menu operation that returns a list of sites. The result is written
to a CSV file (default) or a SQLite database row-by-row with upsert semantics, based on
the `OUTPUT_FORMAT` environment variable.

**Why this priority**: Every data-extraction menu operation (1–89) depends on the output
writer. Without it, no data can be persisted.

**Independent Test**: Call `output.Write(data, "test_table", "listOrgSites")` with mock
data — CSV file appears in `data/`, SQLite file is created with correct schema and rows;
re-running with same data upserts rather than duplicates.

**Acceptance Scenarios**:

1. **Given** `OUTPUT_FORMAT=csv` and a list of dicts, **When** `Write()` is called,
   **Then** a CSV is created in `data/` with headers matching all dict keys.
2. **Given** `OUTPUT_FORMAT=sqlite` and endpoint name `"listOrgSites"`, **When**
   `Write()` is called, **Then** a SQLite table is created with `id` as PRIMARY KEY
   and the correct index columns per the strategy map.
3. **Given** the same data is written twice with `OUTPUT_FORMAT=sqlite`, **When** the
   second write completes, **Then** the row count stays the same (upsert, not duplicate).
4. **Given** an unknown endpoint name, **When** `Write()` is called, **Then** it falls
   back to the `default` auto-increment strategy and writes successfully.

---

### User Story 3 — Interactive Menu Dispatch (Priority: P2)

A NOC engineer runs `./misthelper` with no flags. A numbered menu appears. They type
`1` and press Enter. The tool dispatches to operation 1 (not yet implemented) and prints
a placeholder message. They type `0` to quit cleanly.

**Why this priority**: The menu is the primary user interface. Even empty stubs let the
team verify navigation, input handling, and EOF safety before operations are ported.

**Independent Test**: Run `go test ./internal/menu/...` with a simulated stdin — assert
that `Dispatch(1)` calls the registered handler, `Dispatch(0)` returns the quit signal,
and providing EOF input exits cleanly rather than panicking.

**Acceptance Scenarios**:

1. **Given** the tool starts in interactive mode, **When** the user types a valid menu
   number, **Then** the registered handler is called with no panics.
2. **Given** `--menu 11` is passed on the command line, **When** the tool starts,
   **Then** operation 11's handler is called directly without showing the menu.
3. **Given** an SSH session that closes unexpectedly, **When** stdin sends EOF,
   **Then** the menu loop exits cleanly with a log message rather than crashing.
4. **Given** the user types an unregistered number, **When** dispatch resolves it,
   **Then** a user-friendly "unknown option" message is printed and the menu re-displays.

---

### User Story 4 — SSH Server Access (Priority: P2)

A NOC engineer SSH-es to the container on port 2200 (`ssh misthelper@<host> -p 2200`).
The connection is accepted, a new isolated session directory is created, and MistHelper-Go
launches inside that session. When the session ends, the session directory is cleaned up.

**Why this priority**: Container-based SSH access is the primary deployment model for
NOC teams without direct host access.

**Independent Test**: Run `go test ./internal/ssh/...` — assert that the server starts on
port 2200, accepts a mock connection, creates a session directory under `data/sessions/`,
invokes the ForceCommand handler, and cleans up the directory on disconnect.

**Acceptance Scenarios**:

1. **Given** the container is running, **When** a user SSH-es to port 2200, **Then** the
   connection is accepted and MistHelper-Go launches in a new session directory.
2. **Given** the ForceCommand is set, **When** the user attempts to drop to a shell,
   **Then** the connection is rejected and the attempt is logged at warn level.
3. **Given** an active SSH session, **When** the session ends, **Then** the temporary
   session directory is removed and a final log entry is written.
4. **Given** the SSH server is already running, **When** a second connection arrives,
   **Then** both sessions are isolated from each other with separate working directories.

---

### User Story 5 — Web UI Skeleton (Priority: P3)

A NOC engineer opens `http://localhost:8055` in a browser. A page loads confirming
MistHelper-Go is running and showing the version number. Basic routes exist that will
be expanded by future features.

**Why this priority**: The web UI is used by some NOC workflows but is not required for
CLI menu operations. A route skeleton can be tested independently of all other packages.

**Independent Test**: Run `go test ./internal/web/...` — assert that an HTTP GET to
`/` returns 200 with the version string, and that `/health` returns `{"status":"ok"}`.

**Acceptance Scenarios**:

1. **Given** the tool starts, **When** an HTTP GET is made to `/`, **Then** a 200
   response is returned with the application name and version.
2. **Given** the tool starts, **When** an HTTP GET is made to `/health`, **Then** a
   200 JSON response `{"status":"ok"}` is returned.
3. **Given** an unregistered path, **When** a GET request is made, **Then** a 404
   response is returned with a plain text message.

---

### Edge Cases

- What happens when `data/` does not exist at startup? The output package must create
  it before the first write rather than returning an error to the caller.
- What happens if SQLite is locked by a concurrent write? The output package must
  retry with timeout rather than returning an error to the user immediately.
- What happens when `--menu N` refers to an unregistered operation? The tool must print
  a clear error and exit non-zero rather than panicking.
- What happens when the SSH server port 2200 is already in use? The server must log the
  conflict and return an error so the process can exit rather than silently failing.
- What happens when `OUTPUT_FORMAT` is set to an unsupported value? The output package
  must log a warning and fall back to CSV rather than crashing.

---

## Requirements *(mandatory)*

### Functional Requirements

#### internal/api

- **FR-001**: The API package MUST expose a `Client` interface with at minimum:
  `ListOrgSites`, `ListSiteDevices`, and `GetOrgInventory` methods, establishing the
  pattern all future operations follow.
- **FR-002**: The `Client` interface MUST include a generic `Paginate` helper that
  iterates all pages of a paginated SDK call and returns the merged result slice.
- **FR-003**: The API package MUST implement retry logic for transient errors (429,
  5xx) with exponential backoff and jitter, configurable via environment variables.
- **FR-004**: The API package MUST provide a `New(cfg Config) (Client, error)` constructor
  that reads credentials from environment and returns an error for missing required values.
- **FR-005**: The API package MUST log all API calls at debug level (URL, method) and
  log retry attempts at info level, never logging token values.
- **FR-028**: The API package MUST insert a fixed delay between consecutive API calls,
  configurable via `API_RATE_LIMIT_MS` environment variable (default 200ms). The adaptive
  delay system (`delay_metrics.json`, `tuning_data.json`) is a future port and is out of
  scope for this scaffold.

#### internal/output

- **FR-006**: The output package MUST expose a `Writer` interface with a `Write(data
  []map[string]any, target, endpointName string) error` method.
- **FR-007**: The output package MUST include a `StrategyMap` containing the full set of
  primary key strategies ported from `ENDPOINT_PRIMARY_KEY_STRATEGIES` in `MistHelper.py`,
  covering at minimum: `natural_pk`, `composite_pk`, `auto_increment_with_unique`, and
  `default`.
- **FR-008**: The `CSVWriter` implementation MUST write output files to the `data/`
  directory, creating the directory if absent, with the exact filename passed by the caller.
- **FR-009**: The `SQLiteWriter` implementation MUST use `INSERT OR REPLACE` for
  `natural_pk` and `composite_pk` strategies to provide upsert semantics.
- **FR-010**: The output package MUST log the row count written and the target filename
  or table name at info level after every successful write.

#### internal/menu

- **FR-011**: The menu package MUST expose a `Dispatcher` interface with `Register(n int,
  handler HandlerFunc)` and `Dispatch(n int) error` methods.
- **FR-012**: The menu package MUST implement a `safeInput` function that wraps `bufio.Reader`
  reads and returns a clean error (not a panic) on EOF or closed stdin.
- **FR-013**: The menu package MUST support direct invocation via `--menu N` CLI flag,
  calling the registered handler for N without displaying the interactive menu.
- **FR-014**: The menu package MUST display a numbered menu table and loop until the user
  enters `0` or the process receives a shutdown signal.
- **FR-015**: The menu package MUST log the selected operation number and handler name at
  info level before dispatching.

#### internal/ssh

- **FR-016**: The SSH server MUST listen on port 2200 by default, configurable via
  `SSH_PORT` environment variable.
- **FR-017**: The SSH server MUST enforce a ForceCommand pattern: every connection runs
  MistHelper-Go's menu; no interactive shell is provided.
- **FR-018**: The SSH server MUST create a unique session directory under
  `data/sessions/session_<id>/` for each connection and remove it when the session ends.
- **FR-019**: The SSH server MUST log connection acceptance and termination (source IP,
  session ID, duration) at info level.
- **FR-020**: The SSH server MUST reject connections that attempt to bypass ForceCommand
  and log the attempt at warn level.
- **FR-029**: The SSH server MUST authenticate clients using password authentication only
  for this scaffold; the default credentials are username `misthelper` and password
  `misthelper123!`, configurable via `SSH_USER` and `SSH_PASSWORD` environment variables.
  Key-based authentication is a future enhancement and is out of scope for this scaffold.

#### internal/web

- **FR-021**: The web server MUST listen on port 8055 by default, configurable via
  `WEB_PORT` environment variable.
- **FR-022**: The web server MUST serve `GET /` returning 200 with application name
  and version.
- **FR-023**: The web server MUST serve `GET /health` returning `{"status":"ok"}` with
  content-type `application/json`.
- **FR-024**: The web server MUST return 404 for unregistered routes with a plain text
  message.

#### cmd/misthelper/main.go

- **FR-025**: `main.go` MUST wire all five internal packages into a single startup
  sequence: config load → API client init → output writer init → SSH server start →
  web server start → menu run.
- **FR-026**: `main.go` MUST handle `--menu N` by passing N to the menu dispatcher
  rather than entering interactive mode.
- **FR-027**: `main.go` MUST use `context.WithCancel` so that OS signals (`SIGTERM`,
  `SIGINT`) propagate shutdown to all running servers and the menu loop cleanly.
- **FR-030**: `main.go` MUST support a `--format csv|sqlite` CLI flag that overrides
  `OUTPUT_FORMAT`. The CLI flag takes precedence over the env var; if neither is set the
  default is `csv`. The resolved format is passed to the output writer at initialization.
- **FR-031**: `main.go` MUST implement graceful shutdown in this fixed order: (1) SSH
  server stops accepting new connections; active sessions are tracked via `sync.WaitGroup`
  and drained for up to 30 seconds; (2) web server shuts down with a 5-second
  `context.WithTimeout` deadline; (3) API client context is cancelled after both servers
  have stopped. Total maximum shutdown time is 35 seconds.

### Key Entities

- **Config**: Holds `APIToken`, `OrgID`, `OutputFormat`, `RateLimitMs`, `SSHPort`,
  `SSHUser`, `SSHPassword`, `WebPort` read from environment variables; `OutputFormat` is
  overridable by `--format` CLI flag (CLI flag takes precedence). Single org per process;
  multi-org switching is out of scope. Validated at startup.
- **Client** (interface): Abstracts the `mistapi-go` SDK client; allows mocking in tests.
- **Writer** (interface): Abstracts CSV and SQLite backends; selected by `OUTPUT_FORMAT`.
- **Strategy**: Describes primary key type, key columns, and index columns for one API
  endpoint; looked up by endpoint name at write time.
- **Dispatcher** (interface): Maps integer menu option numbers to `HandlerFunc` values;
  shared between interactive and `--menu N` invocation paths.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All five internal packages compile independently (`go build ./internal/...`)
  with zero errors and zero warnings from `go vet ./...`.
- **SC-002**: All unit tests pass (`go test ./... -race -cover`) with coverage ≥ 70%
  across each internal package.
- **SC-003**: `golangci-lint run ./...` reports zero issues after scaffolding is complete.
- **SC-004**: An operator can start the binary, SSH to port 2200, navigate the (stub)
  menu, and exit cleanly — the entire flow completes in under 5 seconds.
- **SC-005**: Writing 1,000 rows via the SQLite writer completes in under 2 seconds on
  a development machine (ensures the upsert path is not pathologically slow).
- **SC-006**: Every `internal/` package has its own `_test.go` file with at minimum one
  test per exported function or interface method.
- **SC-007**: The `StrategyMap` in `internal/output/` covers 100% of the endpoint names
  present in `ENDPOINT_PRIMARY_KEY_STRATEGIES` in `MistHelper.py` at the time of porting.

---

## Assumptions

- Go 1.24, `mistapi-go` v0.4.73+, and `godotenv` v1.5.1 are already pinned in `go.mod`
  and require no version changes for this scaffold.
- The 5-Item Rule (max 5 files per package, max 5 params per function, max 25 lines
  per function body) is enforced throughout; any violation is treated as a build blocker.
- The SSH server uses the standard library `golang.org/x/crypto/ssh` package for the
  server-side handshake; no additional SSH dependencies are introduced.
- The web server uses the standard library `net/http` mux; no third-party HTTP framework
  is introduced at this scaffold stage.
- SQLite support uses `modernc.org/sqlite` (CGO-free, pure Go) to preserve the
  scratch-image container build; the dependency is added to `go.mod` as part of this work.
- `OUTPUT_FORMAT` defaults to `csv` when neither the environment variable nor the
  `--format` CLI flag is set. The `--format` flag takes precedence over the env var.
- A single `MIST_ORG_ID` is loaded per process; multi-org switching is out of scope.
- SSH client authentication uses password-only for this scaffold; default credentials
  are `misthelper`/`misthelper123!` (configurable via `SSH_USER`/`SSH_PASSWORD` env vars).
  Key-based auth is a future enhancement.
- API rate limiting uses a simple fixed delay (`API_RATE_LIMIT_MS`, default 200ms); the
  Python adaptive delay system (`delay_metrics.json`, `tuning_data.json`) is a future
  port and is out of scope for this scaffold.
- The SSH host key is generated at startup and stored in `data/` if not already present;
  it is not baked into the container image.
- Polyglot backends (ArangoDB, Redis) are out of scope for this scaffold; only CSV and
  SQLite are implemented.
- WebSocket operations (menu 5–8, 87–89) and destructive operations (menu 90–100) are
  out of scope; the menu dispatcher registers stubs for these ranges only.
- The web UI serves static HTML only at this stage; the full Dash/React equivalent is
  out of scope for this scaffold.
- No new features beyond what is described here will be introduced; this spec governs
  foundational scaffolding only.

---

## Clarifications

### Session 2026-05-22

- Q: Does the scaffold support multiple org IDs or one org per process? → A: Single `MIST_ORG_ID` per process; multi-org switching is out of scope (mirrors Python behavior).
- Q: How is the output format selected — env var only, CLI flag only, or both? → A: Both. `OUTPUT_FORMAT` env var as default, overridable by `--format csv|sqlite` CLI flag. Default is `csv` if neither is set. CLI flag takes precedence.
- Q: What SSH client authentication method is required for the skeleton? → A: Password only. Default credentials `misthelper`/`misthelper123!`, configurable via `SSH_USER`/`SSH_PASSWORD` env vars. Host key generated on first boot. Key-based auth is a future enhancement.
- Q: What is the required graceful shutdown ordering and timing? → A: SSH first (drain active sessions up to 30s via `sync.WaitGroup`), then web server (5s `context.WithTimeout` deadline), then API context cancel. Total maximum shutdown time: 35 seconds.
- Q: What rate limiting approach is required for the scaffold? → A: Simple fixed delay via `API_RATE_LIMIT_MS` env var (default 200ms). Python adaptive delay system (`delay_metrics.json`, `tuning_data.json`) is a future port, not part of this scaffold.
