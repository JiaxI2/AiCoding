---
id: pwsh-budget-ratchet
status: draft
approvedTree: ""
scope:
  - internal/repohealth/**
  - internal/cli/cli_test.go
  - config/pwsh-budget.json
  - config/schemas/pwsh-budget.schema.json
  - config/schemas/cli-report.schema.json
  - config/README.md
  - docs/architecture/POWERSHELL_BOUNDARY.md
  - docs/COMMANDS.md
  - docs/operations/evidence/pwsh-budget-baseline-f56c17e.json
  - docs/operations/testing/GLOBAL_TEST_CASES.md
  - docs/operations/testing/REPORT_SCHEMA.md
  - docs/todolist/0034-pwsh-budget-ratchet.md
  - docs/todolist/done/0034-pwsh-budget-ratchet.md
  - CHANGELOG.md
gates:
  - profile: full
---

# PWSH-002 PowerShell 棘轮计划

## 目标

复用既有 `doctor pwsh-budget` / PWSH-002，把 TODO 0033 落地后真跑得到的顶层
PowerShell 脚本集合固定为只降不升的基线。PWSH-001 继续只报告，不新增命令、领域或
PowerShell 保留类别。

## 已批准范围

- `config/pwsh-budget.json` 保存按提交与原始 doctor 输出可追溯的基线历史；当前条目必须
  等于 `f56c17e1b8be8723fc4f884cdf537f2f9cd959cd` 上的实测集合。
- PWSH-002 同时执行既有调用点预算与新的脚本集合棘轮；缺配置、未知字段、基线历史上升、
  当前集合漂移或既有退休候选变为 unspecified 均 fail-closed。
- 删除脚本时必须在同一提交追加严格子集基线；新增或替换脚本都不能靠保持总数绕过。
- CLI JSON 只为既有 PWSH-002 data 增加 ratchet 证据，不修改 `report.Result` 或 PWSH-001
  字段集合。

## 不变量

- `doctor pwsh` 对 unspecified 的退出码仍为 0；`docs/spec/backlog-archival-hygiene/PLAN.md`
  第 38 行契约不变。
- 不对 deprecated、thinShell 增加门禁，不重写或删除本项之外的 PowerShell 脚本。
- `remainingScripts` 具体数值不是长期契约；“当前集合等于最后基线，后续基线只能为严格
  子集”才是契约。
- config 与 schema 均拒绝未知字段；原始 baseline 输出随提交入仓。

## 验证

单测覆盖配置缺失/损坏、基线集合匹配、缺标记、带标记新增、删除并下调、上调/替换拒绝。
真实负例分别新增无标记与带合法 `RETIRE-AFTER` 的顶层 `.ps1`，确认命令非零并指出路径；
临时上调 baseline 同样非零。全部还原后运行 PWSH-001/PWSH-002、DocSync、governance、
Full，并在 TODO 0034 记录原始输出与 summary。
