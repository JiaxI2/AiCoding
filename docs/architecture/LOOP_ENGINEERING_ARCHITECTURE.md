# Loop Engineering 架构（有界迭代工作的裁决面）

Status: Accepted

> 本文定义 Loop Engineering 的职责边界、状态机与数据契约。它**不定义新的控制面**：
> 执行仍归 [核心架构](AICODING_CORE_ARCHITECTURE.md) 的六模块，验证证据归
> [ADR 0007](../decisions/0007-validation-evidence.md)，扩展路径见
> [EXTENSION_ADAPTER_CONTRACT](EXTENSION_ADAPTER_CONTRACT.md)。与契约文档冲突时以其为准。

## 本篇回答的问题

- Loop 到底新增了什么？为什么不是"再加一个执行器"？
- 一次迭代从哪里出发、在哪里停、谁来判？
- Token 和上下文压力如何进入停止决策？
- 出问题时如何定位、如何回滚？

---

## 1. 结论

**Loop Engineering 是有界迭代工作的裁决者（adjudicator），不是执行器（executor）。**

```text
validationevidence:  hook 不跑测试，只查 Receipt
loop engineering:    AiCoding 不跑循环，只裁决下一步
```

把 loop 拆成四段，只有一段是缺的：

| 环节 | 归属 | 状态 |
|---|---|---|
| observe 当前状态 | `validationevidence` | 已有 |
| act 执行一次尝试 | **Agent 自己** | 不属于 AiCoding |
| verify 门禁 | `testengine` + `validationevidence` | 已有 |
| **decide 继续/停止** | `internal/loopkit` | **唯一新增** |

因此本 Kit 只新增一个 Primitive：**转移决策函数 `Decide`**。
其余全部是既有 Primitive 的组合。

**明确不做**（每一条都对应一个已被拒绝的第二控制面）：

```text
不做调度器            Taskfile / Git hook / CI 已存在
不做 verifier 模型     确定性 Receipt 强于概率判断
不做 work run          让 AiCoding 转循环即第二控制面
不做 harness 自改      违反"不隐式修改无关状态"
不做 per-day 配额      属账单域，非仓库工具域
```

---

## 2. 为什么裁决者比执行器更强

通用 loop engineering 的共识是"验证者 ≠ 执行者"，但通用方案只能做到
**换一个模型来判**——那仍是概率判断，业界称之为 verification burden。

AiCoding 已有一个确定性、内容绑定、可审计的验证者：

```text
通用方案：   actor 模型  ──▶  verifier 模型      概率 ▶ 概率
AiCoding：   Agent      ──▶  validationevidence  概率 ▶ 确定
                              │
                              ├─ 绑定 Git Tree OID
                              ├─ 不可变 Receipt + 完整性校验
                              └─ resultsDigest 逐用例审计
```

`validation check` 在 ~200ms 内给出的不是意见，是证明。
**代价：零 token、零模型调用。**

---

## 3. 系统位置

```text
             ┌──────────────┐
             │   todolist   │  规划了什么（已有，markdown 只读）
             └──────┬───────┘
                    ▼
        ┌───────────────────────┐
        │       WorkSpec        │  一项工作的边界契约（新增）
        │  trigger·stop·authority│
        └───────────┬───────────┘
                    ▼
        ┌───────────────────────┐        ┌────────────────────┐
        │      work next        │◀───────│ validationevidence │ 已有
        │   Decide（纯函数）      │        └────────────────────┘
        └───────────┬───────────┘                  ▲
                    ▼                              │
        ┌───────────────────────┐        ┌────────────────────┐
        │   Agent 执行一次尝试    │───────▶│     testengine     │ 已有
        │   （AiCoding 不参与）   │        └────────────────────┘
        └───────────┬───────────┘
                    ▼
        ┌───────────────────────┐
        │      work record      │  仅追加（新增）
        └───────────┬───────────┘
                    ▼
        .aicoding/state/work/<id>/attempts.jsonl
                    │
                    └────────▶ 回到 work next

   旁路（不参与迭代，但定义边界）：
        lifecycle   声明式收敛（目标已知）
        runner      单次任务集执行（无迭代语义）
```

