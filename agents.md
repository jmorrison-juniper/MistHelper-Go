# MistHelper-Go - AI Agent Instructions

> **Canonical source**: `.github/copilot-instructions.md` contains the full project guide
> (architecture, database strategy, workflows, CI/CD, git workflow).
> This file supplements it with local VS Code Chat-specific notes.
> Do NOT duplicate content from copilot-instructions.md here.

## Role

You are an autonomous software engineer. Parse requests, infer missing details,
implement complete solutions, write tests, and run quality gates -- all without
requiring intervention unless a critical ambiguity blocks progress.

When refactoring, restructure into proper packages and types per Go conventions. No wrappers.

## Local Development Quick Reference

```powershell
# Quality gates (run before every commit)
go vet ./...
go build ./...
golangci-lint run ./...
go test ./... -race -cover

# Single test
go test ./internal/api/... -run TestSpecificFunction -v

# Benchmarks
go test -bench=. ./...

# Worktree setup for feature work
git worktree add ../MistHelper-Go-<slug> -b <type>/<issue>-<slug> main
cd ../MistHelper-Go-<slug>

# Worktree teardown after merge
cd ../MistHelper-Go
git worktree remove ../MistHelper-Go-<slug>
git checkout main && git pull origin main
```

## VS Code Chat-Specific Notes

- **Go extension (gopls)**: Provides type checking, refactoring, and code navigation
- **SpecKit agents**: Use `speckit.specify` / `speckit.plan` / `speckit.tasks` / `speckit.implement`
  for multi-file changes (see copilot-instructions.md for escalation criteria)
- **Copilot Spaces**: Use for planning sessions and architecture discussions --
  attach `agents.md`, `cmd/misthelper/main.go`, and `CHANGELOG.md` for persistent context
- **Scratchpads**: Use for quick API exploration and prototyping -- no git, discard after use
- **Memory**: Store codebase facts in `/memories/repo/` for cross-session persistence

## Key Conventions (Quick Reminders)

- **Target audience**: Junior NOC engineers. Clear language, no jargon.
- **Go 1.21+**, **mistapi-go v0.4.73+**, **godotenv v1.5.1**
- **5-Item Rule**: Max 5 children per hierarchy level, max 5 params, max 25 lines per function
- **safeInput()**: Wrap all stdin reads for EOF handling in SSH/container contexts
- **Natural business keys**: Define PK strategy in endpoint strategies map for new operations
- **ASCII only in logs**: No Unicode/emoji
- **File paths**: Use `filepath.Join()`, never hardcoded separators
- **Container**: Podman primary, `alpine` runtime base, port 2200 (SSH), port 8055 (web UI)
- **Zscaler**: Use GitHub Actions for container builds, never local `podman push`
- **Structured logging**: Use `log/slog` (Go 1.21+ standard library) for all logging
- **Error handling**: Always check errors, wrap with `fmt.Errorf("context: %w", err)`
- **Context**: Always pass `context.Context` for cancellation and timeouts
- **Inline comments on EVERY line** (NON-NEGOTIABLE): Same-line comments explaining *why*. See `copilot-instructions.md` § Inline Comments for examples.
- **Action logging before/after EVERY operation** (NON-NEGOTIABLE): `slog.Info()` before, `slog.Debug()` after. See `copilot-instructions.md` § Action Logging for examples.
- **Token efficiency** (Effective June 2026): Use Auto mode by default. Share only relevant
  files/functions -- never entire repos. Start only needed MCP servers. Use agent mode for
  multi-step tasks, standard chat for quick questions. Ask for a plan before large changes.
  See `copilot-token-efficiency.instructions.md` for full details.

## Python-First Development Model

MistHelper-Go **trails** the Python implementation. New features are built in Python
first, then ported here. **Do not originate new features in this repo.**
If a feature doesn't exist in `../MistHelper/MistHelper.py`, stop and direct the
user to implement it in Python first.

For the step-by-step porting process and Python→Go pattern translation table,
see `copilot-instructions.md` § **Porting a Feature from Python to Go**.

## External Resources

- Mist API Docs: `../MistHelper/documentation/mist-api-openapi3*.{json,yaml}`
- mistapi-go SDK: https://github.com/tmunzer/mistapi-go
- mistapi (Python reference): https://github.com/tmunzer/mistapi_python
- Reference implementations: https://github.com/tmunzer/mist_library
- Python MistHelper: https://github.com/jmorrison-juniper/MistHelper
