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
- `validation-policy.json`: pre-push Context Gate 的远端 ref、必需验证 profile、快进与删除策略。
- `impact-policy.json`: 影响规则的单一文件；当前 `raceScope.packages` 登记 Full race 包集合，GO-007 机器阻断并发包漏登。Release 不读取该缩减集合，始终全仓 race。
- `mcp-registry.json` and `mcp/components/*.json`: upper-layer MCP composition and runtime injection.
- `templates/provision/`: `aicoding provision` 编译期内嵌的最小 SDD 文档骨架单一来源。
- `templates/kit/`: `aicoding kit init` 编译期内嵌的 manifest、workspec 与外部边界卡单一来源。
<!-- AICODING:REPOSITORY_MAP:END -->
