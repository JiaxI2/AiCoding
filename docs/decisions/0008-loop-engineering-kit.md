# ADR 0008: Loop Engineering Kit 有界工作裁决面

PrimitiveReview: required

## Status

Accepted。实现位于 `internal/loopkit`，Kit 默认不启用；第一阶段只提供确定性契约和裁决，
执行始终归 Agent，验证事实始终归 `validationevidence`。

## 1. Decision

新增 Loop Engineering Kit，唯一新增 Primitive 是纯函数：

```go
func Decide(spec workspec.Spec, history []Attempt, gates []GateStatus, now time.Time) (Decision, error)
```

四个参数全部注入，函数内无文件、Git、网络、进程或时钟读取。控制契约由三个正交轴组成：

- `trigger`：`explicit`、`scheduled`、`agent-proposed`；
- `stop`：尝试数、耗时、总 token、失速阈值和上下文压力阈值；
- `authority`：检测式写范围、必需门禁和人工检查点。

裁决返回一个非终止态 `continue`，或五个具名终止态：`stop-satisfied`、`stop-budget`、
`stop-stalled`、`stop-violation`、`checkpoint`。规则按契约顺序求值，第一条命中即返回。

## 2. 与既有权威的边界

| 主题 | 权威 | Loop 允许做 | Loop 禁止做 |
|---|---|---|---|
| 声明式收敛 | `lifecycle` | 工作目标可引用 lifecycle 门禁 | 安装、更新、卸载或替代 adapter |
| 待实现队列 | `todolist` | 把 TODO 计划转换为 WorkSpec | 改写 TODO 生命周期或扫描 backlog |
| 验证执行 | `testengine` | 接收门禁状态值 | 执行测试或新增 test engine |
| 内容证据 | `validationevidence` | 保存 `profile/identity/receiptID` 字符串引用 | 定义、签发或复制 Receipt |
| 一次行动 | Agent | 裁决下一步和停止原因 | 代替 Agent 修改工作树 |

边界判据：**目标状态能被声明的，用 lifecycle；只能由门禁判定的迭代目标，才用 loop。**

## 3. 权限是检测式，不是预防式

`authority.writeScope` 是事后裁决输入：CLI 比较实际 Git 状态与 allow/deny 后，把结果作为
`GateStatus` 注入 `Decide`。它不能阻止操作系统写文件，也不扩大用户授权。越界事实触发
`stop-violation`，由调用者决定恢复或请求人工处理；Kit 不自动 reset、checkout 或删除文件。

## 4. 为什么永不实现 `work run`

内建 `work run` 会同时拥有循环调度、Agent 调用、验证触发和终止策略，形成第二控制面，并与
现有 runner、testengine、Git hook 和 CI 重叠。因此正式命令只允许 `work validate/next/status/record`：
前三者只读，`record` 仅追加 Agent 已完成的一次尝试。不存在 `prepare`、`step` 或无限循环入口。

## 5. 为什么不定义第二 Receipt

验证证据已经由 `validationevidence` 绑定 Git Tree OID、验证语义和完整性摘要。Loop 自建
Receipt 会允许裁决者给自己签发通过证明。`gateref.GateRef` 因而只持三个字符串，且不 import
`validationevidence`；CLI 层负责调用现有 check 语义并组装 `GateStatus`。

## 6. 状态、可靠性与回滚

`Attempt` 内嵌既有 `tokenusage.Usage`、`subjectTreeOID`、`gateRefs` 与起止时间。连续相同 Tree
用于失速判断；token 和上下文数据不另建模型。工作会话状态属于当前 worktree，存放在
`.aicoding/state/work/<id>/`，其中 `attempts.jsonl` 只追加。解析失败、spec digest 漂移或越界
均 fail-closed。

回滚时删除 `internal/loopkit`、三份 loop schema、模板、Kit manifest/registry entry、四条 CLI
路由与本文即可；不迁移或删除 validation Receipt，不修改 lifecycle/testengine/report 内核。

## §12 Checklist 自评

**架构**

- 单一职责：只裁决一次状态转移；观察、行动和验证均由既有权威承担。
- 可继续拆分：合同、profile、gate 引用和转移函数已经按稳定变化点分包，不增加中心 Manager。
- 可复用：`Decide` 只接收值对象，可由 CLI、CI 或未来独立消费者直接调用。
- 无重复实现：token 复用 `report/tokenusage`，证据复用 `validationevidence`，无第二 Receipt。
- 新 Primitive 必要性：现有模块没有“基于历史、预算和证据决定继续还是停止”的纯函数。

**性能**

- Fast Path：一次线性扫描 history 和 required gates；函数内零 I/O、零 Git、零 Agent。
- 无关扫描：不遍历仓库、Kit registry 或 Receipt 目录。
- 重复 I/O/计算：调用层一次收集 Git/证据后注入；Tree OID 直接复用尝试记录。
- 最小 Context：历史保存结构化摘要和引用，不保存对话或测试日志正文。

**质量**

- 确定性：时间显式注入；相同四参数产生相同决策字节。
- 接口稳定：三正交轴、五终止态和 GateRef 是首期最小外部契约。
- 独立测试：表驱动测试覆盖预算顺序、失速、上下文、门禁、违规和内容身份。
- 自由组合：Agent 可按决策行动，但 Kit 不拥有调度、发布或工作区修复权限。