### 与 lifecycle 的边界判据

> **目标状态能被声明的，用 lifecycle；只能被验证的，才用 loop。**

`lifecycle` 知道自己要收敛到什么（manifest 的 `requiredPaths`、`state`）；
loop 不知道目标长什么样，只能通过门禁经验性地发现自己到了。

没有这条判据，loop 会缓慢侵蚀 lifecycle 的职责。

---

## 4. 状态机

```text
                    ┌─────────────┐
                    │   出发前置   │  spec digest / trigger 授权 /
                    │  （fail-closed）│  预算余量 / 上轮无越界 / 无挂起 checkpoint
                    └──────┬──────┘
                           │ 全部满足
                           ▼
                    ┌─────────────┐
              ┌────▶│   Decide    │  纯函数 · 零 IO · 确定性
              │     └──────┬──────┘
              │            │
              │   ┌────────┼────────────────────┬──────────────┐
              │   ▼        ▼                    ▼              ▼
              │ continue  checkpoint     stop-satisfied   stop-budget
              │   │        │             stop-stalled     stop-violation
              │   │        │                    │              │
              │   ▼        ▼                    └──────┬───────┘
              │ Agent    人工裁决                       ▼
              │ 执行       │                        终止（具名）
              │   │        │
              │   ▼        │
              │ record ◀───┘
              └───┘
```

### 五个具名终止态

| 终止态 | 触发条件 | 语义 | 后续处置 |
|---|---|---|---|
| `stop-satisfied` | 所有 requiredGates 有有效 Receipt | 成功 | 可提交/推送 |
| `stop-budget` | attempts / elapsed / tokens 任一耗尽 | 投降 | 加预算重来 |
| `stop-stalled` | 连续 N 次无进展 | 投降 | **换思路**，加预算无用 |
| `stop-violation` | 写入越界 / spec 被篡改 / 授权不符 | 安全阻断 | 人工排查 |
| `checkpoint` | 命中人工检查点或上下文压力 | 移交 | 人工裁决后恢复 |

投降拆成 budget 与 stalled 两种，因为**处置方式不同**：
预算耗尽是"加预算再来"，失速是"方法不对"。合并会误导使用者。

---

## 5. 出发（Trigger）

三种发起方式，与停止规则正交：

```text
explicit        人或 Agent 显式调用           默认
scheduled       外部调度器（Task/CI/cron）    运维回归、夜间维护
agent-proposed  Agent 主动提议               最危险，默认关闭
```

**AiCoding 不拥有调度器。** `trigger` 字段声明**谁被允许发起**，不实现发起：

```text
CI cron / GitHub Action  ─┐
Taskfile                 ─┼─▶  work next  ─▶  读退出码
外部 Agent 运行时          ─┘
```

`work next` 会拒绝一个 `trigger: explicit` 的 spec 被自动化路径调用——
这是**授权检查**，不是调度实现。

---

## 6. 停止（Stop）

### 6.1 Stop 是规则集合，不是枚举

全部求值，先命中者决定终止态：

```json
"stop": {
  "maxAttempts": 5,
  "maxElapsedSeconds": 1800,
  "maxTotalTokens": 2000000,
  "stallThreshold": 2,
  "contextPressureThreshold": 80,
  "requiredGates": [{ "profile": "release" }]
}
```

这样"最多 5 次、或半小时、或 200 万 token、或门禁绿"是一个自然的合取，
不需要发明第五种 control mode。

### 6.2 失速检测复用 Tree OID（零新增代码）

通用方案靠"最近几步输出相似度"——启发式、不可靠、要花 token。
AiCoding 已经有内容身份：

