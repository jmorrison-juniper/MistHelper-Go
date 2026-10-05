# Spec Conformance Checklist

**Linked Spec Issue**: #<!-- Issue number -->

## Acceptance Criteria
- [ ] All acceptance criteria from the linked Spec Issue are met
- [ ] Each criterion has a corresponding test or verification

## Quality
- [ ] Tests added or updated for all changed functionality
- [ ] `go vet ./...` passes clean
- [ ] `golangci-lint run ./...` passes clean
- [ ] `go build ./...` compiles successfully
- [ ] `go test ./... -race -cover` passes with adequate coverage

## Security
- [ ] No hardcoded secrets, tokens, or passwords
- [ ] `gosec ./...` passes with no new findings
- [ ] `govulncheck ./...` clean (no known CVEs in dependencies)
- [ ] Sensitive data handled via `.env` / environment variables only

## Deployment
- [ ] Dry-run verified locally (ran affected menu operations)
- [ ] `.env` changes documented in `.env.example` (if applicable)
- [ ] Container builds successfully (if Containerfile changed)

## HTTP routes (if `internal/web` changed)
- [ ] `internal/web` tests cover each changed JSON route (`/` and `/health`)

## Documentation
- [ ] README.md updated (if user-facing changes)
- [ ] Changelog entry added with version `YY.MM.DD.HH.MM` format
- [ ] `node docs/check.mjs` and `node --test docs/check.test.mjs` pass
- [ ] Each changed Markdown file scores 80 or more with `ste-linter --config .ste-linter.toml --min-score 80`
