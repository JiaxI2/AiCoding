# Graph First（图优先 · 网状思维）

Status: Accepted and Frozen

> 设计顺序：**Graph → Primitive → Workflow → Implementation。** 先理解网络、再实现节点；
> 先复用/组合已有 Graph，再考虑新增节点。系统能力可以持续增长，但 **Graph 复杂度的增长
> 必须尽可能趋近于零**。本文是设计法的最上层，向下连接（不重复）已有文档：

```text
Graph First（本文，网络形状 + 设计顺序 + 收敛纪律）
  → PRIMITIVE_CONSTITUTION（节点质量：12 条约束 + 评审 Checklist）
  → AICODING_CORE_ARCHITECTURE（冻结的内核节点 + 拒绝清单）
  → EXTENSION_ADAPTER_CONTRACT（新节点如何接入 = 边契约）
  → config/dependency-governance.json（机器强制的边）
```

## 1. AiCoding 的真实 Graph（不是抽象，是当前强制的边）

Graph = Node + Edge。本仓库的**边由 `config/dependency-governance.json` 的
`goPackageBoundaries` 机器强制**（`aicoding governance dependencies --json`），依赖只向下。

**节点分层（按被依赖度 / 复用度）**：

| 层 | 节点 | 角色 | 稳定性 |
|---|---|---|---|
| 事实层 | `internal/gitx` | 唯一 git 进程边界，**零 internal 依赖** | 冻结 |
| 原语层（中心节点） | `internal/registry`（快照+digest）、`internal/runner`（ExecutionPlan）、`internal/report`（证据信封）、`internal/platform`（根/路径） | 最高复用、最多入边 | 冻结 |
| 编排枢纽 | `internal/lifecycle`（adapter catalog，用 runner 组合领域） | 把统一动词翻译到领域 | 冻结契约 |
| 领域层（互相隔离的兄弟节点） | `internal/kit`、`internal/mcpcontrol`、`internal/repocontext` | 各自 owned state | 按扩展路径新增 |
| 聚合/入口 | `internal/repohealth`（doctor/verify 聚合）、`internal/testengine`（唯一测试引擎）、`internal/cli`（命令目录，入口） | 组合下层，不被下层依赖 | 冻结契约 |
| 外围叶子（读原语） | `cache`、`tagpolicy`、`todolist`、`cstyle`、`docsync`、`reuse`、`pwshregex`、`releasegate`、`bootstrap` | 单一职责叶子 | 自由增减 |

**关键边规则（强制）**：

1. **依赖只向下**：叶子/领域可用原语层；原语层永不 import 领域或入口（否则 blast radius 失控）。
2. **gitx 零 internal 依赖**：事实层不认识任何上层。
3. **领域互相隔离**：`kit`/`mcpcontrol`/`repocontext` 互不 import（无兄弟耦合）。
4. **数据流边（digest 链）**：事实 → `registry` 快照(inputDigest) → `runner` plan(planDigest) → 执行 → `report` 信封。可对账性就是这条边的契约。

## 2. 中心节点为什么必须冻结

中心节点 = 入边最多的节点（`registry`/`runner`/`report`/`platform`/`gitx`）。改动它们的
验证半径是全量、blast radius 最大。所以**新能力永远是"新叶子/新领域节点连接到中心节点"，
而不是改中心节点**。这与 [核心架构](AICODING_CORE_ARCHITECTURE.md) 的冻结条件、拒绝清单同一件事，
只是从网络视角陈述：*优化整个网络，而不是某一条路径。*

## 3. Network Thinking：新需求的动作顺序

来任何需求，先走网络、后写代码（对应哲学 Step 1–7）：

1. **识别节点**：这需求涉及哪些现有节点？
2. **找已有节点**：是否已有 Primitive/领域/Workflow/数据流可复用？
3. **找连接点**：不是"我要新增什么"，而是"应该连到哪里"。优先连入已有 Graph。
4. **数据流**：数据从哪产生→经哪些节点→在哪转换→流向哪？有无重复路径/重复转换/重复状态？
5. **控制流 & 所有权**：谁拥有资源、谁管生命周期？控制权必须唯一。
6. **生命周期**：融入已有生命周期（八动词），不建平行生命周期。
7. **最后才允许新节点**：必须证明不能复用/组合/扩展；新节点须有长期复用价值，并把边接入治理。

## 4. 收敛 Checklist（每次改动后必答）

- Graph 是否更简单 / 更稳定 / 依赖更少 / 状态更少？
- 是否减少了重复路径？是否提高了复用与组合？
- 是否形成了新的重复路径（反而要收敛）？
- 新节点的边是否已接入 `dependency-governance.json` 强制？
- 中心节点是否零改动？
- 还能继续收敛吗？

任一为否 → 继续优化。

## 5. 两个 worked 例子（Graph First 的正反面）

**正例 — 补齐缺失的边（本轮）**：`internal/repocontext`、`internal/todolist` 作为新节点接入
Graph 后，其**边一度未被治理强制**（能被误改成 `repocontext → lifecycle` 的环而无人拦）。
本轮把 `repocontext` 域节点按 `kit`/`mcpcontrol` 同构接入 `goPackageBoundaries`：领域隔离
（互不依赖）、不向上依赖 `cli`/`lifecycle`/`repohealth`/`testengine`。注入
`repocontext → lifecycle` 现在会被 `governance dependencies` 明确拒绝。**新节点加入必须同时
加入边强制**，否则网络结构是"碰巧成立"而非"保证成立"。

**克制例 — 不做过早收敛**：`kit`/`mcp`/`repocontext` 各自实现"期望态→写 owned→收敛"路径
（[ADR 0003](../decisions/0003-repo-context-domain.md)）。这是一条**候选重复路径**，但当前三者
owned 形态不同（venv/junction/文本），强行抽公共节点是过早收敛（也是 Graph smell）。正确做法
是**记录触发条件**——出现第二个"生成文本"领域时，再把 `reconcile` 收敛为共享 Primitive。
收敛要靠证据，不靠预建。

## 6. 最终目标

优秀软件不是节点最多、功能最多、代码最多，而是拥有**最稳定的 Graph、最少的 Node、最清晰
的 Edge、最高的 Reuse 与 Composition、最低的 Complexity**。Graph 是一等公民；Architecture、
Primitive、Workflow、Implementation 都服务于 Graph。任何功能只是 Graph 中的一个 Node——
真正持续优化的是整个网络。
