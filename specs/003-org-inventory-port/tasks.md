# Tasks: Org Inventory Port (Menu Option 26)

**Input**: Design documents from `specs/003-org-inventory-port/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/menu-option-26-org-inventory.md`

**Scope Lock**: Implement exactly menu option `26` (`List Org Inventory`) with required support modules only (`internal/api`, `internal/output`, `internal/menu`, `cmd/misthelper`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Task can run in parallel (different file, no incomplete dependency)
- **[Story]**: User story mapping label (`[US1]`, `[US2]`, `[US3]`)
- Every task includes explicit file path(s)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm bounded implementation surface and prepare minimal test scaffolding.

- [ ] T001 Confirm/lock inventory SDK compatibility and imports in `go.mod`
- [ ] T002 [P] Add option-26 API test scaffolding in `internal/api/client_test.go`
- [ ] T003 [P] Add option-26 dispatcher test scaffolding in `internal/menu/dispatcher_test.go`
- [ ] T004 [P] Add option-26 CLI wiring test scaffolding in `cmd/misthelper/main_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared output strategy contract required by all stories.

**CRITICAL**: No user story implementation starts until these tasks complete.

- [ ] T005 Register endpoint strategy key `getOrgInventory` in `internal/output/strategies.go`
- [ ] T006 [P] Add strategy registration/shape tests for `getOrgInventory` in `internal/output/strategies_test.go`
- [ ] T007 [P] Add writer endpoint-key assertion test coverage for `getOrgInventory` in `internal/output/writer_test.go`

**Checkpoint**: Output strategy foundation is complete and validated for menu option 26 routing.

---

## Phase 3: User Story 1 - Direct dispatch for `--menu 26` (Priority: P1) MVP

**Goal**: Make direct CLI dispatch run option 26 end-to-end using API retrieval + `output.Writer` routing.

**Independent Test**: Running direct dispatch tests proves `--menu 26` executes option 26 only, fetches org inventory, and exports with endpoint key `getOrgInventory`.

### Tests for User Story 1 (write first, fail first)

- [ ] T008 [P] [US1] Add API success/empty/error tests for org inventory retrieval in `internal/api/client_test.go`
- [ ] T009 [P] [US1] Add direct dispatcher option-26 route test in `internal/menu/dispatcher_test.go`
- [ ] T010 [P] [US1] Add `--menu 26` command wiring test in `cmd/misthelper/main_test.go`
- [ ] T011 [P] [US1] Add menu-to-writer endpoint-key assertion test for `getOrgInventory` in `internal/menu/dispatcher_test.go`

### Implementation for User Story 1

- [ ] T012 [US1] Implement org inventory API client method with parity options (`vc=true`, pagination limit behavior) in `internal/api/client.go`
- [ ] T013 [US1] Implement option-26 direct dispatch handler path in `internal/menu/dispatcher.go`
- [ ] T014 [US1] Wire option-26 export through `output.Writer` using endpoint key `getOrgInventory` in `internal/menu/dispatcher.go`
- [ ] T015 [US1] Ensure deterministic CLI success/failure propagation for direct option 26 in `cmd/misthelper/main.go`

### Validation for User Story 1

- [ ] T016 [US1] Run focused direct-dispatch tests for `internal/api/client_test.go`, `internal/menu/dispatcher_test.go`, and `cmd/misthelper/main_test.go`

**Checkpoint**: User Story 1 is independently functional (MVP).

---

## Phase 4: User Story 2 - Interactive selection for option 26 (Priority: P2)

**Goal**: Ensure interactive menu selection `26` executes the same operational path and output behavior as direct mode.

**Independent Test**: Interactive selection tests confirm option 26 dispatches correctly and yields equivalent routing/outcome to direct mode.

### Tests for User Story 2 (write first, fail first)

- [ ] T017 [P] [US2] Add interactive input path test for selecting option 26 in `internal/menu/dispatcher_test.go`
- [ ] T018 [P] [US2] Add menu display label assertion test for `26. List Org Inventory` in `internal/menu/display_test.go`
- [ ] T019 [P] [US2] Add interactive/direct equivalence assertion for endpoint key and status in `internal/menu/dispatcher_test.go`

### Implementation for User Story 2

- [ ] T020 [US2] Wire interactive selection path for option 26 to shared handler logic in `internal/menu/dispatcher.go`
- [ ] T021 [US2] Ensure option-26 display label remains exactly `List Org Inventory` in `internal/menu/display.go`

### Validation for User Story 2

- [ ] T022 [US2] Run focused interactive/menu tests for `internal/menu/dispatcher_test.go` and `internal/menu/display_test.go`

**Checkpoint**: User Story 2 is independently functional and consistent with US1 behavior.

---

## Phase 5: User Story 3 - Python-first parity boundaries and non-26 stability (Priority: P3)

**Goal**: Enforce strict scope boundary (only option 26) and preserve existing non-26 stub behavior.

**Independent Test**: Regression tests show non-26 operations are unchanged and option-26 payload/flow does not introduce net-new business logic.

### Tests for User Story 3 (write first, fail first)

- [ ] T023 [P] [US3] Add non-26 dispatch regression tests (adjacent options and unknown-option behavior) in `internal/menu/dispatcher_test.go`
- [ ] T024 [P] [US3] Add parity-boundary assertions for inventory payload shape (no net-new derived fields) in `internal/api/client_test.go`
- [ ] T025 [P] [US3] Add command-level regression for non-26 behavior unchanged in `cmd/misthelper/main_extra_test.go`

### Implementation for User Story 3

- [ ] T026 [US3] Align org inventory record normalization to Python-first parity boundaries in `internal/api/client.go`
- [ ] T027 [US3] Preserve/verify non-26 stub behavior while keeping option-26 path isolated in `internal/menu/dispatcher.go`

### Validation for User Story 3

- [ ] T028 [US3] Run focused parity/regression tests for `internal/api/client_test.go`, `internal/menu/dispatcher_test.go`, and `cmd/misthelper/main_extra_test.go`

**Checkpoint**: Scope lock and parity boundaries are enforced with regression safety.

---

## Phase 6: Polish & Cross-Cutting Validation

**Purpose**: Final integrated verification and quality gates for merge readiness.

- [ ] T029 Run full unit/integration suite with race+coverage across `cmd/misthelper/main.go`, `internal/api/client.go`, `internal/menu/dispatcher.go`, and `internal/output/strategies.go` via `go test ./... -race -cover`
- [ ] T030 Run static analysis/lint/build quality gates against `go.mod`, `cmd/misthelper/main.go`, and `internal/**` via `go vet ./...`, `golangci-lint run ./...`, and `go build ./...`
- [ ] T031 Validate quickstart done criteria checklist from `specs/003-org-inventory-port/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundational)**: Depends on Phase 1.
- **Phase 3 (US1)**: Depends on Phase 2.
- **Phase 4 (US2)**: Depends on Phase 3 (shared handler from US1 required).
- **Phase 5 (US3)**: Depends on Phase 3 (can run in parallel with late US2 work if handler is stable).
- **Phase 6 (Polish)**: Depends on Phases 3, 4, and 5.

### User Story Dependency Graph

- **US1 (P1)** -> **US2 (P2)**
- **US1 (P1)** -> **US3 (P3)**
- **US2 (P2)** and **US3 (P3)** -> **Final Polish**

### Within-Story Ordering Rule

- Tests first -> implementation -> validation.
- No implementation task starts until its corresponding tests exist and fail for the intended delta.

## Parallel Execution Examples

### US1 Parallel Set

- Run `T008`, `T009`, `T010`, and `T011` in parallel (different test files/concerns), then execute `T012` -> `T013` -> `T014` -> `T015` -> `T016`.

### US2 Parallel Set

- Run `T017`, `T018`, and `T019` in parallel, then execute `T020` -> `T021` -> `T022`.

### US3 Parallel Set

- Run `T023`, `T024`, and `T025` in parallel, then execute `T026` -> `T027` -> `T028`.

## Implementation Strategy (MVP First)

1. Deliver **MVP = US1** (direct `--menu 26` with writer routing and deterministic result).
2. Add **US2** for interactive parity.
3. Add **US3** for strict scope/parity enforcement and non-26 regression protection.
4. Run final quality gates and quickstart validation before merge.
