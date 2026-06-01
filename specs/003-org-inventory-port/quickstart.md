# Quickstart: Implementing Menu Option 26 (Org Inventory Port)

## Goal

Implement exactly one feature: menu option `26` (`List Org Inventory`) with Python-first parity and writer-based export routing.

## Prerequisites

- Repo checked out on feature branch context for `specs/003-org-inventory-port`
- Valid `.env` with Mist credentials/org context for local runs
- Go toolchain and lint stack installed

## Implementation Sequence

1. **API layer first**
   - Add org inventory retrieval method in `internal/api/client.go`.
   - Match Python semantics (`getOrgInventory`, pagination, `vc=true`).
   - Add/extend API tests for success, empty, and error paths.

2. **Output strategy alignment**
   - Add strategy entry for endpoint key `getOrgInventory` in `internal/output/strategies.go`.
   - Add tests verifying lookup and expected PK strategy behavior.

3. **Menu wiring**
   - Replace option 26 stub path with real handler wiring in menu flow.
   - Ensure both direct dispatch and interactive selection call the same operation logic.
   - Keep all non-26 options unchanged.

4. **CLI verification wiring**
   - Confirm `--menu 26` reaches dispatcher and returns deterministic success/failure.
   - Add/extend `cmd/misthelper` tests as needed.

5. **Regression hardening**
   - Add tests to ensure adjacent and unrelated stubs remain unchanged.

## Validation Checklist

- [ ] Direct dispatch path works: `--menu 26`
- [ ] Interactive selection path works for `26`
- [ ] Output path uses `output.Writer`
- [ ] Endpoint key used is `getOrgInventory`
- [ ] Non-26 menu behavior unchanged
- [ ] Python parity constraints maintained

## Quality Gates

Run and pass all:

- `go build ./...`
- `go vet ./...`
- `golangci-lint run ./...`
- `go test ./... -race -cover`

## Done Criteria

Feature is complete when AC-001 through AC-005 are satisfied and all quality gates pass without widening scope beyond option 26.
