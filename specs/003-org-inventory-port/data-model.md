# Data Model: Org Inventory Port (Menu 26)

## Entity: MenuOperationDispatchContext

| Field | Type | Required | Description | Validation |
| - | - | - | - | - |
| `operation_number` | int | Yes | Selected menu operation number | Must equal `26` for this feature path |
| `dispatch_mode` | enum(`direct`,`interactive`) | Yes | Source of invocation (`--menu` or typed selection) | Must be one of allowed enum values |
| `org_id` | string | Yes | Organization identifier from config | Non-empty; UUID-shaped string expected |
| `request_time` | timestamp | Yes | Invocation timestamp for logs/traceability | Generated at dispatch time |

### Dispatch state transitions

`received` -> `validated` -> `executing` -> (`succeeded` | `failed`)

- `received` to `validated`: operation number and context checks pass.
- `validated` to `executing`: API call begins.
- `executing` to `succeeded`: inventory fetched + writer completed.
- `executing` to `failed`: API or writer error surfaced deterministically.

## Entity: OrgInventoryRecord

| Field | Type | Required | Description | Validation |
| - | - | - | - | - |
| `id` | string | Yes (for natural PK strategy) | Inventory row identifier from Mist API | Non-empty for upsert-safe persistence |
| `org_id` | string | Usually | Owning organization | Should match configured org where present |
| `site_id` | string | Optional | Site association for device | Empty allowed for unassigned records |
| `mac` | string | Optional | Device MAC | Preserve API value as-is |
| `serial` | string | Optional | Device serial | Preserve API value as-is |
| `model` | string | Optional | Device model | Preserve API value as-is |
| `type` | string | Optional | Device type | Preserve API value as-is |
| `...` | mixed | Optional | Additional API parity fields | No net-new derived business fields in this feature |

### Notes

- Record shape remains API-driven to preserve Python parity.
- Flattening/normalization follows existing output pipeline conventions.

## Entity: ExportJobResult

| Field | Type | Required | Description | Validation |
| - | - | - | - | - |
| `endpoint_key` | string | Yes | Output strategy lookup key | Must be exactly `getOrgInventory` |
| `record_count` | int | Yes | Number of records sent to writer | Must be `>= 0` |
| `status` | enum(`success`,`error`) | Yes | Final export status | Deterministic outcome for automation |
| `error_message` | string | Conditional | Failure context | Present when status is `error` |

### Export state transitions

`initialized` -> `writing` -> (`success` | `error`)

- `initialized` to `writing`: records prepared and writer invoked.
- `writing` to `success`: writer returns nil.
- `writing` to `error`: writer returns error; error propagated to caller.

## Relationships

- `MenuOperationDispatchContext (1)` -> `(N) OrgInventoryRecord`
- `MenuOperationDispatchContext (1)` -> `(1) ExportJobResult`

## Invariants

1. Menu scope invariant: this feature handles operation `26` only.
2. Routing invariant: all persistence/export goes through `output.Writer`.
3. Strategy invariant: endpoint key used for this feature is `getOrgInventory`.
4. Parity invariant: no net-new business fields or workflows beyond Python behavior.
