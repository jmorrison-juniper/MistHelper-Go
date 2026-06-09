# Tasks: SSH Wish Migration

**Input**: Design documents from `/specs/002-ssh-wish-migration/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/ssh-wish-session-contract.md`, `quickstart.md`

**Execution Policy (Critical)**:

- Do **not** compile, run tests, lint, or run containers during implementation phases.
- Compile/run is allowed **only** in the dedicated Validation phase.
- Validation must run against a **local container** workflow, not GitHub container build completion.

## Phase 1: Setup (Shared Migration Context)

**Purpose**: Establish migration scaffolding and explicit parity targets before code changes.

- [ ] T001 Create SSH migration parity matrix mapping FR-001..FR-010 and SC-001..SC-005 in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T002 Add Wish runtime dependencies to `go.mod` and `go.sum` (`github.com/charmbracelet/wish`, `github.com/charmbracelet/ssh`)
- [ ] T003 [P] Add explicit Wish migration notes and no-run-until-validation guardrails to `specs/002-ssh-wish-migration/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared SSH transport foundation required by all user stories.

**CRITICAL**: Complete this phase before user-story implementation.

- [ ] T004 Refactor `internal/ssh/server.go` server scaffolding to Wish-oriented lifecycle while preserving exported `Server` API (`NewServer`, `ListenAndServe`, `Shutdown`)
- [ ] T005 [P] Implement auth/config adapter from `api.Config` (`SSHUser`, `SSHPassword`, `SSHPort`) to Wish auth hooks in `internal/ssh/server.go`
- [ ] T006 [P] Implement host key integration path using existing `internal/ssh/hostkey.go` from Wish server setup in `internal/ssh/server.go`
- [ ] T007 Remove/retire obsolete low-level `golang.org/x/crypto/ssh` channel/session handling paths in `internal/ssh/server.go` while keeping only required key-parsing usage
- [ ] T008 [P] Update constructor/startup wiring compatibility in `cmd/misthelper/main.go` for any `internal/ssh.NewServer` signature changes
- [ ] T009 Establish lifecycle logging parity anchors (startup, auth attempt/result, session start/end, error) in `internal/ssh/server.go`

**Checkpoint**: Wish transport foundation is in place; user stories can proceed.

---

## Phase 3: User Story 1 - Login to Menu Without Friction (Priority: P1) MVP

**Goal**: Authenticated SSH users land directly in the interactive menu with existing auth/config behavior preserved.

**Independent Test**: Valid credentials over SSH land directly in menu with expected initial menu text; invalid credentials are rejected.

### Tests for User Story 1

- [ ] T010 [P] [US1] Add auth compatibility unit tests for username/password acceptance/rejection semantics in `internal/ssh/server_test.go`
- [ ] T011 [P] [US1] Add integration test for valid login direct-to-menu landing and menu render assertions in `internal/ssh/server_integration_test.go`
- [ ] T012 [P] [US1] Add integration test for invalid credential rejection and no-menu-session behavior in `internal/ssh/server_integration_test.go`

### Code Migration for User Story 1

- [ ] T013 [US1] Implement Wish listener bind behavior honoring `SSHPort` config in `internal/ssh/server.go`
- [ ] T014 [US1] Implement password auth callback parity using `SSHUser`/`SSHPassword` in `internal/ssh/server.go`
- [ ] T015 [US1] Implement ForceCommand-like post-auth direct menu session handoff in `internal/ssh/server.go`
- [ ] T016 [US1] Preserve menu dispatcher wiring (`menu.NewDispatcher`) and output writer bridge in `internal/ssh/server.go`
- [ ] T017 [US1] Preserve startup/auth/session-start logging intent parity in `internal/ssh/server.go`

**Checkpoint**: US1 is implementation-complete and independently verifiable in validation phase.

---

## Phase 4: User Story 2 - Reliable Interactive Input Dispatch (Priority: P1)

**Goal**: Numeric menu input + Enter dispatches exactly once with stable echo/newline behavior.

**Independent Test**: Valid option + Enter dispatches one handler call; unknown option returns expected feedback and keeps the session active.

### Tests for User Story 2

- [ ] T018 [P] [US2] Add integration test for single-dispatch behavior on valid numeric option + Enter in `internal/ssh/server_integration_test.go`
- [ ] T019 [P] [US2] Add integration test for unknown-option feedback with active session continuity in `internal/ssh/server_integration_test.go`
- [ ] T020 [P] [US2] Add PTY/input unit tests for CR/LF normalization and echo consistency in `internal/ssh/server_extra_test.go`

### Code Migration for User Story 2

- [ ] T021 [US2] Implement Wish PTY/session stream adapter for line-oriented dispatcher input in `internal/ssh/server.go`
- [ ] T022 [US2] Implement Enter handling normalization (CR/LF variants) to guarantee one dispatch per submitted line in `internal/ssh/server.go`
- [ ] T023 [US2] Ensure unknown-option handling remains menu-owned while transport preserves active session in `internal/ssh/server.go`
- [ ] T024 [US2] Add dispatch-path logging parity (before/after dispatch, no secrets) in `internal/ssh/server.go`

**Checkpoint**: US2 input reliability behavior is implemented and ready for validation.

---

## Phase 5: User Story 3 - Clean Exit and Session Stability (Priority: P2)

**Goal**: Ctrl+C/Ctrl+D/disconnect terminate sessions cleanly while preserving server readiness and per-session isolation.

**Independent Test**: Termination signals and disconnect paths exit cleanly; reconnect succeeds; per-session directories remain isolated.

### Tests for User Story 3

- [ ] T025 [P] [US3] Add integration tests for Ctrl+C, Ctrl+D, and client disconnect clean session termination in `internal/ssh/server_integration_test.go`
- [ ] T026 [P] [US3] Add integration test for post-termination reconnect readiness in `internal/ssh/server_integration_test.go`
- [ ] T027 [P] [US3] Add unit/integration tests for per-session directory creation semantics (`data/sessions/session_<id>`) in `internal/ssh/server_test.go`

### Code Migration for User Story 3

- [ ] T028 [US3] Implement per-session workspace creation before dispatcher loop in `internal/ssh/server.go`
- [ ] T029 [US3] Implement clean termination handling for Ctrl+C/Ctrl+D/disconnect in `internal/ssh/server.go`
- [ ] T030 [US3] Preserve session teardown and lifecycle end/error logging parity in `internal/ssh/server.go`
- [ ] T031 [US3] Ensure subsequent-session readiness after termination paths in `internal/ssh/server.go`

**Checkpoint**: US3 termination and session stability implementation is complete.

---

## Phase 6: Polish & Cross-Cutting Migration Cleanup

**Purpose**: Finalize migration consistency across code and tests before validation execution.

- [ ] T032 Remove/retire obsolete raw SSH protocol assumptions in legacy-focused test cases across `internal/ssh/server_test.go` and `internal/ssh/server_extra_test.go`
- [ ] T033 [P] Update migration notes and behavior parity documentation in `specs/002-ssh-wish-migration/research.md` and `specs/002-ssh-wish-migration/quickstart.md`
- [ ] T034 [P] Confirm no direct low-level `x/crypto/ssh` server/channel handling imports remain in `internal/ssh/server.go`

---

## Phase 7: Validation (First Allowed Compile/Run Stage)

**Purpose**: Execute all compile/run actions only now, including mandatory local-container acceptance.

- [ ] T035 Run `go build ./...` and record outcome/evidence in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T036 Run `go vet ./...` and record outcome/evidence in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T037 Run `golangci-lint run ./...` and record outcome/evidence in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T038 Run `go test ./... -race -cover` and record outcome/evidence in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T039 Build local container image from current working tree and record image details in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T040 Run local `misthelper-go` container with `.env` and `data/` mounts and record runtime details in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T041 Execute SSH acceptance checks (login/menu render/single dispatch/unknown option/session active/Ctrl+C/Ctrl+D/disconnect/reconnect/session dir) against local container and record results in `specs/002-ssh-wish-migration/validation-report.md`
- [ ] T042 Verify lifecycle logging parity (startup/auth/session start/end/error, no secrets) from local container logs and record parity table in `specs/002-ssh-wish-migration/validation-report.md`

**Checkpoint**: Validation complete only when all Stage A/B/C/D checks in `quickstart.md` are satisfied locally.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Starts immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1; blocks all user stories.
- **Phases 3-5 (US1-US3)**: Depend on Phase 2 completion; execute in priority order (US1 -> US2 -> US3).
- **Phase 6 (Polish)**: Depends on completion of targeted user stories.
- **Phase 7 (Validation)**: Depends on completion of all implementation/test update phases; this is the only phase allowed to compile/run.

### User Story Dependencies

- **US1 (P1)**: Starts after Foundational phase.
- **US2 (P1)**: Starts after US1 migration skeleton is in place (`T013-T016`) because it relies on active Wish menu bridge behavior.
- **US3 (P2)**: Starts after US1 foundational login/session path is complete.

### Within-Story Ordering Rules

- Story tests are authored before corresponding migration tasks.
- Code migration tasks complete before story checkpoint is considered done.
- No command execution for compile/lint/test/container is permitted before Phase 7.

## Parallel Execution Examples

- **US1**: `T010`, `T011`, and `T012` can be authored in parallel; `T013-T017` then proceed sequentially.
- **US2**: `T018`, `T019`, and `T020` can be authored in parallel; `T021-T024` then proceed sequentially.
- **US3**: `T025`, `T026`, and `T027` can be authored in parallel; `T028-T031` then proceed sequentially.
- **Cross-cutting**: `T033` and `T034` can run in parallel once `T032` scope is clear.

## Implementation Strategy

### MVP First

1. Complete through **US1** (Phase 3) for direct login-to-menu parity.
2. Add **US2** input reliability improvements.
3. Add **US3** clean exit/session stability behaviors.
4. Perform cleanup in Phase 6.
5. Execute all compile/run and local-container acceptance in Phase 7 only.

### Delivery Discipline

- Keep migration slices small and file-targeted.
- Preserve menu business logic and output wiring.
- Treat validation report as the source of truth for acceptance evidence.

---

## Final Acceptance Checklist (Spec-Tied)

### Migration Acceptance Checks

- [ ] AC-001 SSH login works with current credentials (maps: FR-002, FR-003)
- [ ] AC-002 Menu renders over SSH immediately after auth (maps: FR-002, SC-002)
- [ ] AC-003 Valid numeric option + Enter dispatches handler exactly once (maps: FR-005, SC-003)
- [ ] AC-004 Unknown option returns expected feedback and session remains active (maps: FR-006)
- [ ] AC-005 Ctrl+C/Ctrl+D/session close exits cleanly and reconnect succeeds (maps: FR-007, SC-004)
- [ ] AC-006 Per-session directory semantics preserved under `data/sessions/session_<id>` (maps: FR-008)
- [ ] AC-007 Lifecycle logging semantics preserved without secrets (maps: FR-009)
- [ ] AC-008 PTY/input reliability quirks resolved (echo + Enter handling) (maps: FR-010)

### Success Criteria

- [ ] SC-001 100% pass on migration acceptance checks
- [ ] SC-002 100% direct post-login menu landing for valid credentials in validation runs
- [ ] SC-003 100% first-Enter dispatch success for tested valid numeric submissions
- [ ] SC-004 100% clean termination for Ctrl+C/Ctrl+D/disconnect with subsequent login success
- [ ] SC-005 0 regressions in config/auth semantics, dispatcher/output wiring, and session isolation

---

## Format Validation

- All implementation tasks use checklist format: `- [ ] T### [P?] [US?] Description with file path`.
- Setup/Foundational/Polish/Validation tasks intentionally omit story labels.
- User story tasks include `[US1]`, `[US2]`, or `[US3]` labels.
- Parallel-safe tasks are marked `[P]`.
