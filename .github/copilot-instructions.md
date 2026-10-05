# MistHelper-Go agent instructions

This file holds rules for MistHelper-Go only. The rules for each repository are in `AGENTS.md` at the repository root. Read `AGENTS.md` first. This file adds to it and does not copy its rules. Follow `AGENTS.md` for writing, safety, and security rules.

## What this repository is

MistHelper-Go is a Go application for Juniper Mist Cloud network operations. It runs in a container for deployment and supports direct execution during local development. Its audience includes junior network operations center engineers.

The Python project is the feature source. Build a requested feature in [MistHelper](https://github.com/jmorrison-juniper/MistHelper) first. Port it here after the Python feature is stable.

The executable registers 89 menu entries. Option 26, List Org Inventory, is implemented. The other 88 entries are placeholders. CSV and SQLite are the available output formats. See [current status](../docs/guide.md#current-status) before you describe or use an operation.

## Language and environment

`go.mod` sets the minimum Go version to 1.26.8. CI and the container builder use Go 1.27.1. The module uses `go mod`. Direct dependencies include `mistapi-go` v0.4.109, `godotenv` v1.5.1, and `modernc.org/sqlite` v1.60.1. Use Podman for container examples.

Run `go mod download` to prepare a new worktree. Copy `.env.example` to `.env` only when you need local credentials. Tests use fake API responses and need no Mist credentials.

Set `MIST_API_TOKEN` and `MIST_ORG_ID` for API operations. Set `SSH_PASSWORD` before deployment. `OUTPUT_FORMAT` selects CSV or SQLite, and `--format` overrides it. The executable also accepts `--menu N` and `--version`.

## Local gates

Run these commands from the repository root. A successful gate returns exit code zero.

| Gate | Command | Expected result |
| - | - | - |
| Format | `gofmt -l $(git ls-files '*.go')` | No file names |
| Static analysis | `go vet ./...` | No findings |
| Build and type check | `go build ./...` | All packages compile |
| Lint | `golangci-lint run ./...` | No findings |
| Tests | `go test ./... -race -cover` | All tests pass |
| Security lint | `gosec ./...` | No findings |
| Dependency scan | `govulncheck ./...` | No known vulnerabilities |
| Documentation | `node docs/check.mjs` | Documentation checks pass |
| Documentation tests | `node --test docs/check.test.mjs` | All six tests pass |
| STE, workstation | `ste-linter --config .ste-linter.toml --min-score 80 AGENTS.md .github/copilot-instructions.md` | Both files score at least 80 with the dictionary |
| STE, CI mode | `HOME=/tmp/ste-lint-no-dictionary STE_DICTIONARY_PATH=/nonexistent ste-linter --config .ste-linter.toml --min-score 80 AGENTS.md .github/copilot-instructions.md` | Both files score at least 80 without the dictionary |

Create `/tmp/ste-lint-no-dictionary` as an empty directory before the CI-mode command. CI also grades `README.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, and `.github/PULL_REQUEST_TEMPLATE.md`.

`go build ./...` includes the type check. This repository has no separate type-check tool. CI has no standalone format command. The `gofmt` formatter in `.golangci.yml` makes the lint gate fail when a Go file is not formatted.

## Architecture and conventions

The application flow is menu selection, Mist API request, response normalization, and CSV or SQLite output.

| Package | Purpose |
| - | - |
| `cmd/misthelper` | Application entry point and dependency wiring |
| `internal/api` | Mist SDK client, configuration, paging, and retry logic |
| `internal/menu` | Menu registry, terminal input, and operation dispatch |
| `internal/output` | Response flattening, CSV and SQLite writers, and endpoint key strategies |
| `internal/ssh` | SSH server and isolated session directories |
| `internal/web` | HTTP status and health routes |

Use `internal/output/strategies.go` for endpoint key strategies. The default API page limit is 1000. The API rate delay defaults to 200 milliseconds. The retry policy uses three attempts, a one-second base delay, and a 30-second maximum delay.

Pass dependencies through constructors. Wrap returned errors with `fmt.Errorf("context: %w", err)`. Use `menu.SafeInput` for menu and SSH input. It handles end-of-file input and carriage-return line endings. Use `docs/development.md` for the operation-porting steps.

## Safety in this repository

The dispatcher requires the exact word `CONFIRM` before it runs an operation marked `Destructive`. Set a strong `SSH_PASSWORD` before deployment.

The application writes CSV files, the SQLite database, and the SSH host key under `data/`. It creates one directory under `data/sessions/` for each SSH session. Do not place credentials or exported network data in source control.

## Containers and ports

The production image uses Go 1.27.1 on Alpine 3.24 to build the application. The runtime image uses Alpine 3.24 and runs as the non-root `misthelper` user. `Containerfile` exposes the SSH and HTTP ports.

| Port | Owner |
| - | - |
| 2200 | SSH menu |
| 8055 | HTTP status and health routes |

The repository has no Compose group or assigned test-port range. CI runs Go tests without a container stack. The container build workflow publishes `ghcr.io/jmorrison-juniper/misthelper-go`.

## Git and GitHub in this repository

Use the existing labels `bug`, `feature`, and `chore` for issue type. Use `docs`, `tests`, or `ci` for scope when those labels fit. Use `in-progress` while work is active. These names match the current repository labels.

Add user-visible changes to `CHANGELOG.md` under `[Unreleased]`. Use `YY.MM.DD.HH.MM` in UTC for version entries. This repository has no changelog-fragment folder.

The pull request template is `.github/PULL_REQUEST_TEMPLATE.md`. It lists the local quality gates and deployment checks.

| Workflow | File | Trigger or purpose |
| - | - | - |
| Quality Gates | `.github/workflows/ci.yml` | Go checks on pull requests and pushes to `main` |
| CodeQL | `.github/workflows/codeql.yml` | Code analysis on pull requests, pushes, and weekly schedule |
| STE lint | `.github/workflows/ste-lint.yml` | STE check on every pull request and on pushes to `main` |
| Documentation | `.github/workflows/docs.yml` | README and documentation regression checks |
| Container build | `.github/workflows/container-build.yml` | Image build on selected `main` changes or manual dispatch |
| Release | `.github/workflows/release.yml` | Image and release creation for version tags |

Branch protection requires CodeQL, Go tests, security scans, build, vet, and lint checks. The repository has the `auto-merge` label. The repository does not record workflow minute costs.

## Known pitfalls

- The lint gate fails when a Go file is not formatted. Run `gofmt -w` on each Go file that you change before you commit.
- The worktree `.git` file points to the main checkout. A container that mounts only the worktree cannot run `git ls-files`. Pass the file list from the host.

## Key files

| File | Purpose |
| - | - |
| `AGENTS.md` | Owner-wide agent rules |
| `go.mod` | Go version and module dependencies |
| `Containerfile` | Container build and runtime settings |
| `internal/api/config.go` | Runtime environment settings |
| `internal/output/strategies.go` | Endpoint key strategies |
| `internal/menu/dispatcher.go` | Operation dispatch and destructive confirmation |
| `docs/guide.md` | User instructions and current feature status |
| `docs/development.md` | Development and porting guidance |
| `CHANGELOG.md` | Release history |
| `.specify/memory/agent-context.md` | Spec Kit plan context |

Keep the Spec Kit plan context in `.specify/memory/agent-context.md`. Do not let Spec Kit overwrite `AGENTS.md`, this file, or `CLAUDE.md`.

## External resources

- [Mist API SDK](https://github.com/tmunzer/mistapi-go)
- [Mist API Python SDK](https://github.com/tmunzer/mistapi_python)
- [Mist API reference implementations](https://github.com/tmunzer/mist_library)
- [Python MistHelper](https://github.com/jmorrison-juniper/MistHelper)
