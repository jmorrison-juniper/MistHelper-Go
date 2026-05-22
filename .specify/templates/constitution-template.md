# [PROJECT_NAME] Constitution
<!-- Example: MistHelper-Go Constitution -->

## Core Principles

### [PRINCIPLE_1_NAME]
<!-- Example: I. Five-Item Rule -->
[PRINCIPLE_1_DESCRIPTION]
<!-- Every level of the project hierarchy must contain no more than five children.
     Hierarchy: Project Root -> Packages/Directories -> Source Files -> Types/Functions -> Methods/Fields
     Hard limits: max 5 params, max 5 logical blocks, max 25 lines per function -->

### [PRINCIPLE_2_NAME]
<!-- Example: II. Package-Based Architecture (No Wrappers) -->
[PRINCIPLE_2_DESCRIPTION]
<!-- All functionality lives within semantically named packages and types (interfaces + structs).
     No standalone wrapper functions. Restructure into proper packages when refactoring.
     Full variable names - no abbreviations. -->

### [PRINCIPLE_3_NAME]
<!-- Example: III. Safety-First (NON-NEGOTIABLE) -->
[PRINCIPLE_3_DESCRIPTION]
<!-- Use safeInput() with EOF handling for all stdin reads.
     NASA/JPL confirmation pattern for destructive operations.
     Validate early, return early. Never log secrets. Always pass context.Context. -->

### [PRINCIPLE_4_NAME]
<!-- Example: IV. Full Deployment Pipeline (NON-NEGOTIABLE) -->
[PRINCIPLE_4_DESCRIPTION]
<!-- go vet -> go build -> golangci-lint -> go test -race -cover -> commit -> push -> CI -> container pull/restart.
     No steps may be skipped. Every changelog update triggers this pipeline. -->

### [PRINCIPLE_5_NAME]
<!-- Example: V. Observability & Logging -->
[PRINCIPLE_5_DESCRIPTION]
<!-- log/slog structured key-value logging. ASCII only (no emoji). Debug/Info/Error levels.
     Secrets never logged. Structured, machine-parseable entries. -->

### [PRINCIPLE_6_NAME]
<!-- Example: VI. Inline Comments (NON-NEGOTIABLE) -->
[PRINCIPLE_6_DESCRIPTION]
<!-- Every executable line of AI-generated code MUST have an inline comment explaining why, not just what.
     Blank lines, closing braces, package/import declarations are exempt.
     Code without inline comments is incomplete and MUST NOT be committed. -->

### [PRINCIPLE_7_NAME]
<!-- Example: VII. Action Logging (NON-NEGOTIABLE) -->
[PRINCIPLE_7_DESCRIPTION]
<!-- slog.Info() before every action, slog.Debug() after with result summary.
     slog.Error() with full context on any error.
     Code without action logging is code without observability. MUST NOT be committed. -->

## [SECTION_2_NAME]
<!-- Example: Technology & Compatibility Constraints -->

[SECTION_2_CONTENT]
<!-- Go 1.21+, mistapi-go v0.4.73+, godotenv v1.5.1, filepath.Join, log/slog,
     Podman primary, GitHub Actions for container builds, data/ output directory -->

## [SECTION_3_NAME]
<!-- Example: Development Workflow & Quality Gates -->

[SECTION_3_CONTENT]
<!-- New operations: API discovery -> PK strategy -> flatten -> output.Writer -> README -> CHANGELOG -> pipeline.
     Testing: go test ./... -race -cover. Security: gosec/govulncheck/CodeQL, fix over suppress. -->

## Governance
<!-- Constitution supersedes all other practice documents.
     Amendments: update constitution + increment semver + update LAST_AMENDED_DATE.
     Compliance review on every PR. Principles VI and VII are non-negotiable quality gates. -->

[GOVERNANCE_RULES]
<!-- Runtime guidance in agents.md (daily) and .github/copilot-instructions.md (comprehensive).
     Python reference at ../MistHelper/MistHelper.py for behavior parity. -->

**Version**: [CONSTITUTION_VERSION] | **Ratified**: [RATIFICATION_DATE] | **Last Amended**: [LAST_AMENDED_DATE]
<!-- Example: Version: 1.0.0 | Ratified: 2026-05-21 | Last Amended: 2026-05-21 -->
