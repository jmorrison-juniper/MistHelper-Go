# CI and offline verification

## Offline tests

Tests need no Mist token, `.env` file, or cloud account. The harness uses fake
HTTP responses for the Mist SDK, local CSV and SQLite files, and loopback
connections for SSH and web tests. It covers pagination, request errors, input
EOF, canceled requests, output records, and failed authentication.

Run from a writable checkout: one executable smoke test creates `data/`.

```bash
go mod download
go mod verify
go vet ./...
go build ./...
go test ./... -race -cover -count=1
```

After the first download, set `GOPROXY=off GOSUMDB=off` when running tests to
confirm that no remote services are needed.

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
go install github.com/securego/gosec/v2/cmd/gosec@v2.29.0
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
golangci-lint run ./...
gosec ./...
govulncheck ./...
node docs/check.mjs
node --test docs/check.test.mjs
```

The `gofmt` formatter in `.golangci.yml` makes `golangci-lint run` fail when a
Go file is not formatted. Run `gofmt -w` on the file to repair it.
`govulncheck` needs the public Go advisory database. These checks do not contact
the Mist API. The documentation check needs Node.js 18 or later, with no extra
packages. It checks the landing headings, relative links, screenshot references,
and PNG headers.
The regression tests confirm that wrong headings, broken links or anchors,
duplicate screenshots, and non-PNG content fail the check. `docs.yml` runs both
commands for README and documentation changes.

## Shared workflows

Shared workflows live in
[misthelper-devtools](https://github.com/jmorrison-juniper/misthelper-devtools).
Each caller pins a release commit; a comment names the release tag.

| Workflow here | Shared workflow |
| --- | --- |
| `ci.yml` (quality-gate issues job) | `reusable-quality-gate-issues.yml` |
| `auto-merge.yml` | `reusable-auto-merge.yml` |
| `close-linked-issues.yml` | `reusable-close-linked-issues.yml` |
| `container-build.yml` (build-and-push job) | `reusable-container-image.yml` |
| `release.yml` (build-container job) | `reusable-container-image.yml` |

The `auto-merge` label merges with `GITHUB_TOKEN`, which starts no push run on
`main`. The shared workflow then dispatches `ci.yml`, `codeql.yml`, and
`container-build.yml`. Do not bypass branch protection or add the auto-merge
label before CodeQL passes.
