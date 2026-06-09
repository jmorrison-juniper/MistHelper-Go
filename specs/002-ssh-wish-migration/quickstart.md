# Quickstart - Implement and Validate SSH Wish Migration

## Scope Reminder

This quickstart is for implementing `specs/002-ssh-wish-migration/spec.md` with strict sequencing:

1. Perform implementation slices first.
2. Do not compile/run during implementation.
3. Run all validation only at the final validation stage.
4. Execute final acceptance in a local container workflow.

## Implementation Sequence (No Execution)

1. Refactor `internal/ssh/server.go` to Wish-based server/session lifecycle.
2. Preserve auth/config wiring from `api.Config`.
3. Preserve menu dispatcher and output writer integration.
4. Preserve per-session directory semantics and lifecycle logging.
5. Add PTY/input compatibility handling for echo + Enter reliability.
6. Update SSH unit/integration tests for new transport architecture and parity checks.

## Validation Stage (First Execution Point)

### 1) Quality Gates

Run quality gates only now (after all implementation slices are complete):

- `go build ./...`
- `go vet ./...`
- `golangci-lint run ./...`
- `go test ./... -race -cover`

### 2) Local Container Validation (Required)

Run container validation locally, without waiting for GitHub workflow completion:

- Build local container image from current working tree.
- Run/restart local `misthelper-go` container with `.env` and `data/` mounts.
- Confirm SSH port and web/status port availability.

### 3) SSH Acceptance Checks in Local Container

Execute all checks with configured credentials:

1. Login lands directly in menu.
2. Menu text renders correctly.
3. Valid numeric option + Enter dispatches exactly once.
4. Unknown option shows expected feedback and keeps session alive.
5. Ctrl+C, Ctrl+D, and disconnect terminate cleanly.
6. Reconnect works immediately after prior termination.
7. Session directory is created under `data/sessions/`.

### 4) Observability Checks

- Confirm lifecycle logs for startup, auth, session start/end, and errors.
- Confirm no secrets are logged.

## Completion Criteria

Implementation is complete when:

- Plan acceptance checks pass in local container flow.
- No no-echo/no-dispatch regression remains.
- Existing menu/output/auth semantics remain intact.
