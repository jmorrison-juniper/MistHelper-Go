# Changelog

All notable changes to MistHelper-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions use UTC timestamp format: `YY.MM.DD.HH.MM`.

## [Unreleased]

### Changed

- 26.10.05.06.27: Add `.golangci.yml` with the `gofmt` formatter, so the lint gate
  fails when a Go file is not formatted (#67).
- 26.10.05.06.23: Format the 15 Go files that `gofmt -l` listed. The change is
  whitespace only (#67).
- 26.10.05.06.19: Remove the host systemd unit `deploy/misthelper-go.service`,
  because deployment is container-only. The Podman Quadlet unit
  `deploy/misthelper-go.container` now publishes its ports on the local host
  only, and the guide tells how to install it (#70).
- 26.10.05.06.15: Pull request template: replace the web UI and end-to-end test
  section with a check for the JSON routes, and add the documentation and
  STE checks (#70).
- 26.10.05.06.10: Spec Kit: the constitution (v1.0.1) and the plan, agent-file, and
  constitution templates name Go 1.26.8, mistapi-go v0.4.109, the CSV and
  SQLite writers, and `AGENTS.md` (#70).
- 26.10.05.06.04: Config: `.env.example` lists only the variables that the code reads:
  `OUTPUT_FORMAT`, `API_RATE_LIMIT_MS`, `SSH_PORT`, `SSH_USER`,
  `SSH_PASSWORD`, and `WEB_PORT`. The ArangoDB and Redis variables stay as
  comments marked as planned. The duplicate `deploy/.env.example` is removed
  (#70).

### Removed

- 26.10.05.06.00: CI: remove the Copilot cloud-agent workflows `copilot-auto-assign.yml`,
  `copilot-label-checkbox.yml`, and `copilot-setup-steps.yml`, and the
  Copilot assignment checkbox in the issue templates. The owner does not use
  the Copilot cloud agent (#69).

### Documentation

- 26.10.05.05.07: adopt the owner-wide `AGENTS.md`, replace duplicate agent
  guidance with repository-specific instructions, and grade both files with
  the shared STE workflow.
- 26.10.05.01.40: reorganize the landing README into What, How, Where, When,
  Why, and Who; move usage, development, CI, and interface details under
  `docs/`; embed three genuine offline terminal screenshots and document the
  absent graphical dashboard as N/A (#64).
- Clarify the implemented inventory operation, placeholder entries, available
  output backends, and current runtime configuration; add a dependency-free
  documentation structure/link/screenshot check.

### Security

- Deps: `golang.org/x/crypto` v0.53.0 → v0.57.0 fixes three `x/crypto/ssh` advisories that the SSH server reaches (GO-2026-6303, GO-2026-6354, GO-2026-6355); `golang.org/x/term` v0.46.0 and `golang.org/x/sys` v0.48.0 follow
- Build: the Go toolchain moves from 1.25.11, which is out of support, to 1.27.1 in CI and the container build stage; the runtime stage moves from `alpine:3.19` (end of life) to `alpine:3.24`
- `go.mod` now sets `go 1.26.0`, the minimum that `golang.org/x/crypto` v0.57.0 needs

### Fixed

- CI: `govulncheck` runs again; `govulncheck@latest` needs Go 1.26 or later (#38)
- CI: the auto-merge dispatch no longer starts a second `ci.yml`, `codeql.yml`, or `container-build.yml` run on a tip that already has a run. The run list of GitHub can answer from old data, so misthelper-devtools v0.5.2 looks for a run on the tip commit itself (jmorrison-juniper/misthelper-devtools#32)
- Copilot: the setup steps file moves to `.github/workflows/copilot-setup-steps.yml`, the only path that GitHub reads, and sets Go 1.27.1 instead of 1.21
- CI: the Copilot assign workflows never assigned the agent. GitHub assigns the Copilot cloud agent only for a user token, and misthelper-devtools v0.2.0 sent the request with `GITHUB_TOKEN`. With v0.3.0, both workflows pass the `COPILOT_ASSIGN_TOKEN` secret, and without it they write a comment on the issue with the cause

### Changed

- Deps: `mistapi-go` v0.4.108 -> v0.4.109 and `modernc.org/sqlite` v1.59.0 -> v1.60.1; refresh their runtime dependencies and checksums.
- Build: require Go 1.26.8 or later and pin the container builder to Go 1.27.1 on Alpine 3.24.
- Build: exclude local tools, credentials, data, and test artifacts from the container build context.
- CI: pin current stable action and lint tool releases; document the existing offline harness and add SDK HTTP contract tests.
- CI: the shared workflows call [misthelper-devtools](https://github.com/jmorrison-juniper/misthelper-devtools) v0.6.0, pinned by commit SHA
- Deps: `modernc.org/sqlite` v1.52.0 → v1.59.0 (#43)
- Deps: `github.com/tmunzer/mistapi-go` v0.4.103 → v0.4.108; `GetOrgInventory` takes a new `disconnectedBefore` filter, and the inventory export leaves it unset, so the export still returns every device (#44)
- CI: the quality-gate issue, auto-merge, linked-issue close, container build, release image, and Copilot assign workflows call the shared workflows in [misthelper-devtools](https://github.com/jmorrison-juniper/misthelper-devtools) v0.6.0, pinned by commit SHA
- CI: `codeql.yml` calls the shared CodeQL workflow in misthelper-devtools, which keeps the floating `v4` tag of `github/codeql-action`; PR #28 replaced that tag with an exact version. The analysis key stays `.github/workflows/codeql.yml:analyze`, so code scanning keeps its alerts, and the check name becomes `codeql / Analyze (go)`
- CI: after an auto-merge, `ci.yml`, `codeql.yml`, and `container-build.yml` start with `workflow_dispatch`, because a merge by `GITHUB_TOKEN` starts no push run
- CI: a checkbox issue now gets the Copilot Coding Agent assigned in the same run that adds the `copilot` label, when the `COPILOT_ASSIGN_TOKEN` secret is set
- CI: `copilot-auto-assign.yml` starts only by hand; the labeled trigger started a run for every label on every issue, and almost every run skipped
- Release: the release image gets the OCI labels from the shared workflow; the tags stay the version without the `v` and `latest`
- Deps: Dependabot checks the `Containerfile` base images each week

## [26.05.22.00.00] - 2026-05-22

### Added

- `internal/api`: `Config` struct, `LoadConfig`, exponential-backoff retry, `mistapi-go` `Client` wrapping `ListSites`
- `internal/output`: `FlattenRecord`, 46-entry `ENDPOINT_PRIMARY_KEY_STRATEGIES` map ported from Python, CSV writer, CGO-free SQLite writer (`modernc.org/sqlite`), `INSERT OR REPLACE` upsert for natural/composite PKs
- `internal/menu`: `SafeInput` (EOF-safe), `Entry`/`Registry`, ASCII TUI `PrintMenu`, `Dispatcher` with ForceCommand pattern and destructive-confirm guard
- `internal/ssh`: RSA 2048-bit host key generate-on-first-boot/persist, password-only SSH server on port 2200, session isolation (`data/sessions/session_<id>/`)
- `internal/web`: HTTP status/health server on port 8055 (`GET /` → ready JSON, `GET /health` → ok JSON)
- `cmd/misthelper`: wired `main.go` — loads config, constructs all five packages, registers 89 stub handlers, starts SSH+web in goroutines, runs interactive menu or `--menu N` direct dispatch, graceful shutdown (SSH 30 s drain → web 5 s)
- SpecKit spec/plan/tasks for foundational scaffolding (`specs/001-foundational-scaffolding/`)
- 27+ unit tests across all 6 packages; `go vet`, `go build`, `golangci-lint` all pass