```text
attempt N   结束  subjectTreeOID = a1b2c3…
attempt N+1 结束  subjectTreeOID = a1b2c3…   ← 完全相同
                  ⇒ 这一轮什么都没改变
                  ⇒ stallCount++

stallCount >= stallThreshold  ⇒  stop-stalled
```

同一个 Tree OID，在 [ADR 0007](../decisions/0007-validation-evidence.md) 里判断
"能否复用 Receipt"，在这里判断"有没有产生变化"。
**同一 Primitive，两个用途，零新增代码。**

两级信号：

```text
tree 未变                          ⇒ 强失速
tree 变了但门禁结论未变              ⇒ 弱失速（计入 stallCount，阈值可分设）
```

### 6.3 越界检测是检测式，不是预防式

复用 `gitx.StatusSnapshot`（单次调用，~200ms）：

```text
实际变更集  ∖  authority.writeScope.allow  ≠ ∅   ⇒  stop-violation
```

> **必须明确：AiCoding 无法阻止 Agent 写盘。**
> 它阻断的是**下一步**，不是已经发生的写入。
> 把 `writeScope` 称作"权限模型"会给使用者虚假的安全感。

---

## 7. Token 与上下文

### 7.1 AiCoding 不测量 token，它记账

记账器已存在，**新增 Primitive 数量为 0**：

```text
internal/report/tokenusage/          已有
  Usage{ input, cached, cacheWrite, output, reasoning, total,
         contextTokens, contextWindow, contextRemaining,
         contextUsedPercent }
  ParseJSONL + Collector （幂等，重复事件不重复计数）

CLI: aicoding codex usage parse|run
```

`Attempt` 直接内嵌 `tokenusage.Usage` 作为子报告，**不新定义 token 类型**。

### 7.2 三层预算

| 层级 | 载体 | 用途 |
|---|---|---|
| per-attempt | `Usage.ContextUsedPercent` | 单次尝试是否逼近上下文上限 |
| per-work | `Σ Usage.TotalTokens` | 累计消耗 → `stop-budget` |
| per-day | **不做** | 属账单域，做了即引入配额存储与跨仓聚合 |

### 7.3 上下文压力是一等停止信号

```text
Usage.ContextUsedPercent > contextPressureThreshold（默认 80）
  ⇒ checkpoint，reason = "context pressure"
```

上下文接近满时输出质量断崖下降，继续跑是纯烧钱。
这是唯一一处 AiCoding 主动建议"停下来重新组织"，而非等预算耗尽。

### 7.4 Token 效率的结构性来源

不是压缩提示词，是**把判断从模型移到确定性 Primitive**：

| 通用 loop 的开销 | 本架构 | 省掉 |
|---|---|---|
| verifier 子 Agent 二次意见 | 查 Receipt | 整个验证 Agent |
| 每轮重述项目上下文 | Skill + 既有文档 | 每轮 system prompt |
| 重跑已通过的验证 | Receipt 复用 | 整轮等待与重试 |
| 模型判断"有无进展" | Tree OID 比较 | 每轮一次判断调用 |
| 模型记忆历史尝试 | `attempts.jsonl` | 历史全部退出上下文 |

---

## 8. 数据契约

### 8.1 存储布局

```text
.aicoding/state/work/<work-id>/
├── state.json      当前 attempt、预算消耗、最后决策、spec digest
└── attempts.jsonl  追加式不可变尝试记录（审计轨迹）
```

沿用既有 domain-owned state 约定（对齐 `config/kits/*.json` 的 `state.root`）。
**不进 git-common-dir**——work 状态是 worktree 局部的工作会话，
与跨 worktree 共享的 Receipt 不同。

### 8.2 决策函数签名

```go
func Decide(
    spec    Spec,           // 契约
    history []Attempt,      // 含 subjectTreeOID 与 tokenusage.Usage
    gates   []GateStatus,   // 来自 validationevidence，由 CLI 层注入
    now     time.Time,      // 注入而非读时钟，保证可测
) (Decision, error)
```

