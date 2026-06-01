# MistHelper-Go - AI Agent Instructions

Global coding standards (autonomous workflow, 5-item rule, inline comments, action logging, quality gates) are in `coding-standards.instructions.md` and apply automatically. This file adds Go-specific and MistHelper-Go-specific guidance only.

When refactoring code, avoid wrappers; restructure into proper packages and types as per project conventions.

---

## Project Overview

MistHelper-Go is a Go rewrite of [MistHelper](https://github.com/jmorrison-juniper/MistHelper) -- a production-grade tool for Juniper Mist Cloud network operations. Goal: single static binary packaged in a ~25MB container image (vs ~500MB Python equivalent), full feature parity with the Python version (193 menu operations), multi-backend output (CSV, SQLite, ArangoDB/Redis), and containerized SSH access.

**Deployment**: Container-only. MistHelper-Go runs exclusively from a Podman/Docker container (`ghcr.io/jmorrison-juniper/misthelper-go`). Direct binary execution is only used during local development (`go run ./cmd/misthelper`). There is no standalone host deployment mode.

**Target Audience**: Junior NOC engineers. Use clear, professional language without jargon. Think Fred Rogers meets NASA/JPL safety standards.

**Python-First Development Model**: MistHelper-Go **trails** the Python implementation. New features are always designed and built in [MistHelper (Python)](https://github.com/jmorrison-juniper/MistHelper) first. Only after a feature is complete and stable in Python does it get ported and refactored for Go. **AI agents must not originate new features in this repo.** If a requested feature does not already exist in `../MistHelper/MistHelper.py`, stop and direct the user to implement it in Python first.

---

## Core Architecture

### Go Project Hierarchy (5-Item Rule)
See `coding-standards.instructions.md` § Structural Discipline for limits (max 5 params, 5 blocks, 25 lines).

Go hierarchy levels:
1. **Project Root** → 2. **Packages / Directories** (`cmd/`, `internal/`, `pkg/`) → 3. **Source Files** → 4. **Types / Functions / Constants** → 5. **Methods / Fields / Expressions**

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
See `coding-standards.instructions.md` § Inline Comments for full rules. Go example:

```go
result, err := client.ListOrgSites(ctx, orgID)  // Fetch all sites for this org from Mist API
if err != nil {                                   // API call may fail on auth or network errors
    return nil, fmt.Errorf("list sites for org %s: %w", orgID, err)  // Wrap error with context for caller
}
```

### Action Logging (NON-NEGOTIABLE)
See `coding-standards.instructions.md` § Action Logging for full rules. Go example:

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
See `coding-standards.instructions.md` § Logging Standards.
- Use `log/slog` (standard library, Go 1.21+) for structured logging
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
- **Build**: Multi-stage Containerfile (build stage with `golang:1.25-alpine`, runtime stage with `alpine`)
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

### Menu Categories (Full Range: 1-193)

| Range | Category | Notes |
| - | - | - |
| 1-59 | Safe Org Exports | Sites (1-7), Inventory (8-14), Device stats (15-19), Events (20-26), Clients (27-30), Gateways (31-36), Templates (37-41), Config/Admin (42-50), SLE (51-55), Misc (56-59) |
| 60-96 | Interactive Safe | Site devices (60-72), Insights (73-79), Stats (80-91), Viewers (92-96) |
| 97-101, 153 | Resource Intensive | Long-running operations, bulk operations |
| 102-123 | WebSocket | Show commands (102-115), Diagnostics (116-123) |
| 124-150 | Interactive | Diagnostics (124-127), Management (128-133), Packet captures (134-135), Tools (136-147), Config (148-150) |
| 151-152 | Continuous | Monitoring loops |
| 154-193 | **Destructive** | Firmware (154-157), Reboots (158-160), VC (161-162), Templates (163-167), Site config (168-170), Test data (171-174), SSH runners (175-176), Clear/reset (177-187), Support tickets (188-193). **NEVER automate without explicit user confirmation.** |

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

See `coding-standards.instructions.md` for naming standards and code readability rules.
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

## Multi-Agent Git Workflow

Global workflow rules are in `git-workflow.instructions.md` (applied via `applyTo: "**"`).
This section adds MistHelper-Go-specific overrides only.

### MistHelper-Go-Specific Error-to-Issue Triggers

| Trigger | Label(s) | Issue Title Pattern |
|---------|----------|---------------------|
| `go vet` finding | `lint`, category | `Vet: <finding>` |
| `golangci-lint` violation | `lint`, rule code | `Lint: <rule> -- <description>` |
| `go test` failure | `bug`, `test` | `Test failure: <test_name>` |
| Runtime panic | `bug` | `Runtime: <panic> in <function>` |
| Security finding | `security` | `Security: <tool> -- <finding>` |
| CI pipeline failure | `ci` | `CI: <workflow> -- <failure>` |

### Required Labels

Every issue and PR MUST have at least:
1. A **type** label: `bug`, `feature`, `chore`, `lint`, `security`, `refactor`
2. A **scope** label: `api`, `menu`, `output`, `ssh`, `web`, `tests`, `ci`, `container`, `docs`
3. A **status** label when in progress: `in-progress`

### Fleet Coordination (MistHelper-Go-Specific)

See `git-workflow.instructions.md` § Agent Coordination for general rules. MistHelper-Go additions:

- **Auto-merge label**: Wait for **CodeQL** (~2-3 min) before adding. Use `gh pr checks <pr> --watch`.

### Agent Worktree Examples

```
MistHelper-Go/                     # main checkout (human or merge agent only)
../MistHelper-Go-agent-1/          # worktree for Agent 1 (feat/101-new-menu)
../MistHelper-Go-agent-2/          # worktree for Agent 2 (fix/102-rate-limit)
```

### Windows Branch Switching & Post-Merge Fix Timing

See `git-workflow.instructions.md` § Windows Branch Switching and § Post-Merge Fix Timing.

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

## Copilot Token Efficiency

See `copilot-token-efficiency.instructions.md` (applied globally via `applyTo: "**"`).

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

See `coding-standards.instructions.md` § Security Findings. Project-specific tools: gosec, govulncheck, CodeQL.
Use `//nolint` only for verified false positives with a justification comment.

### Delivery Artifacts (Per Release Tag)

1. **Container image** -- multi-arch (amd64/arm64) pushed to GHCR (~25MB alpine-based). This is the sole distribution artifact.

---

## Complexity-Driven SpecKit Escalation

See `git-workflow.instructions.md` § SpecKit Escalation for the full decision tree.

**MistHelper-Go-specific escalation triggers**:
- Changes touching 3+ files or 2+ packages
- New menu operations or API integrations
- Any change to destructive operations (menu 90-100)
- Database schema or primary key strategy changes

---

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at specs/003-org-inventory-port/plan.md
<!-- SPECKIT END -->

