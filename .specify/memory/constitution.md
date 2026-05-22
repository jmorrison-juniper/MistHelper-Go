<!-- Global coding standards (5-Item Rule, architecture,
     safety-first input, logging, quality gates) are defined in the
     user-level VS Code instructions file:
       %APPDATA%/Code/User/prompts/coding-standards.instructions.md
     This constitution extends those global standards with
     MistHelper-Go-specific principles and constraints. -->

# MistHelper-Go Constitution

## Core Principles

### I. Five-Item Rule (Structural Discipline)

Every level of the project hierarchy MUST contain no more than five
children. Violations MUST be resolved by extracting into sub-levels
before merging.

Hierarchy levels (largest to smallest):
1. Project Root
2. Packages / Directories (`cmd/`, `internal/`, `pkg/`)
3. Source Files (individual `.go` files within packages)
4. Types / Functions / Constants (top-level constructs in files)
5. Methods / Fields / Expressions (struct methods and function bodies)

Function and method hard limits:
- **Max 5 parameters** per function. If more are needed, use an options
  struct, or split into multiple functions.
- **Max 5 logical blocks** per function body (an if/else counts as one
  block, a for-loop counts as one block, etc.). If exceeded, extract
  blocks into helper functions.
- **Max 5 operations** per statement block. Complex expressions MUST be
  broken into intermediate variables.
- **Max 25 lines** per function (5 blocks x ~5 lines). If longer,
  extract logical sections into helper functions.

**Rationale**: Keeps code navigable, reviewable, and maintainable for
junior NOC engineers who are the primary audience.

### II. Package-Based Architecture (No Wrappers)

All functionality MUST live within semantically named packages and
types (interfaces + structs). Standalone wrapper functions that merely
delegate to a type method are prohibited. When refactoring, code MUST
be restructured into proper packages — not wrapped.

Package examples from the codebase:
`internal/api`, `internal/menu`, `internal/output`,
`internal/ssh`, `internal/web`.

Type and interface naming MUST be descriptive:
`APIClient`, `MenuHandler`, `OutputWriter`, `SSHServer`, `WebHandler`.

Variable and iterator naming MUST use full words — no abbreviations:
`for _, device := range devices` NOT `for _, d := range devices`.

Only export (capitalize) symbols that other packages need to use.
Start unexported, promote to exported as required.

AI-generated marker text (`...existing code...`, double ellipses) MUST
never appear in committed code.

**Rationale**: Go's package system provides clear ownership,
discoverability, and testability. Full names reduce cognitive load for
operators reading unfamiliar code.

### III. Safety-First (NON-NEGOTIABLE)

All input handling MUST use the `safeInput()` pattern with EOF handling
and context logging. Every `bufio.Reader.ReadString()` call in
SSH/container contexts, destructive confirmations, and interactive menus
MUST use this wrapper:

```go
func safeInput(reader *bufio.Reader, prompt string, context string) (string, error) {
    fmt.Print(prompt)                                                // Display prompt to user
    input, err := reader.ReadString('\n')                            // Read until newline from stdin
    if err != nil {                                                  // Check for EOF or read errors
        log.Printf("EOF detected in %s - session disconnected", context)
        return "", fmt.Errorf("input EOF in %s: %w", context, err)  // Return wrapped error for caller
    }
    return strings.TrimSpace(input), nil                             // Strip whitespace from user input
}
```

Destructive operations (firmware upgrades, reboots, VC conversions,
device command execution — menu items 90-100) MUST require explicit
typed confirmation following the NASA/JPL pattern:
```go
confirmation, err := safeInput(reader, "Type 'UPGRADE' to proceed: ", "firmware_upgrade")
if err != nil || confirmation != "UPGRADE" {                         // Validate confirmation string
    return  // Early return on validation failure
}
```

All external inputs MUST be validated before use (reject path traversal,
special characters, etc.). The pattern is: **validate early, return
early** — never proceed with unvalidated data.

Secrets and credentials MUST never appear in logs, outputs, or error
messages. API tokens and passwords MUST be redacted at the logging
boundary.

Always pass `context.Context` for cancellation and timeouts. Never
launch goroutines without a way to stop them:
```go
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()
```

