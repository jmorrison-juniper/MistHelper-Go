# Implementation Plan: SSH Wish Migration

**Branch**: `main` | **Date**: 2026-05-24 | **Spec**: `specs/002-ssh-wish-migration/spec.md`
**Input**: Feature specification from `specs/002-ssh-wish-migration/spec.md`

## Summary

Migrate `internal/ssh/server.go` from raw `golang.org/x/crypto/ssh` channel handling to a Wish-based SSH server while preserving:

- direct post-auth menu landing (ForceCommand-like UX),
- existing `menu.Dispatcher` + `output.Writer` wiring,
- auth/config semantics from `api.Config` (`SSHUser`, `SSHPassword`, `SSHPort`),
- per-session directory lifecycle (`data/sessions/session_<id>`),
- operational lifecycle logging semantics.

The migration also explicitly resolves PTY/input reliability issues (echo/newline/Enter dispatch) by shifting terminal/session lifecycle to Wish middleware and Bubble Tea-compatible input plumbing while keeping menu business logic unchanged.

## Technical Context

**Language/Version**: Go 1.25 (repo), compatible with constitution minimum Go 1.21+  
**Primary Dependencies**: `github.com/charmbracelet/wish`, `github.com/charmbracelet/ssh` (gliderlabs fork), `github.com/charmbracelet/log` (if needed by wish adapters), existing `golang.org/x/term`/`x/crypto`  
**Storage**: Filesystem session workspaces under `data/sessions/`; existing output backends unchanged  
**Testing**: `go test ./... -race -cover` (deferred until validation stage per feature constraint)  
**Target Platform**: Linux container runtime (Podman primary), Windows local development  
**Project Type**: CLI + embedded SSH + web service  
**Performance Goals**: Maintain current interactive responsiveness and support concurrent SSH sessions without increased startup latency  
**Constraints**:

- Do not compile/run during implementation slices.
- Validate only at final validation stage.
- Final validation must execute in a local container flow (not waiting for GitHub workflow).
- Preserve current externally visible behavior except PTY/input reliability fixes.

**Scale/Scope**: `internal/ssh` transport migration + minimal call-site/config wiring updates; no menu feature additions.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
| - | - | - |
| Five-Item Rule | PASS | Planned decomposition keeps functions split across focused helpers; no monolithic replacement function. |
| Package-Based Architecture (No wrappers) | PASS | Keep logic in `internal/ssh` package types; no wrapper-only shims. |
| Safety-First Input Handling | PASS | Menu input path remains dispatcher-backed; SSH transport adaptation preserves EOF-safe behavior and early-return handling. |
| Deployment Pipeline Awareness | PASS WITH CONSTRAINT | Spec requires no compile/run until validation phase; plan enforces deferred quality gates then full validation at end. |
| Observability & Logging | PASS | Preserve start/end/auth/session lifecycle logs with structured fields. |
| Inline Comments + Action Logging | PASS | Implementation slices include comment/log parity updates in all touched SSH paths. |
| Python-First Constraint | PASS | This is transport parity hardening in Go for existing behavior, not net-new feature. |

No constitution violations requiring exceptions are planned.

## Project Structure

### Documentation (this feature)

```text
specs/002-ssh-wish-migration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── ssh-wish-session-contract.md
└── tasks.md (created later by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── misthelper/
    └── main.go

internal/
├── menu/
├── output/
└── ssh/
    ├── hostkey.go
    ├── server.go
    ├── server_test.go
    ├── server_extra_test.go
    └── server_integration_test.go
```

**Structure Decision**:

- Keep migration centered in `internal/ssh/server.go`.
- Preserve `hostkey.go` as-is except compatibility touchpoints (if any).
- Update tests under `internal/ssh/*test.go` to validate Wish behavior and PTY reliability.
- Keep `cmd/misthelper/main.go` constructor/startup signatures compatible with minimal changes.

## Dependency Update Notes

1. Add Wish stack to `go.mod`/`go.sum`:
   - `github.com/charmbracelet/wish`
   - `github.com/charmbracelet/ssh`
2. Keep `golang.org/x/crypto` only for hostkey generation/parsing if still referenced by `hostkey.go`; remove direct `x/crypto/ssh` server/channel handling from `server.go`.
3. Mark raw server path in `server.go` as replaced (not dual runtime path), preserving rollback by git revert and clear migration checkpoints.
4. Ensure no dependency introduces non-container runtime assumptions.

## File-Level Change Map

| File | Change Type | Planned Change |
| - | - | - |
| `internal/ssh/server.go` | Replace/refactor | Swap low-level `x/crypto/ssh` handshake/channel loop for Wish server + middleware pipeline; preserve auth/session/menu handoff semantics and lifecycle logging. |
| `internal/ssh/server_integration_test.go` | Refactor/add | Replace raw protocol-centric assertions with Wish session behavior tests (login→menu, PTY Enter dispatch, clean exits, unknown option behavior). |
| `internal/ssh/server_test.go` | Update | Keep auth/config tests; adapt unit boundaries around Wish-compatible server config/auth hooks. |
| `internal/ssh/server_extra_test.go` | Update | Preserve ID/config helpers; add focused cases for newline/echo handling contract where practical. |
| `cmd/misthelper/main.go` | Minimal update | Keep `NewServer` call and lifecycle flow; update only if constructor/options signatures change. |
| `go.mod` / `go.sum` | Update | Add Charmbracelet dependencies, remove stale raw SSH-only direct deps where safe. |
| `README.md` (optional if behavior note needed) | Small doc note | If needed, annotate SSH subsystem now backed by Wish without user-visible credential changes. |

