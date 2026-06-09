# Feature Specification: SSH Wish Migration

**Feature Branch**: `[002-ssh-wish-migration]`  
**Created**: 2026-05-24  
**Status**: Draft  
**Input**: User description: "Migrate MistHelper-Go embedded SSH server from low-level handlers to Wish while preserving current app behavior, dispatcher wiring, authentication semantics, per-session isolation/logging semantics, and improving PTY/input reliability."

## Problem Statement

The current SSH server implementation exhibits interactive PTY/input quirks that reduce operator confidence and can interrupt normal menu-driven workflows. The migration must improve reliability of interactive behavior while preserving current operational semantics and user-facing behavior.

## Goals

- Replace the SSH server internals in `internal/ssh/server.go` with a Wish-based server/session lifecycle.
- Preserve menu dispatcher integration with the existing `internal/menu` flow.
- Preserve output backend wiring and behavior.
- Preserve ForceCommand-like user experience where authenticated users land directly in the interactive menu session.
- Preserve credentials/config semantics (SSH username/password and port values from `api.Config`).
- Preserve per-session directory creation behavior and logging semantics as closely as practical.
- Improve PTY/input handling reliability (echo and Enter handling) to remove current interaction quirks.

## Non-Goals

- Adding new menu operations.
- Changing business logic in existing menu handlers.
- Reworking web server behavior or web workflows.
- Introducing new authentication providers or credential stores.
- Changing output backend formats or destination behavior.

## Constraints

- Follow SpecKit workflow before implementation.
- Defer compile/build/test execution until implementation is complete and in validation stage.
- Final validation must run in a local container workflow; do not wait for GitHub container build completion.
- Maintain parity with current SSH credentials and configuration source behavior.
- Maintain compatibility with current operational run modes used by the repository.

## Architecture Approach

- Introduce a Wish-based SSH server bootstrap in `internal/ssh/server.go` while retaining current external server responsibilities and startup surface.
- Keep the menu-session handoff contract intact so authenticated sessions route directly into the existing interactive dispatcher path.
- Preserve current configuration read path for SSH username/password/port from `api.Config`.
- Preserve host key and session workspace lifecycle responsibilities with equivalent outcomes (startup, per-session creation, teardown).
- Apply middleware/session handling patterns that improve terminal interaction consistency (PTY negotiation, input buffering, newline/echo behavior) without changing menu business logic.

## Risks and Mitigations

- **Risk**: Interactive behavior changes under different SSH clients.  
  **Mitigation**: Add cross-client acceptance checks and PTY-focused regression scenarios.
- **Risk**: Subtle regressions in ForceCommand-like flow (users not landing in menu).  
  **Mitigation**: Add explicit login-to-menu parity checks in migration acceptance tests.
- **Risk**: Session lifecycle or cleanup drift causes stale directories/resources.  
  **Mitigation**: Add per-session directory and teardown verification checks.
- **Risk**: Logging semantics drift reduces observability for operators.  
  **Mitigation**: Verify key lifecycle events and message intent parity with current behavior.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Login to Menu Without Friction (Priority: P1)

As a NOC engineer, I can SSH into MistHelper-Go and immediately enter the interactive menu session, so operational tasks start quickly with no manual shell pivoting.

**Why this priority**: This is the core SSH workflow and must remain intact during migration.

**Independent Test**: Authenticate with current credentials and verify direct landing in menu prompt without additional commands.

**Acceptance Scenarios**:

1. **Given** the SSH server is running with configured credentials, **When** a user logs in successfully, **Then** the session enters the interactive menu directly.
2. **Given** a successful login, **When** the initial screen is rendered, **Then** expected menu text is visible over SSH.

---

### User Story 2 - Reliable Interactive Input Dispatch (Priority: P1)

As a NOC engineer, I can type menu options and press Enter with predictable behavior, so I can execute handlers without retrying inputs.

**Why this priority**: Existing interaction quirks directly impact task completion reliability.

**Independent Test**: Send numeric option + Enter and verify exactly one dispatch to the expected handler.

**Acceptance Scenarios**:

1. **Given** an active menu session, **When** the user enters a valid numeric option and presses Enter, **Then** the corresponding handler is dispatched.
2. **Given** an active menu session, **When** the user enters an unknown option and presses Enter, **Then** the expected unknown-option feedback appears and the session remains active.

---

### User Story 3 - Clean Exit and Session Stability (Priority: P2)