**Rationale**: MistHelper-Go operates in production NOC environments via
SSH containers. EOF from disconnected sessions, accidental destructive
commands, and credential exposure are real operational risks that MUST
be mitigated at the code level.

### IV. Full Deployment Pipeline (NON-NEGOTIABLE)

After any code change, the complete deployment pipeline MUST be
executed. No steps may be skipped.

1. **Validate** — The following MUST all pass before any commit:
   - `go vet ./...` — zero static analysis findings
   - `go build ./...` — clean compile (type safety enforced by compiler)
   - `golangci-lint run ./...` — zero lint violations
   - `go test ./... -race -cover` — all tests pass with race detector
2. **Commit** — Message format:
   `<type>(<scope>): <description>\n\nCloses #<issue>` (Conventional
   Commits). Version entries in CHANGELOG.md use
   `YY.MM.DD.HH.MM` format (UTC timestamp).
3. **Push** — `git push origin <branch>` then create PR via
   `gh pr create`. Pushing `main` after merge triggers container build.
4. **Wait for CI** — `gh pr checks <pr-number> --watch` until all
   checks including CodeQL are green before adding `auto-merge` label.
5. **Pull image** —
   `podman pull ghcr.io/jmorrison-juniper/misthelper-go:latest`
6. **Restart container** — Stop, remove, and re-run with volume mounts.
7. **Verify** — `podman ps` confirms the container is healthy.

Every changelog update triggers this pipeline. There are no standalone
git operations.

**Rationale**: The user expects the running container to reflect the
latest code after every change. Partial deployments leave the
production environment in an inconsistent state.

### V. Observability & Logging

All log output MUST use ASCII characters only. Unicode characters
(including emoji) MUST be replaced with ASCII substitutions for
cross-platform compatibility.

Use `log/slog` (Go 1.21+ standard library) for structured logging.
Logging levels MUST follow these standards:
- **Debug**: Internal state changes, raw API responses
- **Info**: User-facing progress messages
- **Error**: Error context with full detail

Structured, machine-parseable key-value log entries via `slog` are
required for all packages:
```go
slog.Info("Fetching device list", "site_id", siteID)
slog.Debug("Received devices from API", "count", len(result))
slog.Error("Failed to fetch devices", "site_id", siteID, "error", err)
```

Secrets MUST never be logged — this is enforced at the logging
boundary, not at the caller.

**Rationale**: MistHelper-Go runs in heterogeneous environments (Windows
local dev, Linux containers, SSH sessions). ASCII-only logging prevents
encoding failures. Structured logs enable automated monitoring and
incident correlation.

### VI. Inline Comments (NON-NEGOTIABLE)

Every line of AI-generated code MUST have an inline comment on the
same line explaining what it does and why. This is not optional and
MUST NOT be skipped under any circumstances.

Comments MUST explain *why* and *what for*, not just restate the code.
Blank lines, closing braces/parens, and package/import declarations
are exempt.

When modifying existing code, inline comments MUST be added to the
changed lines AND to any adjacent uncommented lines in the same block.
When existing code is found lacking inline comments during any edit,
comments MUST be added to the entire function or block being touched.

```go
// WRONG: No comments or restating the code
result, err := client.ListOrgSites(ctx, orgID)  // list sites

// CORRECT: Explaining intent and context
result, err := client.ListOrgSites(ctx, orgID)  // Fetch all sites for this org from Mist API
if err != nil {                                  // API call may fail on auth or network errors
    return nil, fmt.Errorf("list sites for org %s: %w", orgID, err)  // Wrap error with context for caller
}
```

Code without inline comments is considered incomplete and MUST NOT
be committed, merged, or deployed.

**Rationale**: Junior NOC engineers are the primary maintainers of
this codebase. Every line must be self-explanatory without external
context. Inline comments eliminate guesswork and reduce onboarding
time from days to hours.

### VII. Action Logging (NON-NEGOTIABLE)

Every meaningful action in AI-generated code MUST have a logging
statement BEFORE and AFTER execution. This enables operators to trace
exactly what happened during any run.

- Log an `Info` message BEFORE every action (API call, file write,
  database operation, data transformation, user prompt).
- Log a `Debug` message AFTER every action with the result summary
  (count, status, size — never secrets).
- Log `Error` with full context on any exception or error return.
- Use structured key-value pairs with `slog` (not string formatting).