## Implementation Slices (No Compile/Run Until Validation)

### Slice 1 - Wish Server Skeleton + Auth Port

- Introduce Wish server bootstrap in `internal/ssh/server.go`.
- Map host/port and auth callback to `api.Config` semantics (`SSHUser`/`SSHPassword`/`SSHPort`).
- Preserve `Server` public lifecycle (`ListenAndServe`, `Shutdown`) to minimize caller changes.
- Keep structured pre/post action logging around startup/auth/session events.

### Slice 2 - Session Lifecycle + Workspace Isolation

- Rebuild session start pipeline via Wish session handler.
- Preserve per-session ID generation + directory creation semantics (`data/sessions/session_<id>`).
- Preserve connection/session start and end logs with equivalent fields.
- Ensure cleanup path is robust on normal exit and disconnect.

### Slice 3 - Menu Dispatcher Bridge

- Adapt Wish session I/O to `menu.NewDispatcher(registry, reader, writer, outputWriter)`.
- Preserve direct login-to-menu behavior (no shell drop).
- Keep output writer wiring unchanged.
- Ensure unknown-option behavior remains menu-owned (transport unchanged).

### Slice 4 - PTY + Input Reliability Hardening

- Add PTY-aware handling strategy for Enter/newline/echo consistency.
- Ensure no double-dispatch on CR/LF combinations.
- Ensure visible user echo is correct and line submission reliably triggers one dispatch.
- Handle resize/control-signal compatibility (Ctrl+C/Ctrl+D, disconnects).

### Slice 5 - Test Suite Migration (Still No Execution Yet)

- Update unit/integration tests to reflect Wish architecture and parity expectations.
- Add compatibility-focused tests for:

  - login→menu landing,
  - numeric option+Enter one dispatch,
  - unknown option returns feedback and keeps session active,
  - Ctrl+C/Ctrl+D/disconnect clean termination,
  - per-session directory lifecycle behavior.
- Prepare local container validation script/checklist.

### Slice 6 - Validation Stage (First Compile/Run)

- Execute quality gates only now.
- Run local container-based validation flow (defined below).
- Capture parity and regression outcomes.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| - | - | - |
| Wish PTY defaults differ from prior raw handling | Input regressions (echo/Enter) | Add explicit PTY compatibility checks and transport adaptation layer in session handler. |
| Auth callback mismatch | Login failures or permissive auth | Keep single source of truth from `api.Config`; retain parity tests for accepted/rejected credentials. |
| Session lifecycle drift | Stale directories or orphan sessions | Preserve existing `newSessionID` + `prepareSessionDir` semantics and test teardown behavior. |
| Dispatcher bridge mismatch | No dispatch or duplicate dispatch | Add contract tests for single Enter -> single dispatch and unknown option feedback. |
| Logging semantic drift | Reduced operator observability | Snapshot key log events and maintain equivalent info/debug/error lifecycle points. |

## Validation Approach (Explicit Local Container Flow)

> **Policy from spec:** no compile/run until this validation stage.

### Stage A - Quality Gates (first execution point)

- Run (in order): `go build`, `go vet`, `golangci-lint`, `go test -race -cover`.
- Fix issues, then rerun until clean.

### Stage B - Local Container Build + Run

1. Build local image from working tree (no GitHub workflow dependency).
2. Stop/remove existing local `misthelper-go` container if running.
3. Run local container with:
   - SSH port mapping (`2200:2200`),
   - web/status mapping (`8055:8055`),
   - mounted `data/` and `.env`.
4. Confirm container healthy/running.

### Stage C - SSH Acceptance Flow in Local Container

- Validate with configured credentials:

  1. login lands directly in menu,
  2. menu text renders,
  3. valid numeric option + Enter dispatches exactly once,
  4. unknown option shows expected feedback and session remains active,
  5. Ctrl+C, Ctrl+D, and disconnect end session cleanly,
  6. reconnect succeeds after prior session exit,
  7. per-session directory created under `data/sessions/`.

### Stage D - Log and Compatibility Review

- Verify lifecycle log parity (startup/auth/session start/session end/error paths).
- Verify no change to output backend behavior or menu business logic.

### Exit Criteria

- All FR/SC checks from spec pass.
- No no-echo/no-dispatch regression reproduced.
- Local container flow passes end-to-end.

## Rollback Strategy

- Rollback trigger: failure of any P1 migration acceptance check.
- Rollback method: revert SSH migration commits to prior `internal/ssh/server.go` implementation.
- Keep tests/docs from migration branch for future iteration where safe.

## Post-Design Constitution Re-Check

All gates remain **PASS** after design:

- Architecture kept package-scoped.
- Auth/config semantics remain unchanged.
- Session safety and observability preserved.
- Validation sequencing respects “no compile/run until validation stage” and container-first final validation.

## Complexity Tracking

No constitution exceptions required at plan stage.
