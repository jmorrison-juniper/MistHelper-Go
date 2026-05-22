# MistHelper-Go Development Guidelines

Auto-generated from all feature plans. Last updated: [DATE]

## Active Technologies

- **Go**: 1.21+
- **mistapi-go**: v0.4.73+ (Thomas Munzer's Go Mist API SDK)
- **godotenv**: v1.5.1
- **log/slog**: Go standard library structured logging
- **golangci-lint**: Multi-linter aggregator
- **gosec**: Go security linter
- **govulncheck**: Dependency CVE scanner
- **Podman**: Container runtime (primary)

## Project Structure

```text
cmd/
└── misthelper/
    └── main.go

internal/
├── api/          # Mist API client wrapper (mistapi-go)
├── menu/         # TUI menu system
├── output/       # CSV, SQLite, ArangoDB, Redis writers
├── ssh/          # SSH server (port 2200)
└── web/          # Web UI (port 8055)

data/             # Runtime output (git-ignored)
specs/            # SpecKit feature specs
.specify/         # SpecKit configuration and templates
```

## Commands

```powershell
# Quality gates (run before every commit)
go vet ./...
go build ./...
golangci-lint run ./...
go test ./... -race -cover

# Single test
go test ./internal/[pkg]/... -run TestFunctionName -v

# Benchmarks
go test -bench=. ./...
```

## Code Style

- Inline comments on **every** executable line (NON-NEGOTIABLE)
- `slog.Info()` before every action, `slog.Debug()` after (NON-NEGOTIABLE)
- `fmt.Errorf("context: %w", err)` for all error wrapping
- `filepath.Join()` for all file paths (never `/` or `\\`)
- `safeInput()` for all stdin reads in SSH/container contexts
- Full variable names, no abbreviations

## Recent Changes

[LAST 3 FEATURES AND WHAT THEY ADDED]

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
