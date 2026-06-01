# Contract: Menu Option 26 - Org Inventory

## Purpose

Define the implementation contract for MistHelper-Go menu option `26` (`List Org Inventory`) under Python-first parity constraints.

## Inputs

| Input | Source | Required | Contract |
| - | - | - | - |
| `operation_number` | CLI flag `--menu` or interactive selection | Yes | Must equal `26` to enter this feature path |
| `org_id` | Runtime config | Yes | Must be present and valid for API calls |
| `context` | Dispatcher context | Yes | Must propagate cancellation/timeouts |

## Processing Contract

1. Dispatcher resolves option `26` to Org Inventory handler.
2. Handler calls API layer org inventory retrieval with parity settings.
3. API layer retrieves org inventory data using Mist SDK endpoint intent equivalent to Python behavior.
4. Handler routes results through `output.Writer.Write(...)` using endpoint key `getOrgInventory`.
5. Handler returns deterministic success or wrapped error.

## Output Contract

| Output | Contract |
| - | - |
| Export routing | Must call `output.Writer` only; no direct file/DB writes in handler |
| Endpoint strategy key | Must be exactly `getOrgInventory` |
| Record payload | API-parity inventory records (flatten/normalize via existing pipeline) |
| Status semantics | Success when API + writer succeed; error otherwise |

## Error Contract

| Scenario | Expected Behavior |
| - | - |
| API failure | Return wrapped error; no false-success message |
| Empty inventory result | Valid success path with zero records |
| Writer failure | Return wrapped error and fail operation |
| Invalid option in dispatcher | Existing unknown-option behavior unchanged |

## Non-Functional Contract

- Scope restricted to option 26 only.
- No change to numbering or behavior of other menu options.
- Maintain structured logging around action boundaries.
- Preserve compatibility with direct and interactive invocation paths.

## Test Contract Mapping

| Requirement | Verification |
| - | - |
| Direct dispatch | `Dispatch(..., 26)` / `--menu 26` command-path tests |
| Interactive dispatch | Input-driven dispatcher tests selecting `26` |
| Writer routing | Assert `Write(..., "getOrgInventory", ...)` usage |
| Scope lock | Regression tests for non-26 options |
| Python parity boundaries | Review + tests confirming no net-new feature behavior |
