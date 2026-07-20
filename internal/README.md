# Internal Package Map

<!-- AICODING:REPOSITORY_MAP:START -->
## Scope

Go implementation package ownership and dependency direction.

## Ownership

- Purpose: Go platform implementation packages.
- Audience: developer
- Entry: `internal/README.md`

## Rule

Do not create a parallel source of truth outside this domain. Add new items only when they have a distinct lifecycle and owner.

## Product authority packages

| Package | Sole responsibility |
|---|---|
| `cli` | typed command catalog, command parsing, help, JSON stdout and exit codes |
| `lifecycle` | static Kit, MCP and runtime Skill adapter composition |
| `repohealth` | product doctor and deterministic verify checks |
| `testengine` | Smoke, Full and Release test registry, execution, timeout and reports |
| `validationevidence` | Git Tree validation identity, immutable PASS Receipts and fail-closed reuse checks |
| `report` | `Result`, `StandardReport`, shared checks, `errorKind` and schema contract |
<!-- AICODING:REPOSITORY_MAP:END -->
