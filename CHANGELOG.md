# Changelog

All notable changes to MistHelper-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versions use UTC timestamp format: `YY.MM.DD.HH.MM`.

## [Unreleased]

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
