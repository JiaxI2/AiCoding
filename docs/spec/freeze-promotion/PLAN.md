---
id: freeze-promotion
status: draft
approvedTree: ""
scope:
  - docs/architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md
  - docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md
  - docs/architecture/PLAN_MODE_ARCHITECTURE.md
  - docs/architecture/README.md
  - internal/testengine/engine.go
  - internal/testengine/freeze.go
  - internal/testengine/freeze_test.go
  - internal/cli/catalog.go
  - docs/todolist/0030-freeze-promotion.md
  - docs/todolist/done/0030-freeze-promotion.md
  - CHANGELOG.md
gates:
  - profile: full
---

# 冻结面晋升计划

## 目标

严格按 TODO 0030 的清单，将已经过实测的 Loop Engineering、Plan Mode、
Validation Evidence 核心 Receipt 契约与 pinned reference `source` 语义纳入冻结边界，
并以 FREEZE-004..007 四条最小静态断言守住机器可判的表面。

## 不变量

- `COMPOUNDING_KNOWLEDGE.md` 保持 Draft，capability registry schema 不晋升。
- 两份架构正文不重写，只提升 Status 并增加统一的解冻条件提示。
- FREEZE-004 只断言 `work run|prepare|step` 不进入 typed catalog。
- FREEZE-005 只断言 `transition.Decide` 保持四参数注入。
- FREEZE-006 只断言 `validationevidence.Fingerprint` 字段集合不变。
- FREEZE-007 只断言 Kit manifest 的 `source` 保持可选，不修改 schema。
- `internal/cli/catalog.go` 仅用于 FREEZE-004 的临时负例，验证后必须逐字还原。

## 验证

先在 clean tree 上批准并绑定 `approvedTree`。单测覆盖四条断言及 registry 登记；
再分别临时加入 `work run`、修改 `Decide` 签名，真实执行对应静态门禁并确认红后还原。
最后运行 DocSync 与 Full，并在通过后把 TODO 0030 翻 Done、用 `git mv` 自归档。
