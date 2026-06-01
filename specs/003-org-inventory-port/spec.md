# Feature Specification: Org Inventory Port

**Feature Branch**: `[003-org-inventory-port]`  
**Created**: 2026-05-27  
**Status**: Draft  
**Input**: User description: "Repository: MistHelper-Go. Create a new SpecKit feature specification for implementing exactly one new menu feature: Get Org Inventory ported from Python MistHelper behavior."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run Org Inventory via direct menu dispatch (Priority: P1)

A NOC engineer runs `--menu 26` and receives a complete organization inventory export that matches Python MistHelper behavior for the same organization and permissions.

**Why this priority**: Direct dispatch is the fastest automation path and is required for parity with existing scripted workflows.

**Independent Test**: Can be fully tested by invoking menu option 26 through direct dispatch and verifying that inventory data is exported through the configured output backend using endpoint strategy `getOrgInventory`.

**Acceptance Scenarios**:

1. **Given** valid organization context and credentials, **When** the user runs `--menu 26`, **Then** the system executes only operation 26 and exports organization inventory data through `output.Writer`.
2. **Given** operation 26 is executed, **When** export completes, **Then** the endpoint strategy name used for persistence/export routing is `getOrgInventory`.
3. **Given** operation 26 is executed, **When** the operation finishes, **Then** menu operations other than 26 remain unchanged from current stub/behavior.

---

### User Story 2 - Run Org Inventory from interactive menu selection (Priority: P2)

A NOC engineer using interactive mode selects menu option 26 ("List Org Inventory") and receives the same inventory output and behavior as direct dispatch.

**Why this priority**: Interactive mode is the primary operator experience for junior NOC engineers and must remain consistent with direct dispatch.

**Independent Test**: Can be tested by launching interactive mode, selecting option 26, and confirming execution path, export behavior, and user-facing completion flow match direct dispatch.

**Acceptance Scenarios**:

1. **Given** interactive mode is active, **When** the user selects option 26, **Then** operation 26 dispatches correctly and exports via `output.Writer` with endpoint strategy `getOrgInventory`.
2. **Given** interactive mode and direct mode are run against the same org, **When** both complete successfully, **Then** they produce equivalent inventory content shape and operation outcome.

---

### User Story 3 - Maintain Python-first parity boundaries (Priority: P3)

A maintainer ports only the existing Python MistHelper "Get Org Inventory" behavior and does not introduce additional fields, workflows, or menu features.

**Why this priority**: Prevents scope drift and protects Python-first governance for MistHelper-Go.

**Independent Test**: Can be tested by comparing expected Python behavior and confirming the Go implementation scope is limited to operation 26 with no net-new feature additions.

**Acceptance Scenarios**:

1. **Given** the implementation scope for this feature, **When** code review is performed, **Then** only operation 26 behavior is newly implemented and no additional menu features are introduced.
2. **Given** existing operation stubs, **When** this feature is merged, **Then** stubs not related to option 26 remain intact.

---

### Edge Cases

- Option 26 is selected but API returns an empty inventory set.
- Option 26 is selected but API call fails (network/auth/rate-limit), requiring user-visible failure handling and no partial silent success.
- Interactive menu receives invalid input before valid selection 26, then proceeds correctly once 26 is selected.
- `--menu 26` is invoked in non-interactive automation where stdout/stderr are monitored; operation must return a deterministic success/failure outcome.

## Requirements *(mandatory)*

### Scope

- Implement exactly one menu feature in MistHelper-Go: option 26, labeled "List Org Inventory".
- Port behavior from existing Python MistHelper inventory behavior; do not invent additional capability.
- Support both execution paths:
  - Direct dispatch via `--menu 26`
  - Interactive selection of menu option 26
- Route all exports/persistence through `output.Writer` using endpoint strategy name `getOrgInventory`.

### Non-Goals

- No implementation of any other menu options beyond operation 26.
- No redesign of menu framework, API client framework, or output framework.
- No changes to destructive menu operations or unrelated operation numbering.
- No new data model or feature that does not already exist in Python behavior.

### Functional Requirements