**四个入参全部由调用方注入，函数内零 IO。**
全部 Git/证据查询集中在 CLI 层一次完成；决策本身可被表驱动测试穷举。

### 8.3 门禁只持引用，绝不重定义

```go
type GateRef struct {
    Profile            string `json:"profile"`
    ValidationIdentity string `json:"validationIdentity"`
    ReceiptID          string `json:"receiptID"`
}
```

> **全仓只允许存在一个验证证据权威。**
> `loopkit` 不得定义自己的 `Receipt` 类型——否则 loop 可以自称门禁通过，
> 而证据体系毫不知情。回归门禁：全仓 `type Receipt` 只应有 `validationevidence` 一处。

---

## 9. 对外 API

```text
aicoding work validate --file <spec.json> --json                       只读
aicoding work next     --file <spec.json> --json                       只读
aicoding work status   --file <spec.json> --json                       只读
aicoding work record   --file <spec.json> --attempt <a.json> --json    仅追加
```

`work next` 返回：

```json
{
  "decision": "continue",
  "attempt": 3,
  "reason": "required gate release is not satisfied for the current tree",
  "allowedScope": { "allow": ["internal/**"], "deny": ["config/**"] },
  "requiredGates": [{ "profile": "release", "satisfied": false }],
  "budget": { "attemptsUsed": 2, "maxAttempts": 5,
              "elapsedSeconds": 412, "totalTokens": 318204 },
  "stall": { "count": 0, "threshold": 2 },
  "checkpoint": null,
  "requiredAction": "aicoding test --profile Release --reuse off --json"
}
```

消费者：

| 消费者 | 关心 | 用哪条 |
|---|---|---|
| Agent | 能否继续、可动什么、什么算完成 | `work next` |
| 人 | 在干什么、为什么停 | `work status` |
| CI | 有界执行、超预算即失败 | `work next` + 退出码 |
| Hook | 有资格提交吗 | `work status`（毫秒级） |

---

## 10. 可靠性与安全

| 面 | 机制 | 失败时行为 |
|---|---|---|
| spec 篡改 | `state.json` 记录 spec digest，每次比对 | `stop-violation` |
| 状态损坏 | `attempts.jsonl` 追加不可变；解析失败即拒绝 | fail-closed |
| 越界写入 | `gitx.StatusSnapshot` 差集 | `stop-violation`（检测式，见 §6.3） |
| 门禁伪造 | 只认 `validationevidence` Receipt | 无法伪造 |
| 无限循环 | **不实现 `work run`** | 风险类别结构性不存在 |
| 预算失控 | 三条独立预算，任一命中即停 | `stop-budget` |
| 并发 | 单 work 单 worktree；跨 worktree 天然隔离 | 无共享写 |

**安全默认值**：`agent-proposed` 触发默认关闭；`writeScope` 无声明即空集（拒绝一切）；
无 `requiredGates` 的 spec 拒绝进入 `continue`。

---

## 11. 演进边界

以下能力**已被评估并明确推迟**，重新提出必须走 ADR 并给出至少两个真实消费者：

| 能力 | 推迟原因 |
|---|---|
| `work run` 内建循环 | 第二控制面入口 |
| `work prepare` / `work step` 工作区管理 | 现有 worktree 已覆盖 |
| Profile 继承式 loop 组合 | 无真实需求 |
| harness 自动调优（hill climbing） | 违反"不隐式修改无关状态" |
| Workflow DSL | 见 [06-plugin-sdk](06-plugin-sdk.md) §7 |
| per-day token 配额 | 属账单域 |

---

## 12. 参考

外部实践（本架构的分歧点见 §1「明确不做」与 §2）：

- [Loop Engineering — Addy Osmani](https://addyosmani.com/blog/loop-engineering/)
- [The Art of Loop Engineering — LangChain](https://www.langchain.com/blog/the-art-of-loop-engineering)
- [Stop Hand-Holding Your Coding Agent — arXiv 2607.00038](https://arxiv.org/abs/2607.00038)
