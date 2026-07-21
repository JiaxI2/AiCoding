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
- `internal-capabilities.json`: 28 个 `internal/` 一级包的唯一能力目录，登记类型、稳定态、公共入口、架构文档与验证命令。
- `schemas/internal-capabilities.schema.json`: 能力目录的结构与分级文档义务 schema。
- `schemas/cli-report.schema.json`: `report.Result`, `StandardReport` and shared product-check JSON contract.
- `validation-policy.json`: pre-push Context Gate 的远端 ref、必需验证 profile、快进与删除策略。
- `impact-policy.json`: 影响规则的单一文件；`raceScope.packages` 登记 Full race 包集合，GO-007 机器阻断并发包漏登，Release 仍始终全仓 race；`changeVerify` 以与 Plan Mode 同构的路径 pattern 将变更确定性映射到 Smoke/Full，未命中路径保守选择 Full。
- `mcp-registry.json` and `mcp/components/*.json`: upper-layer MCP composition and runtime injection.
- `templates/provision/`: `aicoding provision` 编译期内嵌的最小 SDD 文档骨架单一来源。
- `templates/kit/`: `aicoding kit init` 编译期内嵌的 manifest、workspec 与外部边界卡单一来源。
<!-- AICODING:REPOSITORY_MAP:END -->
