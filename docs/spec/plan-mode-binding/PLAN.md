---
id: plan-mode-binding
status: approved
scope:
  - internal/plan/**
  - internal/cli/**
  - internal/gitx/**
  - config/schemas/plan-spec.schema.json
  - docs/architecture/PLAN_MODE_ARCHITECTURE.md
  - docs/decisions/0002-aicoding-agent-dev-kit-plan-mode.md
  - docs/decisions/0009-plan-mode-rework.md
  - docs/COMMANDS.md
  - .githooks/pre-commit
  - CHANGELOG.md
  - docs/todolist/0006-plan-mode-binding.md
approvedTree: "7d09305d98143e5830808ebf81be7fe89d08ac12"
decision: ""
gates:
  - profile: full
---

# Plan Mode 内容绑定

实现 `plan approve`、基于 Git Tree 的漂移裁决、approved plan scope 覆盖检查，
并将 pre-commit 的 Plan Mode 门禁由 warning 升为 enforce。

完成判据以 TODO 0006 的五条端到端用例、Full profile、依赖治理和 ADR 门禁为准。
