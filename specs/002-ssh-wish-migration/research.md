# Phase 0 Research - SSH Wish Migration

## Decision 1: Use Wish as the sole SSH server runtime

**Decision**: Replace raw `golang.org/x/crypto/ssh` server/channel loop in `internal/ssh/server.go` with a Wish server and middleware pipeline.

**Rationale**:

- Wish provides mature SSH session lifecycle handling and PTY integration out of the box.
- The current issues (no-echo/no-dispatch) are transport/session problems, not menu business logic problems.
- A single runtime path reduces transport divergence and maintenance cost.

**Alternatives considered**:

- Patch raw `x/crypto/ssh` handling incrementally.
- Rejected because PTY/input edge behavior remains hand-rolled and brittle.
- Keep dual runtimes behind feature flag.
- Rejected because it doubles test matrix and operational complexity.

## Decision 2: Preserve current auth/config semantics from `api.Config`

**Decision**: Keep username/password/port semantics exactly aligned with `api.Config` (`SSHUser`, `SSHPassword`, `SSHPort`) while adapting to Wish auth hooks.

**Rationale**:

- Zero user-facing credential migration required.
- Maintains parity with existing startup/runtime behavior.
- Avoids introducing authentication policy drift.

**Alternatives considered**:

- Add new auth providers or key-based login now.
- Rejected as out-of-scope and non-goal.

## Decision 3: Keep menu dispatcher as the authoritative interactive flow

**Decision**: Continue to wire Wish session I/O into `menu.NewDispatcher(...)` without changing dispatcher behavior.

**Rationale**:

- Preserves menu integration and output writer behavior.
- Limits migration blast radius to SSH transport.
- Keeps unknown-option and handler-dispatch logic unchanged.

**Alternatives considered**:

- Introduce a new Wish-native menu shell.
- Rejected because it would duplicate existing menu logic and violate scope.

## Decision 4: Preserve per-session workspace lifecycle

**Decision**: Retain current session ID generation and per-session directory creation pattern (`data/sessions/session_<id>`).

**Rationale**:

- Existing operational tooling expects this structure.
- Maintains session isolation and traceability semantics.

**Alternatives considered**:

- Move to in-memory-only sessions.
- Rejected due to loss of parity and trace artifacts.

## Decision 5: PTY and line-input compatibility strategy

**Decision**: Implement explicit compatibility handling for line endings, echo, and Enter semantics at the Wish session bridge layer.

**Rationale**:

- Required to eliminate current no-echo/no-dispatch quirks.
- Ensures one submitted line maps to one dispatcher dispatch.

**Alternatives considered**:

- Depend on defaults only.
- Rejected because observed issues indicate default behavior is not sufficient for this app contract.

## Decision 6: Validation sequencing constraint

**Decision**: Do not compile/run during implementation slices; execute all compile/test/lint checks only in the final validation stage.

**Rationale**:

- Explicit user constraint in this feature request.
- Encourages planned slice-by-slice code changes before test execution.

**Alternatives considered**:

- Continuous compile/test while implementing.
- Rejected for this feature due to explicit constraint.

## Decision 7: Final validation environment

**Decision**: Perform final acceptance in a local container workflow (build/run/SSH acceptance) rather than waiting for GitHub container workflow completion.

**Rationale**:

- Explicit user requirement.
- Faster feedback on interactive SSH behavior and PTY handling.

**Alternatives considered**:

- CI-only validation.
- Rejected because it delays interactive SSH confirmation and violates the specified flow.
