# Contract: Wish SSH Session Compatibility

## Contract Intent

Define observable behavior that the Wish-based SSH migration must preserve and improve. This contract is client-visible and testable through SSH interactions.

## Interface Surface

### Startup and Bind

| Input | Behavior |
| - | - |
| `api.Config.SSHPort` | Server listens on configured SSH port. |
| Host key from `internal/ssh/hostkey.go` | Existing host key lifecycle remains valid. |

### Authentication

| Input | Behavior |
| - | - |
| Username/password from SSH client | Must be validated against `api.Config.SSHUser` and `api.Config.SSHPassword`. |
| Valid credentials | Session proceeds directly to interactive menu. |
| Invalid credentials | Session rejected; no menu session starts. |

### Session Lifecycle

| Event | Required Outcome |
| - | - |
| Successful login | Menu rendered without extra shell commands. |
| Valid option + Enter | Exactly one dispatch of matching handler path. |
| Unknown option + Enter | Unknown-option feedback shown; session remains active. |
| Ctrl+C / Ctrl+D | Session terminates cleanly. |
| Client disconnect | Session terminates cleanly; next login still works. |

### PTY and Input Handling Compatibility

| Concern | Contract Rule |
| - | - |
| Echo behavior | User keystrokes remain visible/consistent for interactive menu entry. |
| Enter handling | CR/LF variants do not produce missed dispatch or duplicate dispatch. |
| Terminal resize | Resize events do not break active menu input loop. |

### Session Workspace

| Rule |
| - |
| Each authenticated session must create an isolated directory using `data/sessions/session_<id>` semantics before dispatch loop starts. |

### Logging Semantics

| Rule |
| - |
| Lifecycle logs must preserve equivalent intent for server start, auth attempt/result, session start, session end, and errors (without secrets). |

## Non-Contractual Internal Details

The choice of middleware order, internal helper names, and private struct layout are implementation details and may change if this contract remains satisfied.

## Verification Mapping

- FR-002, FR-005, FR-006 map to authentication + dispatch behaviors.
- FR-007 maps to termination handling.
- FR-008 maps to session workspace rule.
- FR-009 maps to logging semantics.
- FR-010 maps to PTY/input compatibility rules.
