# Implementation Plan: Org Inventory Port

**Branch**: `main` | **Date**: 2026-05-27 | **Spec**: `specs/003-org-inventory-port/spec.md`
**Input**: Feature specification from `specs/003-org-inventory-port/spec.md`

## Summary

Port exactly one feature into MistHelper-Go: **menu option 26 (`List Org Inventory`)** with Python-first parity to MistHelper’s `OrgInventoryExporter.inventory()` behavior (`getOrgInventory`, `vc=True`, paginated). The implementation will wire option 26 through both direct (`--menu 26`) and interactive flows, route output only via `output.Writer`, and use endpoint strategy key `getOrgInventory`.

No unrelated menu options will be implemented or altered.

## Technical Context

**Language/Version**: Go 1.21+ (repo currently Go 1.25 toolchain compatible)
**Primary Dependencies**: `mistapi-go` v0.4.73+, `godotenv` v1.5.1
**Storage**: Existing output backends behind `output.Writer` (CSV/SQLite, plus existing polyglot flow in repo)
**Testing**: `go test ./... -race -cover` plus focused package tests
**Target Platform**: Linux container runtime (Podman primary), Windows local development
**Project Type**: CLI + menu dispatcher + API wrapper + output backends
**Performance Goals**: Option 26 completes with parity pagination behavior and deterministic success/failure path for automation
**Constraints**:

- Scope locked to option 26 only
- Python-first parity: no net-new fields/logic beyond Python behavior
- Preserve all non-26 stubs/behavior
- Use existing architecture (no framework redesign)

**Scale/Scope**: Multi-file but bounded changes in `internal/api`, `internal/menu`, `internal/output`, `cmd/misthelper`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
| - | - | - |
| Five-Item Rule | PASS | Add small focused methods/files only; avoid large functions. |
| Package-Based Architecture (No wrappers) | PASS | Add concrete operation logic inside existing packages/types. |
| Safety-First Input Handling | PASS | Keep existing `SafeInput` + dispatcher flow; no bypass. |
| Deployment Pipeline Awareness | PASS | Plan includes full quality gates and container validation after implementation. |
| Observability & Logging | PASS | Add info-before/debug-after logs around option-26 API and writer actions. |
| Inline Comments + Action Logging | PASS | Required in implementation scope; tracked in validation checklist. |
| Python-First Constraint | PASS | Mirrors Python `getOrgInventory(vc=True, limit=1000)` behavior only. |

No constitution exceptions are required.

## Project Structure

### Documentation (this feature)

```text
specs/003-org-inventory-port/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── menu-option-26-org-inventory.md
└── tasks.md (created later by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/misthelper/
└── main.go

internal/
├── api/
│   ├── client.go
│   └── client_*test.go
├── menu/
│   ├── dispatcher.go
│   ├── display.go
│   ├── entry.go
│   └── *_test.go
└── output/
    ├── strategies.go
    └── *_test.go
```

**Structure Decision**:

- Keep operation execution in menu layer, data retrieval in API layer, persistence routing in output layer.
- Add or extend targeted tests near touched package boundaries.
- Do not add new top-level packages.

## Architecture & File-Level Change Plan

| File | Change Type | Planned Change |
| - | - | - |
| `internal/api/client.go` | Extend | Add org inventory retrieval method that calls Mist SDK inventory endpoint with parity options (`vc=true`, paginated limit). Return normalized `[]map[string]any` and wrapped errors. |
| `internal/api/client_test.go` + optional `client_extra_test.go` | Extend | Add tests for success path, empty inventory, and API error propagation for new inventory method. |
| `internal/menu/entry.go` | Optional minimal | If needed, add typed dependency wiring for operation handlers without changing non-26 behavior. |
| `internal/menu/dispatcher.go` | Extend | Wire option 26 handler path for direct and interactive dispatch while keeping existing unknown/stub behavior. |
| `internal/menu/display.go` | Verify/adjust only if needed | Ensure label remains exactly `List Org Inventory` for option 26 in rendered menu output. |
| `internal/menu/dispatcher_test.go` + `display_test.go`/`run_test.go` | Extend | Add assertions that option 26 dispatches correctly in both direct and interactive flows and does not affect adjacent stubs. |
| `internal/output/strategies.go` | Extend | Register strategy key `getOrgInventory` with natural PK semantics aligned to Python strategy expectations. |
| `internal/output/strategies_test.go` + `writer_test.go` | Extend | Verify strategy lookup and writer routing for endpoint `getOrgInventory`. |
| `cmd/misthelper/main.go` + `main_test.go`/`main_extra_test.go` | Minimal verify | Ensure `--menu 26` reaches dispatcher path unchanged; add regression guard for non-26 behavior. |
| `README.md` / `CHANGELOG.md` | Deferred to implementation | Update only if code implementation changes operator-visible behavior or operation status. |

