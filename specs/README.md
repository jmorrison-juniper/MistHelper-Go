# MistHelper-Go Feature Specs

This directory contains SpecKit feature specifications.

## Structure

Each feature lives in a numbered subdirectory:

```
specs/
├── 001-feature-name/
│   ├── spec.md        # Feature specification (speckit.specify output)
│   ├── plan.md        # Implementation plan (speckit.plan output)
│   ├── tasks.md       # Task list (speckit.tasks output)
│   ├── research.md    # Research notes (speckit.plan output)
│   └── contracts/     # API contracts
└── README.md          # This file
```

## Workflow

1. `speckit.specify` — Create spec from feature description
2. `speckit.clarify` — Surface underspecified areas (recommended)
3. `speckit.plan` — Generate implementation plan
4. `speckit.tasks` — Break plan into ordered tasks
5. `speckit.implement` — Execute tasks
6. `speckit.analyze` — Cross-check for consistency

## Escalation Criteria

Escalate to SpecKit (spec required before coding) when:
- Changes touch 3+ files or 2+ packages
- New menu operations or API integrations
- Architectural changes (new packages, interface changes, data flow)
- Bug fixes where root cause is unclear or spans multiple packages
- Any change to destructive operations (menu 90-100)
- Performance or concurrency work
- Database schema or primary key strategy changes

See `.specify/memory/constitution.md` for the full ruleset.
