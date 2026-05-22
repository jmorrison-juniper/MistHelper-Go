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
- Full feature parity with MistHelper (187 menu operations)
- Built on [`tmunzer/mistapi-go`](https://github.com/tmunzer/mistapi-go) — the official Go Mist API SDK

## Requirements (Development Only)

- Go 1.21+
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

## License

Apache 2.0
