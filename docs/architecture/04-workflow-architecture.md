# 04 工作流架构：Plan → Execute → Verify → Release（Workflow Architecture）

Status: Derived View（派生视图）

> 本文不定义新契约；报告契约见
> [CLI、验证与测试报告 Schema](../operations/testing/REPORT_SCHEMA.md) 与
> [契约冻结与获取/激活边界](FREEZE_AND_ACQUISITION_BOUNDARY.md)，冲突时以其为准。

## 本篇回答的问题

- 计划、执行、验证、发布如何用命令闭环？
- 如何验证结果（人和 Agent 都按什么标准判断"做成了"）？
- 多 Agent 如何协同？

## 1. 工作流骨架与命令的对应

| 阶段 | 它做什么 | 命令 / 载体 |
|---|---|---|
| Plan（计划） | 先拿到"将要发生什么"的可审计清单，再请求批准 | `lifecycle plan --action <A> --scope <S>`（机器计划，带 planDigest）；计划模式（Plan Mode）specialty 工具与 `docs/spec/` 工件（人机决策） |
| Execute（执行） | 只执行批准过的意图 | `lifecycle install\|update\|uninstall\|rollback` |
| Verify（验证） | 三分工：环境体检 / 静态契约 / 官方测试 | `doctor --all` / `verify --profile` / `test --profile` |
| Release（发布） | 结构快检 + 发布门禁，不自动建 tag | `release verify` / `release gate`；tag 政策见 [Tagging Policy](../governance/TAGGING_POLICY.md) |

## 2. 意图与执行分离（为什么先 plan 后 apply）

`plan` 与 apply 共用同一领域路径（plan = action + dryRun）：预览计算的就是执行将做的，
不存在独立的预览实现，所以**预览与执行不会漂移**（对照 Git 的 index：`git add`
写入的就是 commit 将使用的 tree 条目）。Agent 的标准动作序列：

```text
lifecycle plan …   → 把 planDigest 和清单给人看/给上层 Skill 审
（批准）
lifecycle install …→ 执行；报告带同一 planDigest，可对账"执行的就是批准的"
```

## 3. 如何验证结果（统一判据）

所有命令输出统一 JSON 报告（`report.Result`）：

| 字段 | 含义 | 判读规则 |
|---|---|---|
| `ok` | 本次是否成功 | `true` 才算过 |
| `errorKind` | `usage` / `execution` / `validation` | 参数错 / 执行失败 / 检查不过 |
| 退出码 | `0` / `1` / `2` | 与 `ok`/`errorKind` 对应，脚本可直接判断 |
| digest 三元组 | `catalogDigest` / `inputDigest` / `planDigest` | 回答"用什么契约、对什么事实、执行了什么意图" |

配套证据：`test --profile` 把 `report.md` + `summary.json` + `results.json` 写进
`test-results/`；`fresh-clone --profile` 证明干净克隆可复现；发布前 `release gate`
必须绿。**Agent 判读只看字段，不解析人类文本**；"做完了"的声明必须能指出对应的
绿色报告，这就是"先验证后声明完成"纪律的机器形态。

## 4. 多 Agent 如何协同

现状（顺序协同，已可用）：

1. **同一契约**：任何 Agent 产出的动作都走同一 CLI、同一 JSON 判据、同一门禁——
   协作者不需要约定私有协议。
2. **先计划后执行**：写操作必须能出示 plan 清单与 planDigest，评审者（人或另一个
   Agent）审的是计划而不是事后描述。
3. **证据链**：每次运行的报告可对账（digest 三元组），接手的 Agent 从
   `test latest`、`lifecycle status` 就能还原现场，不依赖上一个会话的口头记忆。
4. **知识唯一**：同名 Skill 审计保证两个 Agent 不会拿着互相矛盾的两份流程知识。

预留（真实并发写场景出现才建，见 [07](07-roadmap.md) §2）：expected-digest 并发守卫
——写操作携带"我看到的事实 digest"，事实已变则拒绝执行（类比 `git push` 的
non-fast-forward 拒绝）。

## 5. 工作流知识的承载分工

流程知识在 SKILL.md（如升级列车的 preflight → plan → confirm → apply → verify），
强制在 hook/门禁（[05](05-governance.md)），执行在 CLI（[01](01-system-architecture.md) §6）。
Workflow 层自己**不留代码**——它是约定，不是组件。
