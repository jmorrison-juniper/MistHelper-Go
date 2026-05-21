# MistHelper-Go

Go rewrite of [MistHelper](https://github.com/jmorrison-juniper/MistHelper) — a production-grade tool for Juniper Mist Cloud network operations.

## Status

**In development.** Not yet feature-complete. See [MistHelper](https://github.com/jmorrison-juniper/MistHelper) for the production Python version.

## Goals

- Python-free: single static binary, no runtime dependencies
- ~25MB container image (vs ~500MB Python equivalent)
- Full feature parity with MistHelper (187 menu operations)
- Built on [`tmunzer/mistapi-go`](https://github.com/tmunzer/mistapi-go) — the official Go Mist API SDK

## Requirements

- Go 1.21+
- Juniper Mist API token (set in `.env`)

## Quick Start

```bash
cp .env.example .env
# Edit .env with your Mist API token and org ID
go run ./cmd/misthelper
```

## Container

```bash
podman build -t misthelper-go .
podman run -d --name misthelper-go \
  -p 2200:2200 -p 8055:8055 \
  -v "${PWD}/data:/app/data:rw" \
  -v "${PWD}/.env:/app/.env:ro" \
  misthelper-go
```

## Project Structure

```
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