## Dependency Update Plan

| Dependency | Action | Reason |
| - | - | - |
| `mistapi-go` | None expected | Existing SDK already supports inventory call pattern needed for parity. |
| `godotenv` | None | No configuration model change required. |
| Other modules | None expected | Feature is a bounded port within current architecture. |

If implementation reveals an SDK method gap, re-evaluate with Python-parity-first constraint before adding dependencies.

## Risk Controls

| Risk | Impact | Control |
| - | - | - |
| Wrong parity flags (e.g., missing `vc=true`) | Inventory count/content drift vs Python | Add explicit parity tests and code comments documenting `vc=true` requirement. |
| Endpoint strategy missing/incorrect (`getOrgInventory`) | Incorrect dedup/upsert behavior in SQLite | Add strategy registration + tests in `internal/output`. |
| Option-26 wiring alters stubs | Regression across unfinished operations | Add regression tests proving non-26 operations remain unchanged. |
| Non-deterministic failure behavior in automation | CI/script instability | Enforce clear error returns and exit behavior for `--menu 26` path. |
| Over-scoping beyond menu 26 | Governance violation | Scope gate in reviews + test naming tied to option 26 only. |

## Validation Approach

### Unit & Integration Validation

- Menu dispatch tests:
  - direct `Dispatch(..., 26)`
  - interactive input selecting `26`
  - non-26 regression checks remain unchanged
- API tests:
  - inventory retrieval success
  - empty result handling
  - API failure propagation
- Output tests:
  - `getOrgInventory` strategy lookup and PK type
  - writer invoked with endpoint key `getOrgInventory`
- Command wiring tests:
  - `--menu 26` route verification in `cmd/misthelper`

### Quality Gates

- `go build ./...`
- `go vet ./...`
- `golangci-lint run ./...`
- `go test ./... -race -cover`

### Parity Validation (Python-first)

- Compare Go option-26 behavior with Python reference for:
  - endpoint intent (`getOrgInventory`)
  - pagination shape
  - inclusion semantics (`vc=true` physical members)
  - output routing through exporter/writer abstraction

## Implementation Phases

### Phase 0 - Research & Parity Baseline

- Confirm Python source of truth for org inventory behavior and options.
- Confirm Go repo currently has option 26 label but stub-only execution.
- Confirm required strategy key is absent/present and plan update accordingly.

### Phase 1 - Design & Contracts

- Finalize operation contract (inputs, outputs, error semantics) for menu 26.
- Finalize data model for org inventory records and export result surface.
- Confirm no dependency additions needed.

### Phase 2 - Implementation Planning (Execution-ready)

- Sequence code changes by package: `api` -> `output` -> `menu` -> `cmd` tests.
- Define test-first checkpoints for each package.
- Define explicit regression checks for non-26 behavior lock.

### Phase 3 - Validate & Harden

- Run quality gates and targeted parity validations.
- Fix any drift with Python behavior.
- Prepare task breakdown (`/speckit.tasks`) from this plan.

## Post-Design Constitution Re-Check

All gates remain **PASS** after design artifacts:

- Scope is constrained to option 26.
- Architecture remains package-based and testable.
- Python-first parity is explicit and enforceable.
- Risk controls and validation gates are defined and measurable.

## Complexity Tracking

No constitution violations or exceptions planned.
