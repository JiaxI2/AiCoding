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
- `schema-closure-exclusions.json` / `schemas/schema-closure-exclusions.schema.json`: 配置 schema
  完备性排除表；精确文件排除不得使用通配，目录排除只允许后缀 `/**`，不存在的排除会失败。
- `internal/docsync/policy_schema.go` 是配置/schema 双向闭合的唯一 binding authority：当前
  35/35 个非 schema JSON 配置逐项验证，29/29 个 schema 由 binding 或 standalone 登记反向引用。
  `config/schemas/**` 只从“配置实例”枚举中排除，仍必须通过反向引用检查，不能成为幽灵 schema。
- `pwsh-budget.json` / `schemas/pwsh-budget.schema.json`: PWSH-002 的顶层 PowerShell 脚本
  基线历史；首条来自已提交的 `doctor pwsh` 原始证据，后续条目只能是前一集合的严格子集。
- `validation-policy.json`: pre-push Context Gate 的远端 ref、必需验证 profile、快进与删除策略。
- `impact-policy.json`: 影响规则的单一文件；`raceScope.packages` 登记 Full/Release 共用的 race 包集合，GO-007 机器阻断并发包漏登；`changeVerify` 以与 Plan Mode 同构的路径 pattern 将变更确定性映射到 Smoke/Full，未命中路径保守选择 Full。
- `mcp-registry.json` and `mcp/components/*.json`: upper-layer MCP composition and runtime injection.
- `templates/provision/`: `aicoding provision` 编译期内嵌的最小 SDD 文档骨架单一来源。
- `templates/kit/`: `aicoding kit init` 编译期内嵌的 manifest、workspec 与外部边界卡单一来源。
- `templates/skill/`: `aicoding skill init` 编译期内嵌的外部 Skill 草稿模板；只允许输出到 AiCoding 仓库外的可写 Codex-Skills worktree。
- `templates/mcp/`: `aicoding mcp init` 编译期内嵌的 component manifest 模板；只给出 disabled registry entry 建议，不自动登记。
- 不提供 `templates/hook/` 或 `hook init`；Hook 只在既有权威实现中维护。
<!-- AICODING:REPOSITORY_MAP:END -->
