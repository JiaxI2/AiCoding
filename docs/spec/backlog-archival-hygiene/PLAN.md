---
id: backlog-archival-hygiene
status: approved
approvedTree: "883dff773fd0bf97afb430a35a593d2607c2ffbf"
scope:
  - internal/repohealth/**
  - internal/cli/catalog_test.go
  - config/schemas/cli-report.schema.json
  - docs/COMMANDS.md
  - docs/operations/testing/REPORT_SCHEMA.md
  - docs/architecture/POWERSHELL_BOUNDARY.md
  - tools/specialty/verify-agent-dev-kit-plan-mode.ps1
  - tools/specialty/verify-codex-kit.ps1
  - tools/specialty/hooks/aef/plan-mode-gate.ps1
  - tools/specialty/hooks/aef/spec-artifact-gate.ps1
  - docs/todolist/0029-backlog-archival-hygiene.md
  - docs/todolist/done/0029-backlog-archival-hygiene.md
  - CHANGELOG.md
gates:
  - profile: smoke
---

# backlog 归档与 PowerShell 退役窗口计划

## 目标

在已完成 todo 归档后，为 `doctor pwsh` 的既有只读退役计数增加逐候选
`retirementTrigger` 与 `unspecified` 计数。标记只来自脚本头部
`# RETIRE-AFTER: <condition>`；缺失仅报告，不形成门禁。

## 不变量

- `internal/todolist` 的顶层、非递归读取行为不改。
- `remainingScripts=20 / thinShells=2 / deprecated=2` 的既有顶层计数语义不改。
- 不新增 PowerShell 脚本、不把脚本重写为 Go、不删除兼容壳。
- `doctor pwsh` 保持同一 typed HelpForm 与命令语法；新增 data 字段在 CLI report schema、
  COMMANDS 与报告契约文档同步登记。
- `retirementTrigger=unspecified` 与 `unspecified` 计数只可观测，不使命令失败。

## 验证

单测覆盖 marker、有/无 marker、nested 薄壳与稳定排序；真实运行 `doctor pwsh --json`
核对每个候选均有 trigger 或 unspecified，并确认既有三个计数不变。最后运行 governance
layout、DocSync 与 Smoke；验证通过后将 0029 翻 Done 并用 `git mv` 归档自身。