When modifying existing code, if the function or block being touched
lacks action logging, logging MUST be added to the entire function or
block.

```go
// WRONG: No logging around actions
result, err := client.ListSiteDevices(ctx, siteID, "all")
processed := flattenDevices(result)

// CORRECT: Log before and after every action
slog.Info("Fetching device list", "site_id", siteID)           // Log before API call
result, err := client.ListSiteDevices(ctx, siteID, "all")       // Call Mist API for all devices
if err != nil {                                                  // Handle API failure
    slog.Error("Failed to fetch devices", "site_id", siteID, "error", err)
    return nil, err                                              // Propagate error to caller
}
slog.Debug("Received devices from API", "count", len(result))   // Log result count
slog.Info("Flattening device response data")                     // Log before data transform
processed := flattenDevices(result)                              // Normalize nested JSON
slog.Debug("Flattened device records", "count", len(processed)) // Log output count
```

Code without action logging is considered incomplete and MUST NOT be
committed, merged, or deployed.

**Rationale**: When a NOC engineer reports "it broke at step 3," the
logs must show exactly what happened before, during, and after step 3.
Code without logging is code without observability.

## Technology & Compatibility Constraints

The following technology choices are binding for all MistHelper-Go code:

- **Go**: 1.21 or newer. No code may target older Go versions.
- **mistapi-go**: v0.4.73+ (Thomas Munzer's Go Mist API SDK). This is
  the sole interface to the Juniper Mist Cloud API. Direct HTTP calls
  to Mist endpoints are prohibited when a mistapi-go method exists.
- **godotenv**: v1.5.1 for `.env` file loading.
- **Module management**: `go mod` is the standard. All dependencies
  tracked in `go.mod` and `go.sum`. No vendoring unless explicitly
  required.
- **Deployment**: Container-only. MistHelper-Go runs exclusively from a
  Podman/Docker container. Direct binary execution (`go run`) is only
  permitted during local development. There is no standalone host
  deployment mode — do not add systemd service files, standalone run
  scripts, or cross-compiled release binaries.
- **Container Runtime**: Podman is the primary runtime. Docker is
  compatible but all documentation and examples MUST use Podman.
- **File Paths**: MUST use `filepath.Join()`. Never hardcode `/` or
  `\\` separators. Windows compatibility is required.
- **Output Backends**: All data operations MUST support multi-backend
  output (CSV, SQLite, and polyglot ArangoDB/Redis) via the
  `output.Writer` interface.
- **Database Keys**: Natural business keys from the Mist API (not
  artificial IDs). Primary key strategy MUST be defined in the
  endpoint strategies map before implementing any new operation.
- **Data Directory**: All outputs MUST go to the `data/` directory,
  enforced at runtime. SSH logs go to `data/per-host-logs/`.
  Database file is `data/mist_data.db`.
- **Container Security**: The container runs as non-root user
  (`misthelper`). The mounted `data/` directory MUST be writable
  before first run.
- **Zscaler/Proxy**: Local `podman push` behind corporate Zscaler is
  blocked. All container builds and pushes MUST use GitHub Actions CI.
- **Logging**: `log/slog` (Go standard library, 1.21+). No third-party
  logging libraries.
- **Error handling**: Always wrap errors with
  `fmt.Errorf("context: %w", err)`. Never discard errors with `_`
  unless explicitly documented.

## Development Workflow & Quality Gates

### Adding New Menu Operations

Every new operation MUST follow this sequence:
1. **API Discovery** — Check `mistapi-go` package for available SDK
   methods corresponding to the Mist API endpoint.
2. **Primary Key Strategy** — Add entry to the endpoint strategies map
   with the appropriate type (natural_pk, composite_pk, or
   auto_increment_with_unique).
3. **Flatten JSON** — Use existing flatten helpers in `internal/api/`
   for nested API response structures.
4. **Multi-Backend Output** — Use the `output.Writer` interface to
   support CSV, SQLite, and ArangoDB/Redis backends.
5. **Update README** — Modify the operation count and add the new
   operation to the menu table.
6. **Version Changelog** — Add entry to `CHANGELOG.md` with
   `YY.MM.DD.HH.MM` format (UTC timestamp).
7. **Execute Full Pipeline** — Run the complete deployment pipeline
   (Principle IV).

### Testing

- **Local development**: Windows 11 PowerShell.
- **Run all tests**: `go test ./... -race -cover -v`
- **Single test**: `go test ./internal/api/... -run TestSpecificFunction -v`
- **Benchmarks**: `go test -bench=. ./...`
- **Skip list**: Menu 63-65 (WIP), 90-100 (destructive) are excluded
  from automated tests.
- **Compile check**: `go build ./...` MUST pass before every commit
  (enforced by Principle IV).

### Security Findings: Fix Over Suppress (NON-NEGOTIABLE)

Security tool findings (gosec, govulncheck, CodeQL) MUST be
**resolved**, not suppressed:

1. **Fix the root cause** — Rewrite code to eliminate the vulnerability
   (e.g., validate inputs, use parameterized queries).
2. **Refactor to avoid the pattern** — Restructure so the flagged
   pattern is not needed (e.g., move a secret default to
   `os.Getenv()` directly).
3. **`//nolint` only for verified false positives** — When the tool
   misidentifies safe code (e.g., an intentional `0.0.0.0` bind gated
   by `isRunningInContainer()`). The annotation MUST include a
   justification comment.

Never use `//nolint`, `// #nosec`, `// type: ignore`, or similar
suppressions as a shortcut to silence legitimate findings. If a
finding requires more than a trivial fix, create a GitHub issue
and track it.

**Rationale**: Suppressions hide risk. Fixes eliminate it. This
codebase operates in production NOC environments where security
findings left unresolved become real attack surfaces.

### Documentation

- **README.md**: User-facing operations guide. MUST be updated for
  every new operation or behavior change.
- **agents.md**: Internal VS Code Chat coding guide. MUST be consulted
  before making architectural decisions.
- **.github/copilot-instructions.md**: Full project guide. The primary
  reference for AI agents.
- **Version format**: `YY.MM.DD.HH.MM` (UTC timestamp), consistent
  across changelog entries, commit messages, and container tags.

### Audience Standard

All user-facing text MUST be written for junior NOC engineers. Use
clear, professional language without jargon. The standard is:
"Fred Rogers meets NASA/JPL safety standards."

## Complexity-Driven SpecKit Escalation (NON-NEGOTIABLE)

Not every task needs full ceremony. Use this decision tree:

**Implement directly** (no spec needed):
- Single-file edits with obvious intent (typo, log message, config)
- Lint/format auto-fixes
- Documentation-only changes
- Adding a test for well-understood behavior

**Escalate to SpecKit** (spec required before coding):
- Changes touching 3+ files or 2+ packages
- New menu operations or API integrations
- Architectural changes (new packages, interface changes, data flow)
- Bug fixes where root cause is unclear or spans multiple packages
- Any change to destructive operations (menu 90-100)
- Performance or concurrency work
- Database schema or primary key strategy changes

**Rationale**: Underpowered models lose track of multi-step
implementations without structured artifacts. The spec anchors intent,
the plan decomposes complexity, and tasks provide checkpoint-by-
checkpoint execution any model can follow. Even capable models benefit
from the spec as a contract preventing scope drift.

Workflow: `speckit.specify` -> `speckit.clarify` (recommended) ->
`speckit.plan` -> `speckit.tasks` -> `speckit.implement` ->
`speckit.analyze`.

If in doubt, escalate. A spec that turns out unnecessary costs
minutes. A botched multi-file change without a spec costs hours.

## Multi-Agent Git Workflow (NON-NEGOTIABLE)

The global coding standards
(`%APPDATA%/Code/User/prompts/coding-standards.instructions.md`)
define the general multi-agent workflow. This section adds
MistHelper-Go-specific enforcement.

### Issue-First Error Pipeline

When any error is detected during development, an issue MUST be
created before attempting a fix:

```powershell
# go vet finding example
gh issue create --title "Vet: printf args mismatch in internal/api/client.go" `
  --label "lint,api" `
  --body "go vet output:`n$(go vet ./...)"

# Test failure example
gh issue create --title "Test failure: TestListOrgSites" `
  --label "bug,test" `
  --body "go test output:`n$(go test ./internal/api/... -run TestListOrgSites -v)"
```

### Branch Naming for MistHelper-Go

Branches MUST follow this pattern and target `main` directly:
- `fix/<issue-number>-<slug>` — bug fixes (e.g., `fix/42-clear-session`)
- `feat/<issue-number>-<slug>` — features (e.g., `feat/50-list-switches`)
- `chore/<issue-number>-<slug>` — maintenance (e.g., `chore/38-lint-fixes`)

**Never branch from another feature branch.** Every branch starts from
`main`.

### Required Labels

Every issue and PR MUST have at least:
1. A **type** label: `bug`, `feature`, `chore`, `lint`, `security`,
   `refactor`
2. A **scope** label: `api`, `menu`, `output`, `ssh`, `web`, `tests`,
   `ci`, `container`, `docs`
3. A **status** label when in progress: `in-progress`

### Fleet Coordination Rules

When multiple agents work on MistHelper-Go simultaneously:

1. **Claim first**: Add `in-progress` label to the issue before
   creating a branch. If another agent already claimed it, pick a
   different issue.
2. **File overlap check**: Run
   `gh pr list --json files --jq '.[].files[].path'`
   to see what files other open PRs touch. Avoid overlapping files.
3. **Package isolation**: Unlike the Python monolith, MistHelper-Go is
   split across packages. Multiple agents can work on different
   packages simultaneously (e.g., one on `internal/api/`, another on
   `internal/output/`) without conflict.
4. **Rebase before push**: Always `git rebase main` before pushing to
   ensure the branch is current.
5. **Squash merge only**: All PRs merge to `main` via squash merge.
   This keeps history linear and readable.
6. **CodeQL wait**: NEVER add `auto-merge` label before CodeQL
   completes. Use `gh pr checks <pr-number> --watch` to confirm.

### Agent Isolation (One Agent = One Worktree)

Every concurrent AI agent MUST operate in its own isolated worktree:
```powershell
# Setup
git worktree add ../MistHelper-Go-<slug> -b <type>/<issue>-<slug> main
cd ../MistHelper-Go-<slug>

# Teardown after merge
cd ../MistHelper-Go
git worktree remove ../MistHelper-Go-<slug>
git checkout main && git pull origin main
git branch -D <type>/<issue>-<slug>
```

### PR Checklist Enforcement

Every PR MUST include in its description:
- `Closes #<issue-number>` (auto-closes the linked issue on squash merge)
- CI status confirmation (all quality gates green including CodeQL)
- Files changed summary (to help detect overlap)
- `auto-merge` label added only after all checks pass

## Governance

This constitution is the authoritative source for MistHelper-Go project
rules. It supersedes all other practice documents when conflicts arise.

**Amendment procedure**:
1. Propose the change with rationale in a commit message or discussion.
2. Update this constitution file with the new or modified principle.
3. Increment the version according to semantic versioning:
   - **MAJOR**: Principle removal or backward-incompatible redefinition.
   - **MINOR**: New principle or materially expanded guidance added.
   - **PATCH**: Clarification, wording, or typo fix.
4. Update `LAST_AMENDED_DATE` to the amendment date.
5. Verify that dependent templates (plan, spec, tasks) remain
   consistent with the updated principles.
6. Execute the full deployment pipeline (Principle IV) if code changes
   accompany the amendment.

**Compliance review**: Every PR and code review MUST verify adherence
to all seven Core Principles. Complexity that violates a principle MUST
be justified in writing (Complexity Tracking table in plan.md).
Principles VI (Inline Comments) and VII (Action Logging) are
non-negotiable quality gates — code lacking either MUST NOT pass
review, regardless of other merits.

**Runtime guidance**: `agents.md` provides daily coding quick reference.
`.github/copilot-instructions.md` is the comprehensive project guide.
The constitution provides the non-negotiable rules; those files provide
the how-to.

**Python-First Development Model**: MistHelper-Go trails the Python
implementation at `../MistHelper/MistHelper.py`. New features are always
built and stabilized in Python first; Go ports follow. This is a hard
constraint — a SpecKit spec for MistHelper-Go must reference the
existing Python operation it is porting. Specs for net-new features
belong in the Python repo, not here.

**Version**: 1.0.0 | **Ratified**: 2026-05-21 | **Last Amended**: 2026-05-21
