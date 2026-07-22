---
id: pathpolicy-consolidation
status: approved
approvedTree: "cd05a49e32c019d47e348e426809c160916b70a4"
scope:
  - internal/pathpolicy/**
  - internal/plan/**
  - internal/testengine/**
  - internal/cli/change.go
  - internal/cli/change_test.go
  - internal/validationevidence/**
  - internal/docsync/**
  - internal/governance/**
  - config/dependency-governance.json
  - config/internal-capabilities.json
  - config/schemas/impact-policy.schema.json
  - config/schemas/validation-policy.schema.json
  - config/schemas/docs-sync-policy.schema.json
  - config/schemas/docs-sync-semantic.schema.json
  - config/schemas/tagging-policy.schema.json
  - docs/decisions/0011-pathpolicy-primitive.md
  - docs/decisions/README.md
  - docs/architecture/02-primitive-core.md
  - docs/architecture/DOC_SYNC_PLUS_SPEC.md
  - docs/CAPABILITIES.md
  - README.md
  - CHANGELOG.md
  - docs/todolist/0028-pathpolicy-consolidation.md
gates:
  - profile: full
---

# pathpolicy 解析收敛计划

## 目标

新增只依赖 Go 标准库的 `internal/pathpolicy` Primitive，把 Plan Mode、change impact 与
validation push context 的路径/selector 解析收敛为同一 glob 方言，同时保持三份配置文件、
字段与裁决顺序不变。为六个 policy 配置面建立 6/6 schema 映射并由 DocSync fail-closed 校验。

## 不变量

- `internal/pathpolicy` 公开函数不超过四个且只依赖标准库。
- `plan check` 与 `change verify` 对相同 staged 输入的确定性裁决字节保持不变；只剔除
  `elapsed*` / `duration*` 观测字段。
- `validation-policy.json` 的 exact ref 与 prefix ref 语义、顺序和 fail-closed 行为不变。
- policy 配置不合并、不改字段、不扩展 glob 方言。
- 六面 schema 是 plan、impact、validation、docs-sync policy、docs-sync semantic 与 tagging；
  现有 plan schema 保留，其余五个补齐。
- 新 Primitive 通过 ADR §12 自评、依赖反向禁令、capability 登记与 Full profile。

## 验证

先在固定 baseline worktree 记录 `plan check --staged` 与 `change verify --staged` 的规范化 JSON；
实现后以新二进制对同一 index 重放并做字节比较。再运行 pathpolicy/plan/testengine/
validationevidence 局部测试、6/6 schema 正例、impact 非法字段负例、DocSync/governance、
依赖边界、单一权威 grep 与 Full profile。所有负例验证后恢复工作树。
