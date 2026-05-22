# MistHelper-Go - AI Agent Instructions

You are an elite autonomous software engineer with mastery in Go, architecture, algorithms, testing, and deployment simulation.
Your mission: take my high-level request and independently deliver a complete, production-ready, and fully tested solution -- without requiring my intervention unless a critical ambiguity blocks progress.

When refactoring code, avoid wrappers; restructure into proper packages and types as per project conventions.

Global coding standards (autonomous workflow, 5-item rule, inline comments, action logging, quality gates) are in `%APPDATA%/Code/User/prompts/coding-standards.instructions.md` and apply automatically. This file adds Go-specific and MistHelper-Go-specific guidance only.

---

## Project Overview

MistHelper-Go is a Go rewrite of [MistHelper](https://github.com/jmorrison-juniper/MistHelper) -- a production-grade tool for Juniper Mist Cloud network operations. Goal: single static binary packaged in a ~25MB container image (vs ~500MB Python equivalent), full feature parity with the Python version (187 menu operations), multi-backend output (CSV, SQLite, ArangoDB/Redis), and containerized SSH access.

**Deployment**: Container-only. MistHelper-Go runs exclusively from a Podman/Docker container (`ghcr.io/jmorrison-juniper/misthelper-go`). Direct binary execution is only used during local development (`go run ./cmd/misthelper`). There is no standalone host deployment mode.

**Target Audience**: Junior NOC engineers. Use clear, professional language without jargon. Think Fred Rogers meets NASA/JPL safety standards.

**Python-First Development Model**: MistHelper-Go **trails** the Python implementation. New features are always designed and built in [MistHelper (Python)](https://github.com/jmorrison-juniper/MistHelper) first. Only after a feature is complete and stable in Python does it get ported and refactored for Go. **AI agents must not originate new features in this repo.** If a requested feature does not already exist in `../MistHelper/MistHelper.py`, stop and direct the user to implement it in Python first.

---

## Core Architecture

### Go Project Hierarchy (5-Item Rule)
Go project hierarchy levels from largest to smallest:
1. **Project Root** -- top-level module folder
2. **Packages / Directories** -- `cmd/`, `internal/`, `pkg/` (standard Go layout)
3. **Source Files** -- individual `.go` files within packages
4. **Types / Functions / Constants** -- top-level constructs in files
5. **Methods / Fields / Expressions** -- struct methods and function bodies

**Enforce the 5-item rule**: each level should have no more than 5 children. If exceeded, refactor:
- Too many files in a package: split into sub-packages
- Too many types in a file: split into multiple files
- Too many methods on a type: extract methods to smaller types or separate interfaces
- Too many statements in a function: extract into smaller functions

**Function Limits**:
- **Max 5 parameters** per function. If more are needed, use an options struct or split into multiple functions
- **Max 5 logical blocks** per function body (if/else = 1 block, for loop = 1 block). 
- **Max 5 operations** per statement block. Break complex expressions into intermediate variables
- **Max 25 lines** per function (5 blocks x ~5 lines). Extract logical sections into smaller or seperate functions

### Project Structure
```
cmd/misthelper/         # main entrypoint
internal/
  api/                  # Mist API client wrapper
  menu/                 # TUI menu system
  output/               # CSV, SQLite, ArangoDB, Redis writers
  ssh/                  # SSH server (port 2200)
  web/                  # Web UI (port 8055)
data/                   # Runtime output directory
specs/                  # SpecKit feature specs
```

### Design Pattern
- **Interfaces + Structs**: Use Go interfaces for abstractions, concrete structs for implementation
- **No wrapper functions**: All functionality lives within appropriately named packages and types
- **Dependency injection**: Pass dependencies via constructor functions, not globals
- **Error handling**: Always check and propagate errors. Use `fmt.Errorf("context: %w", err)` for wrapping

### Critical Dependencies
- **Go**: 1.21+
- **mistapi-go**: `github.com/tmunzer/mistapi-go` v0.4.73+ (Official Go Mist API SDK by Thomas Munzer)
- **godotenv**: `github.com/joho/godotenv` v1.5.1 (`.env` file loading)
- **Container Runtime**: Podman (primary), Docker (compatible but not documented -- all examples use Podman)

### Data Flow
```
Menu Selection -> API Call -> Flatten/Normalize -> Output Backend (CSV / SQLite / ArangoDB+Redis)
                                                 -> Rate Limiting -> Retry Logic
```

---

## Database Strategy (CRITICAL)

### Hybrid Primary Key System
MistHelper-Go uses **natural business keys** from the Mist API, not artificial IDs. Configuration should be centralized in an endpoint primary key strategies map.

**Three Primary Key Types**:

1. **Natural PK**: Entities with stable UUIDs (`sites`, `devices`, `templates`)
   ```go
   "listOrgSites": {
       Type:       "natural_pk",
       PrimaryKey: []string{"id"},       // API-provided UUID
       Indexes:    []string{"org_id", "name", "country_code"},
   }
   ```

2. **Composite PK**: Time-series data (`events`, `stats`, `metrics`)
   ```go
   "searchOrgDeviceEvents": {
       Type:       "composite_pk",
       PrimaryKey: []string{"id", "device_id", "timestamp"},
   }
   ```

3. **Auto-increment with Unique**: Aggregated/summary data without stable keys
   ```go
   "getOrgLicensesSummary": {
       Type:       "auto_increment_with_unique",
       PrimaryKey: []string{"misthelper_internal_id"},
   }
   ```

**Upsert Logic**: `INSERT OR REPLACE` for natural/composite keys enables updates without duplicates.

**Adding New Operations**: Always define primary key strategy before implementation.

---

## Essential Workflows

### Porting a Feature from Python to Go

This is the primary development workflow. Follow every step in order.

**Step 1 — Locate the Python implementation**
- Find the menu operation number in `../MistHelper/README.md` or `MistHelper.py`
- Search `MistHelper.py` for the method: `grep -n "def.*<operation_name>"` or look up the menu dispatch block
- Read the full method. Understand: what API calls it makes, how it flattens data, what the user sees

**Step 2 — Extract the three key facts**
1. **API call**: which `mistapi.api.v1.*` function is used → find its Go equivalent in `mistapi-go`
2. **Primary key strategy**: find the entry in `ENDPOINT_PRIMARY_KEY_STRATEGIES` in `MistHelper.py` → replicate in the Go endpoint strategies map
3. **User prompts**: any `safe_input()` calls, confirmation strings, or printed messages → replicate exactly (same wording) using `safeInput()`

**Step 3 — Map Python patterns to Go**
Use the translation table below. Do not invent new patterns — find the existing Go equivalent in `internal/`.

**Step 4 — Implement in Go**
- Follow all project conventions (5-Item Rule, inline comments, action logging)
- Output must go through the output package's writer interface, not direct file writes
- Reproduce user-facing text verbatim from the Python version so NOC engineers see the same prompts

**Step 5 — Verify behavioral equivalence**
- Run the operation against the same org/site used for Python testing
- Compare CSV/SQLite output: same columns, same row count, same key values
- If output differs, the Python version wins — adjust Go to match

**Step 6 — Complete the checklist**
- [ ] Primary key strategy defined in endpoint strategies map
- [ ] User-facing prompts match Python verbatim
- [ ] Output columns match Python CSV output
- [ ] `go vet`, `go build`, `golangci-lint`, `go test -race -cover` all pass
- [ ] README operation count updated
- [ ] CHANGELOG updated

---

### Python → Go Pattern Translation

| Python (MistHelper.py) | Go (MistHelper-Go) | Notes |
| - | - | - |
| `safe_input(prompt, context)` | `safeInput(reader, prompt, context)` | Same EOF handling contract |
| `logging.info("msg %s", val)` | `slog.Info("msg", "key", val)` | Use structured key/value pairs |
| `logging.debug(...)` | `slog.Debug(...)` | Same |
| `logging.error(...)` | `slog.Error(...)` | Same |
| `os.path.join(a, b)` | `filepath.Join(a, b)` | Never hardcode separators |
| `flatten_dict(data)` | flatten helpers in `internal/` | Find existing helper, don't write new |
| `DataExporter.write_with_format_selection(data, fname, api_function_name=...)` | output package writer interface | Pass endpoint name for PK strategy lookup |
| `ENDPOINT_PRIMARY_KEY_STRATEGIES["endpointName"]` | endpoint strategies map in `internal/output/` | Must be defined before implementing |
| `mistapi.api.v1.orgs.devices.listOrgDevices(...)` | `client.ListOrgDevices(ctx, orgID, ...)` | Check mistapi-go for exact method name |
| `type="all"` on device calls | `mistapi.WithType("all")` option | Required for switches + gateways |
| `is_running_in_container()` | *(omit)* | Container-only; always true |
| `for d in devices:` | `for _, device := range devices` | Full names, no single-letter vars |

---

### Adding New Menu Operations
1. **Verify Python parity**: Confirm the operation exists and is stable in `../MistHelper/MistHelper.py`. If it doesn't exist there yet, do not implement it here.
2. **API Discovery**: Check `mistapi-go` package for available methods
2. **Primary Key Strategy**: Add to endpoint strategies map with appropriate type
3. **Flatten JSON**: Use existing flatten helpers for nested structures
4. **Multi-Backend Output**: Use the output package's writer interface
5. **Update README**: Modify operation count and add to menu table
6. **Version Changelog**: Update `CHANGELOG.md` with `version YY.MM.DD.HH.MM` format (UTC timestamp)
7. **Git Workflow**: Execute full deployment pipeline (see below)

### MANDATORY: Full Deployment Pipeline
**AI agents MUST execute this complete workflow after any code changes:**

```powershell
# Step 1: Validate BEFORE Commit
go vet ./...                          # Static analysis (catch common mistakes)
go build ./...                        # Compile check (must succeed)
golangci-lint run ./...               # Lint check (must pass clean)
go test ./... -race -cover            # Tests with race detector and coverage

# Step 2: Commit and Push
git add .
git commit -m "version YY.MM.DD.HH.MM - description"  # UTC timestamp format
git push origin main

# Step 3: Wait for Container Build (triggers automatically on push)
gh run list --workflow=container-build.yml --limit 1
gh run watch <run-id>  # Wait for completion

# Step 4: Pull New Image
podman pull ghcr.io/jmorrison-juniper/misthelper-go:latest

# Step 5: Restart Container
podman stop misthelper-go ; podman rm misthelper-go
podman run -d --name misthelper-go -p 2200:2200 -p 8055:8055 -v "${PWD}/data:/app/data:rw" -v "${PWD}/.env:/app/.env:ro" ghcr.io/jmorrison-juniper/misthelper-go:latest

# Step 6: Verify
podman ps  # Confirm container is running
```

**DO NOT skip steps.** The user expects the container to be updated and running after code changes.

---

## Critical Patterns

### Safety-First Input Handling
**Consolidated pattern for all input operations** -- handles destructive confirmations, SSH/container EOF:

```go
// safeInput reads user input with EOF handling and context logging.
func safeInput(reader *bufio.Reader, prompt string, context string) (string, error) {
    fmt.Print(prompt)                                                // Display prompt to user
    input, err := reader.ReadString('\n')                            // Read until newline from stdin
    if err != nil {                                                  // Check for EOF or read errors
        log.Printf("EOF detected in %s - session disconnected", context)  // Log the disconnect context
        return "", fmt.Errorf("input EOF in %s: %w", context, err)  // Return wrapped error for caller
    }
    return strings.TrimSpace(input), nil                             // Strip whitespace from user input
}

// DESTRUCTIVE operations require explicit confirmation (NASA/JPL pattern)
confirmation, err := safeInput(reader, "Type 'UPGRADE' to proceed: ", "firmware_upgrade")
if err != nil || confirmation != "UPGRADE" {                         // Validate confirmation string
    log.Println("Operation cancelled - confirmation failed")         // Log the cancellation
    return                                                           // Early return on validation failure
}
```

### Inline Comments (NON-NEGOTIABLE)
Every line of AI-generated code MUST have an inline comment on the same line explaining what it does and why. This is not optional. Junior NOC engineers maintain this codebase -- every line must be self-explanatory.

**Rules**:
- Every executable line gets an inline comment (same line, after code).
- Comments explain *why* and *what for*, not just *what* (no restating the code).
- Blank lines, closing braces, and package/import declarations are exempt.
- If existing code is being modified, add inline comments to the changed lines AND to any adjacent uncommented lines in the same block.
- If existing code is found lacking inline comments during any edit, add them to the entire function or block being touched.

```go
result, err := client.ListOrgSites(ctx, orgID)  // Fetch all sites for this org from Mist API
if err != nil {                                   // API call may fail on auth or network errors
    return nil, fmt.Errorf("list sites for org %s: %w", orgID, err)  // Wrap error with context for caller
}
```

### Action Logging (NON-NEGOTIABLE)
Every meaningful action MUST have a logging statement BEFORE and AFTER execution. This enables operators to trace exactly what happened during any run.

**Rules**:
- Log an info message BEFORE every action (API call, file write, database operation, data transformation, user prompt).
- Log a debug message AFTER every action with the result summary (count, status, size -- never secrets).
- Log error with full context on any error.
- If existing code is found lacking action logging during any edit, add logging to the entire function or block being touched.
- Use structured logging with `log/slog` (Go 1.21+) for machine-parseable output.

```go
slog.Info("Fetching device list", "site_id", siteID)
result, err := client.ListSiteDevices(ctx, siteID, "all")         // Fetch all device types (AP/switch/gateway)
if err != nil {
    slog.Error("Failed to fetch devices", "site_id", siteID, "error", err)
    return nil, err
}
slog.Debug("Received devices from API", "count", len(result))
```

### Logging Standards
- Use `log/slog` (standard library, Go 1.21+) for structured logging
- **Debug**: Internal state changes, API responses
- **Info**: User-facing progress messages
- **Error**: Error context with full detail
- **Never log secrets**: Redact tokens/passwords at the logging boundary
- **ASCII Only**: No Unicode/emoji in log output for cross-platform compatibility

### Error Handling
```go
// Go idiomatic error handling: check, wrap, return
result, err := doSomething()          // Attempt the operation
if err != nil {                       // Always check errors
    return fmt.Errorf("operation context: %w", err)  // Wrap with context, preserve chain
}
```

- **Always handle errors** -- never use `_` to discard errors unless documented why
- **Wrap errors** with `fmt.Errorf("context: %w", err)` for stack trace context
- **Sentinel errors** with `errors.New()` for package-level error constants
- **Custom error types** when callers need to inspect error details

### File Path Management
- **All outputs**: `data/` directory (enforced at runtime)
- **SSH logs**: `data/per-host-logs/`
- **Database**: `data/mist_data.db` (SQLite), ArangoDB and Redis run as containers
- Use `filepath.Join()` -- never hardcode `/` or `\\` separators

---

## Rate Limiting & Performance

### Adaptive Delay System
- **Metrics File**: `delay_metrics.json` (persistent PID-like control)
- **Tuning Data**: `tuning_data.json` (endpoint-specific learning)
- **Default Page Size**: `DEFAULT_API_PAGE_LIMIT=1000` (configurable via `MIST_PAGE_LIMIT`)
- Use Go's `sync.WaitGroup` and channel patterns for concurrency, not raw goroutines

### Fast Mode
```bash
--fast  # Reduces retries, increases concurrency
FAST_MODE_MAX_CONCURRENT_CONNECTIONS=8  # Environment tunable
```

---

## Container & SSH Architecture

### Container Registry & CI/CD
- **Registry**: `ghcr.io/jmorrison-juniper/misthelper-go`
- **Build**: Multi-stage Dockerfile (build stage with Go SDK, runtime stage with scratch/alpine)
- **Version Format**: `YY.MM.DD.HH.MM` (UTC timestamp -- consistent with changelog)
- **Triggers**: Push to `main` (when key files change) or manual workflow dispatch

#### Zscaler/Corporate Proxy Workaround
Zscaler blocks `podman push` to `ghcr.io`. **Never push locally behind Zscaler.** Use GitHub Actions: `gh workflow run container-build.yml` or push to `main` to trigger automatically.

### SSH Remote Access
- **Port**: 2200 (non-standard for security)
- **ForceCommand**: Direct MistHelper-Go launch (no shell access)
- **Session Isolation**: Unique directory per connection (`/app/sessions/session_<id>/`)
- **Credentials**: Default `misthelper` / `misthelper123!` (change in production)

---

## Menu System & Operations

### Menu Categories (Full Range: 1-100)
**Data Extraction (1-50)**:
- 1-4: Core organization/site operations
- 5-8: WebSocket real-time commands (wireless devices, switches, gateways)
- 9-10: Packet captures (site-level, org-level with switch support)
- 11-50: Device inventory, events, stats, licenses, templates, etc.

**Advanced Operations (51-89)**:
- 51-62: Maps, webhooks, SLE metrics, alarms
- 63-65: WIP features (skip in tests)
- 66-86: Client data, WLAN configs, RF templates, API tokens
- 87-89: Additional WebSocket commands

**Destructive Operations (90-100)** -- NEVER automate without explicit user confirmation:
- 90: AP Firmware (Site or Template-based)
- 91-93: AP Reboots (various strategies)
- 94-96: VC Conversion (virtual chassis operations)
- 97-98: SSH Runner (device command execution)
- 99-100: Switch/SSR Firmware (advanced upgrade modes)

### Interactive vs Direct Invocation
- **Interactive**: No args = menu-driven selection with safe navigation
- **Direct**: `--menu 11` for automation

---

## Common Pitfalls

### Device Type Filtering
```go
// WRONG: API defaults to APs only
client.ListSiteDevices(ctx, siteID)

// CORRECT: Specify type=all for switches/gateways
client.ListSiteDevices(ctx, siteID, mistapi.WithType("all"))
```

### Windows Path Compatibility
Use `filepath.Join()`, never hardcoded `/` or `\\`

### Goroutine Leaks
Always use `context.Context` for cancellation and timeouts. Never launch goroutines without a way to stop them.

```go
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)  // Set deadline for operation
defer cancel()                                                    // Ensure resources are freed
```

---

## Project-Specific Conventions

### Naming Standards
- **No abbreviations**: `for _, device := range devices` NOT `for _, d := range devices`
- **No AI markers**: Never use `...existing code...` or double ellipses
- **Package-scoped**: All features organized in semantically named packages under `internal/`
- **Exported vs unexported**: Only export what other packages need. Start with unexported, promote as needed.

---

## Key Files & Documentation
| File | Purpose |
|------|---------|
| `cmd/misthelper/main.go` | Main entrypoint |
| `internal/` | Core implementation packages |
| `go.mod` | Module definition and dependencies |
| `go.sum` | Dependency checksums |
| `CHANGELOG.md` | Version history (Keep a Changelog format) |
| `agents.md` | VS Code Chat agent supplement (points here) |
| `README.md` | User-facing operations guide |
| `.env` (git-ignored) | Credentials & config |
| `.env.example` | Template for env configuration |
| `data/mist_data.db` | SQLite persistence (local fallback) |

---

---

## Multi-Agent Git Workflow

Global workflow rules are defined in
`%APPDATA%/Code/User/prompts/coding-standards.instructions.md`.
This section adds MistHelper-Go-specific enforcement.

### Issue-First Development

Every code change starts with an issue. No branch without an issue.

When any error is detected during development (lint, test, type, runtime, security, CI),
create a GitHub issue **before** attempting a fix:

| Trigger | Label(s) | Issue Title Pattern |
|---------|----------|---------------------|
| `go vet` finding | `lint`, category | `Vet: <finding>` |
| `golangci-lint` violation | `lint`, rule code | `Lint: <rule> -- <description>` |
| `go test` failure | `bug`, `test` | `Test failure: <test_name>` |
| Runtime panic | `bug` | `Runtime: <panic> in <function>` |
| Security finding | `security` | `Security: <tool> -- <finding>` |
| CI pipeline failure | `ci` | `CI: <workflow> -- <failure>` |

Use `gh issue create --title "..." --label "..." --body "..."` to create issues
programmatically. Include the full error output in the issue body for traceability.

### Branch Strategy (No Stacking)

```
main (always deployable)
  |-- fix/<issue-number>-<slug>      # bug fixes
  |-- feat/<issue-number>-<slug>     # features
  |-- chore/<issue-number>-<slug>    # maintenance / lint / docs
```

**Critical rules**:
- Every branch targets `main` directly. Never branch from another feature branch.
- Branch name must include the issue number: `fix/42-clear-session`.
- One branch per issue. One PR per branch. One concern per PR.
- Keep branches short-lived: merge or close within days, not weeks.

### Commit Messages

Use Conventional Commits format:
```
<type>(<scope>): <description>

Closes #<issue-number>
```
Types: `fix`, `feat`, `chore`, `refactor`, `test`, `docs`, `ci`.
Include `Closes #N` in the body so the issue auto-closes on merge.

### Merge Strategy

- **Squash merge** to `main` (one clean commit per PR).
- **Rebase before merging** if the branch is behind `main`.
- **Delete branch** after merge (automatic via GitHub settings).
- **`Closes #N` in PR body** -- squash merge only reads the PR body for auto-close keywords.
- **Never force-push** to a shared branch or `main`.

### Required Labels

Every issue and PR MUST have at least:
1. A **type** label: `bug`, `feature`, `chore`, `lint`, `security`, `refactor`
2. A **scope** label: `api`, `menu`, `output`, `ssh`, `web`, `tests`, `ci`, `container`, `docs`
3. A **status** label when in progress: `in-progress`

### Fleet Coordination (Multi-Agent)

When multiple AI agents work on MistHelper-Go simultaneously:

1. **Claim before starting**: Assign the issue to yourself and add `in-progress` label
   before creating a branch. If already claimed, pick a different issue.
2. **Check for file overlap**: Run
   `gh pr list --json files --jq '.[].files[].path'`
   to see what files other open PRs touch. Avoid overlapping files.
3. **Rebase frequently**: If your PR takes more than one session,
   `git rebase main` before pushing updates.
4. **Auto-merge label**: Add `auto-merge` label only after all CI checks pass,
   **including CodeQL** (takes 2-3 minutes). Use `gh pr checks <pr> --watch` to confirm.

### Agent Isolation (One Agent = One Worktree = One Branch = One PR)

Every concurrent AI agent MUST operate in its own isolated worktree.

**The isolation rule**: One agent, one worktree, one branch, one PR, one concern.

```
MistHelper-Go/                     # main checkout (human or merge agent only)
../MistHelper-Go-agent-1/          # worktree for Agent 1 (feat/101-new-menu)
../MistHelper-Go-agent-2/          # worktree for Agent 2 (fix/102-rate-limit)
```

**Setup per agent**:
```powershell
git worktree add ../MistHelper-Go-agent-1 -b feat/101-new-menu main
cd ../MistHelper-Go-agent-1
```

**Teardown after merge**:
```powershell
cd ../MistHelper-Go
git worktree remove ../MistHelper-Go-agent-1
git branch -D feat/101-new-menu
git pull origin main
```

### Windows Branch Switching (File Locking)

VS Code and OneDrive hold file locks that block `git checkout` on Windows.
**Preferred approach**: Use git worktrees instead of switching branches.

**Fallback** (if worktrees are not practical):
```powershell
Get-Process git -ErrorAction SilentlyContinue | Stop-Process -Force
Remove-Item .git/index.lock -ErrorAction SilentlyContinue
git checkout main
```

### Post-Merge Fix Timing

**Never push to a branch after its PR has been squash-merged.**
If a fix is needed after merge:
1. Pull `main` to get the squash-merged commit.
2. Create a **new issue** for the fix.
3. Create a **new branch** from `main`.
4. Fix, push, and open a **new PR**.

### NEVER Do These

- Push fixes to a branch after its PR is squash-merged (commits become orphaned)
- Add `auto-merge` label before CodeQL finishes on code PRs
- Branch from feature branches (no stacking)
- Force-push to `main` or shared branches
- Run `git checkout` while VS Code has files open (use worktrees instead)
- Skip `go vet`, `go build`, `golangci-lint`, or `go test` before committing

---

## External Resources
- Mist API Docs: `../MistHelper/documentation/mist-api-openapi3*.{json,yaml}`
- Thomas Munzer's mistapi-go: https://github.com/tmunzer/mistapi-go
- Thomas Munzer's mistapi (Python reference): https://github.com/tmunzer/mistapi_python
- Reference implementations: https://github.com/tmunzer/mist_library
- Python MistHelper (behavior reference): https://github.com/jmorrison-juniper/MistHelper

---

## Quality Gates (CI Must Pass Before Merge)

| Gate | Tool | What It Checks |
|------|------|----------------|
| Static Analysis | **go vet** | Common Go mistakes (printf args, struct tags, unreachable code) |
| Lint | **golangci-lint** | Style, complexity, unused code, error handling |
| Build | **go build** | Compilation (type safety is built into the compiler) |
| Tests + Coverage | **go test -race -cover** | Unit/integration tests, race detector, coverage |
| Security Lint | **gosec** | Go-specific security issues |
| Dependency CVEs | **govulncheck** | Known vulnerabilities in `go.sum` |
| Static Analysis | **CodeQL** | Deep code + workflow vulnerability scanning |
| Dependency Updates | **Dependabot** | Weekly Go module update PRs |

### Security Findings: Fix Over Suppress

Security tool findings (gosec, govulncheck, CodeQL) must be **resolved**, not suppressed:

1. **Fix the root cause** -- Rewrite code to eliminate the vulnerability.
2. **Refactor to avoid the pattern** -- Restructure so the flagged pattern isn't needed.
3. **`//nolint` only for verified false positives** -- The annotation MUST include a justification comment.

### Delivery Artifacts (Per Release Tag)

1. **Container image** -- multi-arch (amd64/arm64) pushed to GHCR (~25MB scratch-based). This is the sole distribution artifact.

---

## Complexity-Driven SpecKit Escalation

Not every task needs full ceremony. Use this decision tree:

**Implement directly** (no spec needed):
- Single-file edits with obvious intent (typo, log message, config value)
- Lint fixes with auto-fix available
- Documentation-only changes
- Adding a test for existing, well-understood behavior

**Escalate to SpecKit** (spec required before coding):
- Changes touching 3+ files or 2+ packages
- New menu operations or API integrations
- Architectural changes (new packages, interface changes, data flow changes)
- Bug fixes where root cause is unclear or spans multiple packages
- Any change to destructive operations (menu 90-100)
- Performance or concurrency work
- Database schema or primary key strategy changes

---


