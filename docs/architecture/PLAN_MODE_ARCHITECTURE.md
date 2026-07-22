# Plan Mode 架构（批准内容树与范围漂移）

Status: Accepted and Frozen

> 解冻必须走 ADR，并同时满足现实问题、稳定变化点、至少两个真实消费者三项条件。

> 本文是 Plan Mode 的唯一架构权威。决策依据见
> [ADR 0009](../decisions/0009-plan-mode-rework.md)；初代 overlay 记录见已被取代的
> [ADR 0002](../decisions/0002-aicoding-agent-dev-kit-plan-mode.md)。

## 1. 结论

Plan Mode 批准的不是一段说明文字，而是一棵 Git 内容树及其允许变化的 scope。

```text
PLAN.md draft/needs-decision
        │ clean worktree + plan approve
        ▼
status: approved + approvedTree: HEAD^{tree}
        │
        ├─ plan check --staged：敏感路径必须被 approved scope 覆盖
        └─ plan status：approvedTree 与当前 HEAD tree 的路径差异分类
```

plan 不执行实现、不运行测试、不签发 Receipt，也不调度循环。唯一写命令是
`plan approve`，且只改一个 PLAN.md 的两个 frontmatter 字段。

## 2. 系统位置

| 上下游 | Plan Mode 消费什么 | Plan Mode 输出什么 |
|---|---|---|
| Git / `gitx` | clean status、Tree OID、tree diff、staged paths | 不拥有 Git 状态 |
| `todolist` | 阶段优先级与人工 Done | 不改 TODO 状态 |
| `validationevidence` | 当前 Tree 的精确 Receipt 判定 | GateRef 与完成建议 |
| `loopkit` | 无直接依赖 | 可消费批准 scope，但不由 plan 调度 |
| pre-commit | staged 敏感路径 | covered / uncovered 裁决与退出码 |

CLI 是编排层：它调用 `gitx` 和 `validationevidence`，再把值对象交给 `internal/plan`。
`internal/plan` 不反向导入 Git 控制面，不定义 Receipt。

## 3. 边界判据

- “是否需要先明确意图与范围”属于 plan。
- “下一次尝试继续还是停止”属于 loop。
- “当前内容是否通过某 profile”属于 validationevidence。
- “这项工作在队列中是否 Done”属于 todolist。

pre-commit 是提交内容的强制检查点；Agent hook 只保留兼容提示，不是安全边界。所谓 scope
权限是检测式的：它能拒绝下一次受治理提交，不能撤销或阻止已经发生的写盘。

## 4. 数据契约

```yaml
id: plan-mode-binding
status: approved
scope:
  - internal/plan/**
approvedTree: "<40-or-64-hex-tree-oid>"
gates:
  - profile: full
```

`draft` 与 `needs-decision` 可为空树；`approved` 与 `implemented` 必须携带合法 Tree OID。
`OPTIONS.md` 存在时必须有 `DECISION.md`。frontmatter 与目录 ID、scope 和 gate 一起由
`plan verify` fail-closed 校验。

status 投影包含：`approvedTree/currentTree/changed/drift/outOfScope/exempt/scopeCovered`，
以及只引用既有证据的 `GateRef`。结果按 plan ID 和路径排序，不含时钟或 commit 元数据。

## 5. 可靠性与安全

| 风险 | 机制 | 失败行为 |
|---|---|---|
| 在漂移内容上批准 | approve 前单次 clean status | 拒绝，不写文件 |
| PLAN 字段缺失或伪造树 | strict parser + Tree OID 形状校验 | verify/check fail-closed |
| 敏感 staged 路径无批准 | approved scope coverage | pre-commit 非零退出 |
| 实现越界 | tree diff 与 scope/exempt 差集 | status 警告并列出路径 |
| 验证事实重复 | 只调用 validationevidence exact check | Receipt miss，不自签通过 |
| metadata-only amend/rebase | 绑定 Tree 而非 commit | 内容不变即无伪漂移 |

Hook 永不构建或测试；缺少 Receipt 只影响完成建议，不让 pre-commit 偷跑 Full/Release。

## 6. 演进边界

首期不做跨 plan 依赖图、模板生成器、自动 `implemented`、自动修复越界或 OS 级权限控制。
PowerShell overlay 仅作为兼容薄壳存在，按 ADR 0009 的 release 节奏退役；任何新语义必须先落
Go 领域与 CLI，脚本不得成为平行事实源。
