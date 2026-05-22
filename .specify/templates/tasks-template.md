---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The examples below include test tasks. Tests are OPTIONAL - only include them if explicitly requested in the feature specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **MistHelper-Go**: `internal/<pkg>/`, `cmd/misthelper/`, `tests/<pkg>/`
- All new packages go under `internal/`
- Test files alongside package files or in `tests/` with `_test.go` suffix

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Package initialization and basic structure

- [ ] T001 Create package structure per implementation plan
- [ ] T002 Add any new dependencies to go.mod via `go get`
- [ ] T003 [P] Configure package-level interfaces and types

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T004 Define primary key strategy in endpoint strategies map
- [ ] T005 [P] Define interfaces for new packages
- [ ] T006 [P] Setup struct types and constructor functions
- [ ] T007 Create base types that all stories depend on
- [ ] T008 Configure error handling and slog logging infrastructure
- [ ] T009 Setup environment configuration loading

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 1 (OPTIONAL - only if tests requested)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T010 [P] [US1] Unit test for [function] in internal/[pkg]/[name]_test.go
- [ ] T011 [P] [US1] Integration test for [user journey] in tests/[pkg]/[name]_test.go

### Implementation for User Story 1

- [ ] T012 [P] [US1] Define [Type1] struct in internal/[pkg]/[file].go
- [ ] T013 [P] [US1] Define [Type2] struct in internal/[pkg]/[file].go
- [ ] T014 [US1] Implement [method] on [Type] in internal/[pkg]/[file].go (depends on T012, T013)
- [ ] T015 [US1] Implement [function/feature] in internal/[pkg]/[file].go
- [ ] T016 [US1] Add input validation and error wrapping
- [ ] T017 [US1] Add slog action logging before/after every operation

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2 (OPTIONAL - only if tests requested)

- [ ] T018 [P] [US2] Unit test for [function] in internal/[pkg]/[name]_test.go
- [ ] T019 [P] [US2] Integration test for [user journey] in tests/[pkg]/[name]_test.go

### Implementation for User Story 2

- [ ] T020 [P] [US2] Define [Type] struct in internal/[pkg]/[file].go
- [ ] T021 [US2] Implement [method] in internal/[pkg]/[file].go
- [ ] T022 [US2] Implement [feature] in internal/[pkg]/[file].go
- [ ] T023 [US2] Integrate with User Story 1 components (if needed)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 3 (OPTIONAL - only if tests requested)

- [ ] T024 [P] [US3] Unit test for [function] in internal/[pkg]/[name]_test.go
- [ ] T025 [P] [US3] Integration test for [user journey] in tests/[pkg]/[name]_test.go

### Implementation for User Story 3

- [ ] T026 [P] [US3] Define [Type] struct in internal/[pkg]/[file].go
- [ ] T027 [US3] Implement [method] in internal/[pkg]/[file].go
- [ ] T028 [US3] Implement [feature] in internal/[pkg]/[file].go

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in README.md and CHANGELOG.md
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX Run `go vet ./...` and `golangci-lint run ./...` - zero violations
- [ ] TXXX [P] Additional unit tests (if requested) in tests/
- [ ] TXXX Security hardening (`gosec`, `govulncheck`)
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (different packages)
  - Or sequentially in priority order (P1 -> P2 -> P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but independently testable

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Types/interfaces before implementations
- Implementations before integrations
- Core logic before menu/output wiring
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (different packages)
- All tests for a user story marked [P] can run in parallel
- Types within a story marked [P] can run in parallel

---

## Quality Gates (run before each commit)

```powershell
go vet ./...                    # Static analysis
go build ./...                  # Compile check
golangci-lint run ./...         # Lint check
go test ./... -race -cover      # Tests with race detector
```

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Always add inline comments on every line (NON-NEGOTIABLE per constitution)
- Always add slog action logging before/after every operation (NON-NEGOTIABLE per constitution)
