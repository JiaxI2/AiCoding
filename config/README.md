# Configuration Catalog

<!-- AICODING:REPOSITORY_MAP:START -->
## Scope

Machine-readable configuration source-of-truth.

## Ownership

- Purpose: Machine-readable platform configuration, registries, policies and schemas.
- Audience: maintainer, agent
- Entry: `config/README.md`

## Rule

Do not create a parallel source of truth outside this domain. Add new items only when they have a distinct lifecycle and owner.

## Architecture governance

- `dependency-governance.json`: layer direction, registry bindings, namespace ownership, Skill/MCP responsibility, stable identity and README version badge authority.
- `schemas/dependency-governance.schema.json`: machine schema for that policy.
- `schemas/cli-report.schema.json`: `report.Result`, `StandardReport` and shared product-check JSON contract.
- `mcp-registry.json` and `mcp/components/*.json`: upper-layer MCP composition and runtime injection.
<!-- AICODING:REPOSITORY_MAP:END -->
