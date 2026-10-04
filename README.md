# MistHelper-Go

Go rewrite of [MistHelper](https://github.com/jmorrison-juniper/MistHelper) — a production-grade tool for Juniper Mist Cloud network operations.

## Status

**Active development — foundational scaffold complete.** All 5 internal packages are wired and tested. 89 menu stubs are registered; each stub will be replaced with a real implementation as operations are ported from the Python version.

See [MistHelper](https://github.com/jmorrison-juniper/MistHelper) for the production Python version.

## Development Model

MistHelper-Go **trails** the [Python MistHelper](https://github.com/jmorrison-juniper/MistHelper). Features are developed and stabilized in the Python repo first, then ported here. Do not request or propose new features for this repo unless they already exist in the Python version.

## Goals

- Python-free: single static binary, no runtime dependencies
- ~25MB container image (vs ~500MB Python equivalent)
- Full feature parity with MistHelper (193 menu operations; Python is now at Menu 194 — Go ports trail)
- Built on [`tmunzer/mistapi-go`](https://github.com/tmunzer/mistapi-go) — the official Go Mist API SDK

## Requirements (Development Only)

- Go 1.26.8+ (`go.mod` sets the minimum; CI and the container build use Go 1.27.1)
- Juniper Mist API token (set in `.env`)

MistHelper-Go is designed to run exclusively from a container in production. Direct binary execution is for local development only.

## Quick Start (Container)

```bash
cp .env.example .env
# Edit .env with your Mist API token and org ID
podman pull ghcr.io/jmorrison-juniper/misthelper-go:latest
podman run -d --name misthelper-go \
  -p 2200:2200 -p 8055:8055 \
  -v "${PWD}/data:/app/data:rw" \
  -v "${PWD}/.env:/app/.env:ro" \
  ghcr.io/jmorrison-juniper/misthelper-go:latest
```

## Quick Start (Local Development)

```bash
cp .env.example .env
# Edit .env with your Mist API token and org ID
go run ./cmd/misthelper
```

## Project Structure

```text
cmd/misthelper/     # main entrypoint
internal/
  api/              # Mist API client wrapper
  menu/             # TUI menu system
  output/           # CSV, SQLite, ArangoDB, Redis writers
  ssh/              # SSH server (port 2200)
  web/              # Web UI (port 8055)
data/               # Runtime output directory
specs/              # SpecKit feature specs
```

## CI Tooling

### Offline tests

Tests need no Mist token, `.env` file, or cloud account. The harness uses fake
HTTP responses for the Mist SDK, local files for CSV and SQLite, and loopback
connections for SSH and web tests. It covers pagination, request errors, input
EOF, cancelled requests, output records, and failed authentication.

```bash
go mod download
go mod verify
go vet ./...
go build ./...
go test ./... -race -cover -count=1
```

After the first download, run tests with `GOPROXY=off GOSUMDB=off` to confirm
that they need no remote services. For the same lint and advisory checks as CI:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
go install github.com/securego/gosec/v2/cmd/gosec@v2.29.0
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
golangci-lint run ./...
gosec ./...
govulncheck ./...
```

`govulncheck` needs access to the public Go advisory database. These checks do
not contact the Mist API. CI pins tool releases so each run uses the same tools.

The shared CI workflows live in [misthelper-devtools](https://github.com/jmorrison-juniper/misthelper-devtools). Each caller pins a devtools release commit, and a comment names the release tag.

| Workflow here | Shared workflow |
| --- | --- |
| `ci.yml` (quality-gate issues job) | `reusable-quality-gate-issues.yml` |
| `auto-merge.yml` | `reusable-auto-merge.yml` |
| `close-linked-issues.yml` | `reusable-close-linked-issues.yml` |
| `container-build.yml` (build-and-push job) | `reusable-container-image.yml` |
| `release.yml` (build-container job) | `reusable-container-image.yml` |
| `copilot-label-checkbox.yml`, `copilot-auto-assign.yml` | `reusable-copilot-assign.yml` |

A merge by the auto-merge label uses `GITHUB_TOKEN`, so it starts no push run on `main`. The auto-merge workflow then starts `ci.yml`, `codeql.yml`, and `container-build.yml` with a `workflow_dispatch` call.

The Copilot workflows need the `COPILOT_ASSIGN_TOKEN` repository secret. GitHub assigns the Copilot cloud agent only for a user token, so `GITHUB_TOKEN` cannot do it. Use a fine-grained token with read access to metadata, and read and write access to actions, contents, issues, and pull requests. The token user must have the Copilot cloud agent enabled for this repository. Without the secret, the workflow writes a comment on the issue with the cause. To give an issue that already exists to the agent, run `copilot-auto-assign.yml` by hand with the issue number.

## License

Apache 2.0