As a NOC engineer, I can end sessions with standard terminal signals or disconnects without leaving the server in a bad state.

**Why this priority**: Clean shutdown behavior is required for operational stability and repeatable access.

**Independent Test**: Trigger Ctrl+C, Ctrl+D, and client disconnects; verify clean session termination and ongoing server availability.

**Acceptance Scenarios**:

1. **Given** an active menu session, **When** the user sends Ctrl+C or Ctrl+D, **Then** the session exits cleanly.
2. **Given** a terminated session, **When** a new client connects, **Then** login and menu startup still work.

---

### Edge Cases

- Client connects without expected PTY behavior.
- Terminal resize events occur repeatedly during active prompts.
- Disconnect happens mid-input or mid-render.
- Session directory creation fails for one session while server remains up.
- Invalid credentials are retried and then corrected in a subsequent attempt.

### Migration Acceptance Checks

1. SSH login works with current credentials.
2. Menu renders over SSH.
3. Typing numeric option + Enter dispatches handler.
4. Ctrl+C/Ctrl+D/session close exits cleanly.
5. Unknown option returns expected message and keeps session alive.

### Test Strategy

- Define parity-focused SSH acceptance tests for login, direct menu landing, and dispatch behavior.
- Define interactive reliability tests focused on echo, Enter handling, and repeated inputs.
- Define lifecycle tests for Ctrl+C, Ctrl+D, client disconnect, and server readiness for new sessions.
- Define compatibility checks for credentials/config usage, session directory behavior, and lifecycle logging intent.
- Execute validation only after implementation is complete, and run final validation in a local containerized workflow.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST migrate SSH server internals to a Wish-based session handling model while preserving externally observable SSH behavior for operators.
- **FR-002**: Authenticated SSH users MUST land directly in the interactive menu session without requiring extra commands.
- **FR-003**: The system MUST preserve existing credentials/config usage for SSH username, password, and port from current configuration sources.
- **FR-004**: The system MUST preserve menu dispatcher integration and output backend wiring behavior.
- **FR-005**: The system MUST correctly dispatch valid numeric menu options submitted with Enter.
- **FR-006**: The system MUST return expected unknown-option feedback and keep the session active after invalid numeric selection.
- **FR-007**: The system MUST support clean exit behavior for Ctrl+C, Ctrl+D, and client-initiated session close.
- **FR-008**: The system MUST preserve per-session directory creation semantics and maintain session isolation.
- **FR-009**: The system MUST preserve session lifecycle logging semantics as closely as practical, including session start and session end outcomes.
- **FR-010**: The migration MUST improve PTY/input reliability to eliminate current interactive echo and Enter handling quirks.

### Key Entities *(include if feature involves data)*

- **SSH Session Context**: Active connection state, terminal behavior state, and cleanup lifecycle.
- **SSH Runtime Configuration**: Username/password/port values used to enforce login and listener behavior.
- **Menu Session Binding**: Contract that ties authenticated SSH sessions to interactive menu dispatch.
- **Session Workspace**: Per-session directory and associated runtime artifacts.

## Compatibility Notes

- Existing SSH credentials remain valid with no required user-side credential changes.
- Existing SSH port behavior remains aligned with current configuration.
- Existing menu/business behavior remains unchanged; only transport/session handling is migrated.
- Existing output backend behavior remains unchanged.
- Existing web server behavior remains unchanged.

## Rollback Plan

- Maintain ability to restore prior SSH server path if migration acceptance checks fail.
- Preserve prior behavior contract documentation so rollback criteria are objective.
- Use migration acceptance checks as go/no-go gates; if any critical check fails, revert to prior server implementation and re-open migration work.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% pass rate on migration acceptance checks for login, menu render, dispatch, clean exit, and unknown-option behavior.
- **SC-002**: 100% of validation runs confirm direct post-login landing in menu session for valid credentials.
- **SC-003**: 100% of tested valid numeric option submissions dispatch the expected handler on first Enter.
- **SC-004**: 100% of tested Ctrl+C, Ctrl+D, and session-close events terminate sessions cleanly without blocking subsequent logins.
- **SC-005**: 0 observed regressions in credentials/config usage, menu dispatcher integration, output backend wiring, and session directory isolation behavior.

## Assumptions

- Existing interactive menu text and behavior are the parity baseline.
- Existing credential/config values in current environments are valid and accessible.
- Local containerized validation environment is available for final validation execution.
- Implementation phase will respect deferred compilation/testing until validation stage.
