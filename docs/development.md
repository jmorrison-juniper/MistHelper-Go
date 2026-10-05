# Development guide

## Goals and development model

MistHelper-Go trails [Python MistHelper](https://github.com/jmorrison-juniper/MistHelper).
Develop and stabilize features in Python first, then port them here. Do not
originate new product features in this repository.

Goals, not claims about the current build:

- A single static Go executable with no Python runtime.
- A container near 25 MB rather than the Python image near 500 MB.
- Feature parity with 193 Python menu operations; Python now extends to 194.
- Mist API access through the official [mistapi-go SDK](https://github.com/tmunzer/mistapi-go).
- CSV, SQLite, and future ArangoDB/Redis outputs.

Use the [current status](guide.md#current-status) to distinguish implemented
behavior from these goals.

## Local development

Use Go 1.26.8 or later. CI and the container builder use Go 1.27.1.
Normal API operations require a Mist API token and an organization ID.
Tests use fake responses and do not need either credential.

```bash
cp .env.example .env
# Set credentials only when intentional live API work is authorized.
go run ./cmd/misthelper
go run ./cmd/misthelper --version
```

Without `--menu`, terminal input opens the interactive menu. Without a terminal,
the process waits for a shutdown signal and serves SSH and HTTP.
`--menu N` runs one operation; `--menu 0` exits cleanly.
`--format csv` or `--format sqlite` overrides `OUTPUT_FORMAT`.

See the [offline checks](ci.md#offline-tests) before committing, and
[screen capture instructions](interface.md#capture-method) for a run that needs
no real Mist credentials.

## Source layout

```text
cmd/misthelper/    Main entrypoint and operation handlers
internal/api/     Mist SDK client, configuration, pagination, retries
internal/menu/    Terminal menu registry, display, input, dispatch
internal/output/  CSV and SQLite writers, flattening, key strategies
internal/ssh/     SSH menu sessions and persisted host key
internal/web/     HTTP JSON status and health handlers
data/             Runtime output; ignored by Git
docs/             User, developer, CI, and interface documentation
specs/            Feature specifications and implementation plans
deploy/           Container service examples
```

## Porting an operation

Locate the stable Python operation and its API calls, primary-key strategy,
prompts, and output columns. Reuse the existing Go client, flattening helpers,
and writer interface. Pass the endpoint name for key-strategy lookup.
Keep the Python behavior and user-facing wording.

Define the key strategy before implementing output. Use contexts for
cancellation, wrap errors, and use the safe input helper for EOF handling.
Never automate destructive operations without explicit confirmation.
The [contributor instructions](../.github/copilot-instructions.md) and
[feature specifications](../specs/README.md) contain the detailed conventions.

## Release history

See [CHANGELOG.md](../CHANGELOG.md). Versions use `YY.MM.DD.HH.MM` in UTC.
Deployment artifacts are container images; there is no supported standalone
host installation.
