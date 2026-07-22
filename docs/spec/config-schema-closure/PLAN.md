---
id: config-schema-closure
status: draft
approvedTree: ""
scope:
  - config/**
  - internal/docsync/**
  - internal/governance/**
  - config/README.md
  - docs/architecture/DOC_SYNC_PLUS_SPEC.md
  - docs/COMMANDS.md
  - docs/operations/VALIDATION_EVIDENCE_BUDGET.md
  - docs/operations/testing/GLOBAL_TEST_CASES.md
  - docs/operations/evidence/config-schema-closure-negative-matrix.md
  - docs/todolist/0035-config-schema-closure.md
  - docs/todolist/done/0035-config-schema-closure.md
  - CHANGELOG.md
gates:
  - profile: full
  - profile: release
---

# Config schema 闭合完备化计划

## 目标

把 `config/` 中的 checked-in JSON 配置收敛到一个可枚举、可反向验证的 schema 闭合面：
每个配置必须由 `internal/docsync/policy_schema.go` 的 binding table 验证，或由有理由的精确
排除覆盖；每个 schema 也必须被 binding 或 standalone 登记反向引用。既有消费方的严格解码、
registry coverage 与 freeze 断言继续并存。

## 已裁决设计

- 配置权威仍是 Git + JSON + checked-in schema；Receipt store 仍是内容寻址文件存储。
- `governance dependencies` 复用一次 repository inventory，不建立第二套 walker 或配置加载器。
- 新增一个 strict exclusion 配置及其 schema。精确文件排除不得含通配；目录排除只允许
  `path/**`，其中 `config/schemas/**` 用于把 schema 文件交给反向引用闭合检查。
- exclusion 中不存在的文件/目录、未登记配置、未登记 schema、缺失 binding schema 与
  schema 语义错误全部 fail-closed 并返回具体路径。
- 动态键对象可使用 schema-valued `additionalProperties`，但必须用 `$comment` 说明这是
  数据模型需要的开放键，而不是绕过 strict schema。

## 实施边界

- 不引入数据库或第三方 JSON Schema 库，不改任何消费方的配置解码逻辑。
- 不移动、合并、重命名既有配置，不建立集中式加载框架，不新增 CLI 或治理领域。
- 不修改 `CodingKit/agents/skills`、TODO 0019、Receipt 类型或 `--reuse` 默认值。
- 内置 validator 只做支持本轮 schema 所需的最小变更；已确认 `$comment` 会自然忽略。
- 当前审计目标是 35/35 configuration bindings（34 个既有配置加 exclusion 配置）和
  29/29 schema references（19 个既有 schema 加 10 个新 schema）；实施中若文件实测变化，
  以重新枚举值为准并在 TODO 留证。

## 验证

单测覆盖 strict binding、配置/schema 双向完备性、目录通配唯一性与幽灵排除。真实工作树
依次注入非法字段、rogue 配置、幽灵排除、exclusion 非法字段及缺失既有 schema，保存原始
非零输出并逐项还原。最终运行 `docsync all`、`governance dependencies`、`plan verify`、
`todolist`、Full 和 Release，并通过正常 pre-push Receipt 门禁推送 main。
