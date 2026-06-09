# Research: Org Inventory Port (Menu 26)

## Decision 1: Python parity behavior for inventory retrieval

- **Decision**: Mirror Python org inventory retrieval semantics: call org inventory endpoint with `vc=true`, paginated limit, and export-oriented flattening.
- **Rationale**: Python reference (`OrgInventoryExporter.inventory()` and `all_inventory_with_limit`) is the governance source of truth for Go ports.
- **Alternatives considered**:
  - `vc=false`/default: rejected because it can collapse VC representation and diverge from Python export intent.
  - `listOrgDevices` instead of inventory endpoint: rejected because spec mandates `getOrgInventory` parity scope.

## Decision 2: Keep operation scope locked to menu option 26 only

- **Decision**: Implement only option 26 execution path and preserve all other operation stubs/behavior.
- **Rationale**: Feature spec explicitly forbids unrelated option implementation and requires stable numbering.
- **Alternatives considered**:
  - Bundle neighboring inventory/license options: rejected as scope expansion.
  - Refactor full dispatcher framework: rejected as non-goal and high-risk regression surface.

## Decision 3: Output routing contract uses `output.Writer` with endpoint key `getOrgInventory`

- **Decision**: Option 26 handler writes via existing `output.Writer` API and registers/uses strategy key `getOrgInventory`.
- **Rationale**: Required by FR-005/FR-006 and repo architecture; avoids direct file/DB writes.
- **Alternatives considered**:
  - Direct CSV/SQLite writes from menu handler: rejected (architecture violation).
  - Reusing a different existing endpoint key: rejected (breaks parity and tests).

## Decision 4: Dependency posture

- **Decision**: No new dependency additions expected.
- **Rationale**: Existing stack (`mistapi-go`, current menu/output packages) already supports required behavior.
- **Alternatives considered**:
  - Add extra SDK/helper package for pagination abstraction: rejected unless implementation proves unavoidable.

## Decision 5: Validation strategy centers on parity + regression boundaries

- **Decision**: Validate with package-level tests for API/menu/output plus direct `--menu 26` wiring tests and non-26 regression guards.
- **Rationale**: This provides measurable proof for AC-001 through AC-005 while containing risk.
- **Alternatives considered**:
  - Manual-only validation: rejected (insufficient for hidden/regression checks).
  - Full end-to-end external API integration in CI only: deferred; unit/integration-style local tests remain primary gate.

## Resolved Clarifications

All technical context items for planning are resolved:

- Python reference behavior identified.
- Go target architecture boundaries identified.
- Endpoint strategy key requirement identified.
- Scope boundary and non-goals confirmed.
- Validation and risk controls defined.