- **FR-001**: System MUST expose menu option 26 as "List Org Inventory" in the interactive menu display.
- **FR-002**: System MUST dispatch option 26 correctly when invoked as `--menu 26`.
- **FR-003**: System MUST dispatch option 26 correctly when selected through interactive input.
- **FR-004**: System MUST retrieve organization inventory data using existing API-layer conventions and Python-parity behavior.
- **FR-005**: System MUST send operation 26 export output through `output.Writer` rather than direct file/database writes.
- **FR-006**: System MUST use endpoint strategy key `getOrgInventory` for output/export strategy lookup.
- **FR-007**: System MUST preserve all current stub behavior for menu operations other than 26.
- **FR-008**: System MUST surface operation success/failure in a deterministic way for CLI automation and interactive operators.
- **FR-009**: System MUST avoid introducing net-new fields or business logic not present in Python behavior for this operation.
- **FR-010**: System MUST keep operation numbering stable so existing references to option 26 remain valid.

### Acceptance Criteria

- **AC-001**: `--menu 26` executes operation 26 end-to-end and exits with a success status when API/export succeed.
- **AC-002**: Interactive selection of option 26 executes the same operational path and output routing as direct dispatch.
- **AC-003**: Export routing for option 26 uses `output.Writer` with endpoint strategy `getOrgInventory`.
- **AC-004**: Existing menu stubs and behavior outside operation 26 are unchanged.
- **AC-005**: Operation behavior aligns with Python MistHelper parity for "Get Org Inventory" scope.

### Test Strategy

- Unit tests in `internal/menu` validate dispatch wiring for both direct and interactive paths to option 26.
- Unit tests in `internal/api` validate inventory retrieval path and expected success/error handling boundaries.
- Unit tests in `internal/output` validate endpoint strategy registration/lookup for `getOrgInventory` and writer integration usage.
- Integration-style command tests in `cmd/misthelper` validate `--menu 26` execution wiring and unchanged behavior for adjacent operations.
- Regression checks confirm no changes to behavior for non-26 menu stubs.

### Affected Files & Architecture Mapping

| Area | File(s) | Responsibility for this feature |
| - | - | - |
| CLI entry and direct dispatch | `cmd/misthelper/main.go` | Ensure `--menu 26` reaches menu dispatcher and executes option 26 path without affecting other options |
| API layer | `internal/api/client.go` (and related tests) | Provide org inventory retrieval behavior consistent with Python parity |
| Menu layer | `internal/menu/dispatcher.go`, `internal/menu/display.go`, `internal/menu/entry.go` (and related tests) | Expose label "List Org Inventory", map option 26, and route both interactive/direct flows |
| Output layer | `internal/output/*` strategy/writer components | Ensure operation 26 writes via `output.Writer` and strategy key `getOrgInventory` |

### Key Entities *(include if feature involves data)*

- **Org Inventory Record**: A normalized inventory item representing organization-level devices/assets and associated attributes exposed by existing Python behavior.
- **Menu Operation 26 Dispatch Context**: Execution context indicating option 26 selection source (direct CLI or interactive menu) and org scope.
- **Export Job Result**: Outcome of writer-routed export, including success/failure status and backend-targeted write summary.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of executions of `--menu 26` in test scenarios dispatch to operation 26 and not to any other operation.
- **SC-002**: 100% of interactive selection tests for option 26 dispatch to the same operation logic as direct dispatch.
- **SC-003**: 100% of operation-26 export tests confirm routing through `output.Writer` with strategy key `getOrgInventory`.
- **SC-004**: 0 unintended behavior changes are observed in regression tests for menu operations outside 26.
- **SC-005**: Python parity review for operation scope confirms no net-new feature behavior introduced.

## Assumptions

- Python MistHelper already defines the authoritative "Get Org Inventory" behavior and that behavior is available for parity reference.
- Existing MistHelper-Go architecture (`internal/api`, `internal/menu`, `internal/output`, `cmd/misthelper/main.go`) remains the correct extension path.
- Existing output backends are already integrated behind `output.Writer`; this feature only needs to use the established writer path.
- Authentication/org context setup already exists and is reused by operation 26.
- Existing stubs for non-26 operations remain intentionally incomplete and must stay unchanged in this feature.
