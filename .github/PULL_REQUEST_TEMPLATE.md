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

## UI / E2E Testing (if web UI changed)
- [ ] E2E tests added/updated for changed UI flows
- [ ] Stable `data-testid` attributes added for new interactive elements
- [ ] AI agent verified selectors via VS Code Browser Agent Tools
- [ ] Screenshots/traces captured for main UI flows (attached or in CI artifacts)

## Documentation
- [ ] README.md updated (if user-facing changes)
- [ ] Changelog entry added with version `YY.MM.DD.HH.MM` format
