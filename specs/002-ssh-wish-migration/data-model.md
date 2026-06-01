# Phase 1 Data Model - SSH Wish Migration

## Overview

This feature changes transport/session orchestration, not domain business entities. The data model below captures runtime entities and contracts that must remain behavior-compatible.

## Entities

### 1) SSH Runtime Configuration

| Field | Type | Source | Rules |
| - | - | - | - |
| `SSHPort` | `int` | `api.Config` | Must bind listener port used by SSH server. |
| `SSHUser` | `string` | `api.Config` | Must be required for password-auth username match. |
| `SSHPassword` | `string` | `api.Config` | Must be required for password-auth verification; never logged. |

Validation notes: semantics remain unchanged from current implementation.

### 2) SSH Session Context

| Field | Type | Source | Rules |
| - | - | - | - |
| `SessionID` | `string` | `newSessionID()` | Format `YYYYMMDD_HHMMSS_XXXX`; unique per active session. |
| `RemoteAddr` | `string` | SSH connection metadata | Logged for auth/session tracing; no credential data. |
| `PTYState` | runtime struct / flags | Wish session metadata | Must support reliable line input + Enter dispatch behavior. |
| `LifecycleState` | enum-like string | session lifecycle transitions | `accepted -> authenticated -> active -> terminated`. |

Validation notes: session transitions must be observable via structured logs.

### 3) Session Workspace

| Field | Type | Source | Rules |
| - | - | - | - |
| `BaseDir` | `string` | `data/sessions` (or test override) | Must remain compatible with existing directory layout. |
| `SessionDir` | `string` | `filepath.Join(BaseDir, "session_"+SessionID)` | Must be created before menu dispatch begins. |
| `CreateResult` | `error` or nil | filesystem operation | Failure must terminate only the affected session path cleanly. |

Validation notes: per-session isolation is required; no shared mutable workspace path.

### 4) Menu Session Binding

| Field | Type | Source | Rules |
| - | - | - | - |
| `Registry` | `*menu.Registry` | existing startup wiring | Must remain shared read-only menu registry. |
| `InputReader` | `*bufio.Reader` | Wish session stream | Must produce reliable line reads for Enter dispatch. |
| `OutputWriter` | `io.Writer` | Wish session stream adapter | Must preserve terminal-friendly rendering and echo expectations. |
| `DataWriter` | `output.Writer` | existing startup wiring | Must remain unchanged for backend output behavior. |

Validation notes: unknown-option and valid-dispatch behavior remain menu-owned.

## Relationships

1. `SSH Runtime Configuration` configures the `Wish SSH server`.
2. Each authenticated connection creates one `SSH Session Context`.
3. Each `SSH Session Context` owns one `Session Workspace`.
4. Each active session instantiates one `Menu Session Binding` tied to existing dispatcher/output components.

## State Transitions

| Transition | Trigger | Required Behavior |
| - | - | - |
| `accepted -> authenticated` | Valid username/password | Continue to menu session startup. |
| `accepted -> terminated` | Invalid credentials / handshake failure | Reject cleanly; emit auth failure logs. |
| `authenticated -> active` | Session workspace + menu bridge ready | Render menu immediately (ForceCommand-like UX). |
| `active -> terminated` | Ctrl+C, Ctrl+D, disconnect, or menu exit | Clean shutdown of session resources and logs. |

## Invariants

1. Auth semantics remain sourced from `api.Config` only.
2. Session directory semantics remain `data/sessions/session_<id>`.
3. One submitted line must map to one dispatch attempt.
4. Output backend wiring remains unchanged.
5. No compile/run actions occur before validation stage.
