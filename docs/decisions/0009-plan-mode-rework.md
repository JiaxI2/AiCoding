# ADR 0009: Plan Mode 内容绑定与强制门禁

PrimitiveReview: required

## Status

Accepted。

## 1. Decision

Plan Mode 从 PowerShell overlay 收敛为 Go 领域能力：`PLAN.md` 是每个计划的机器契约，
`plan approve` 在 clean worktree 上把 `HEAD^{tree}` 写入 `approvedTree`，`plan status`
检测批准树之后的 scope 内漂移与越界，`plan check --staged` 只放行被 approved plan scope
覆盖的架构敏感路径。pre-commit 对未覆盖路径 fail-closed，但不在 hook 内运行任何测试。

批准命令是 plan 域唯一写入口，只改目标 `docs/spec/<id>/PLAN.md` 的 `status` 与
`approvedTree`。状态转为 `implemented` 仍由人确认；CLI 只在 scope 覆盖完整且当前树的
声明 gate 均命中有效 validationevidence Receipt 时给出建议。

## 2. 四个领域的边界

```mermaid
flowchart LR
    T["todolist：排队与完成状态"] --> P["plan：意图、批准树与 scope"]
    P --> L["loop：一次迭代后继续或停止"]
    V["validationevidence：当前树 PASS Receipt"] --> P
    V --> L
    P -. "不执行测试、不调度循环" .-> V
```

| 领域 | 唯一问题 | 允许持有 | 明确禁止 |
|---|---|---|---|
| `todolist` | 哪些工作待做、状态为何 | Markdown 头部状态与 Verify | 批准实现树、判断验证通过 |
| `plan` | 哪棵内容树获批、变更是否在 scope | Tree OID、scope、GateRef | 执行测试、签发 Receipt、循环调度 |
| `loop` | 一次尝试后继续还是停止 | WorkSpec、Attempt、GateRef | 修改 PLAN、批准内容、签发 Receipt |
| `validationevidence` | 当前树是否有可复用 PASS 证明 | 唯一 Receipt 与完整性存储 | 决定计划范围或工作优先级 |

plan 与 loop 都只持 `profile/validationIdentity/receiptID` 字符串引用；仓库内不新增第二个
`Receipt` 类型。

## 3. 为什么强制点是 pre-commit

Agent hook 依赖调用方自觉触发，适合作为迁移期提示和交互辅助，不能证明所有提交路径都经过
检查。pre-commit 位于 Git 内容进入历史前的共同边界，命令行、IDE 与 Agent 的 staged 内容
都会到达这里，因此承担强制责任。门禁只运行预构建 CLI 的路径裁决，不构建、不测试、不联网；
验证 profile 继续在 hook 外运行并由 Receipt 表达。

这仍不是操作系统权限模型：Agent 或人可以在提交前写入任意文件，也可以显式绕过本地 hook。
AiCoding 能诚实保证的是，在正常受治理提交路径上检测 staged 越界并拒绝下一步，而不是阻止
已经发生的磁盘写入。

## 4. 为什么绑定 Tree 而不是 commit

批准对象是内容，不是提交元数据。message-only amend 会改变 commit OID，但 `HEAD^{tree}`
不变；只重排或 rebase 而最终内容相同，也应继续命中同一批准。反过来，只要任一受版本控制
文件内容变化，Tree OID 就变化并进入 drift 分类。该语义与 ADR 0007 的 Receipt 内容身份一致，
避免为提交说明、作者或父提交变化制造伪漂移。

批准写入 PLAN.md 本身不能递归包含在所绑定 Tree 中，因此 `docs/spec/**` 由 plan policy
明确 exempt；业务 scope 应描述实现内容，不把批准元数据当作实现进度。

## 5. 漂移与完成建议

对单个已批准计划，CLI 只做一次 `approvedTree..HEAD^{tree}` 文件差异，然后分类：

```text
changed ∩ scope                  -> drift（实现进行中或需复核 plan）
changed ∖ scope ∩ exempt        -> exempt（计划元数据等）
changed ∖ scope ∖ exempt        -> outOfScope（警告，阻断下一步由调用方决定）
```

当每条 scope pattern 至少命中一个变化路径、没有越界，且所有声明 gate 对当前 Tree 的
validationevidence exact check 命中时，`plan status` 才返回 `completionSuggested=true`。
CLI 不自动写 `implemented`，避免把人类契约降格为自动机副作用。

## 6. PowerShell 退役路径与回滚

旧脚本按三个 release 阶段退役：本阶段保持 Go CLI 薄壳；下一 release 标记 deprecated 并从
默认 hook registry 移除；再下一 release 删除薄壳和 overlay 专用说明。期间脚本不得重建路径
匹配、漂移或 Receipt 语义。

回滚强制门禁只需恢复 `.githooks/pre-commit` 的 warning 行为；回滚内容绑定则移除
`plan approve` 与 status 漂移投影，但保留 per-plan 目录和历史 frontmatter。Receipt 存储、
todolist 与 loop 状态均无需迁移。

## §12 Checklist 自评

**架构**

- 单一职责：plan 只拥有批准内容与范围漂移裁决；执行、循环和证据各归既有权威。
- 可继续拆分：CLI 负责 Git/证据编排，`internal/plan` 保持解析、纯分类和唯一文件写入。
- 可复用：Tree diff 进入 `gitx`；plan 与 loop 共用 GateRef/Receipt 引用语义。
- 无重复实现：不新增 Receipt、测试执行器、TODO 状态机或 Agent hook 控制面。
- 新 Primitive 必要性：既有模块没有“批准树 + scope”绑定与 staged 覆盖判据。

**性能**

- Fast Path：pre-commit 只读 staged paths、plan frontmatter 和路径规则，不运行 profile。
- 无关扫描：status 单计划只做一次 tree diff；Receipt 使用精确 identity lookup。
- 重复 I/O/计算：Tree OID 与 diff 在 CLI 层收集后注入纯分类函数。
- 最小 Context：输出只列路径、Tree OID 与 GateRef，不读取 Markdown 正文语义或报告正文。

**质量**

- 确定性：相同 approved/current Tree、scope 与 policy 得到相同排序结果。
- 接口稳定：唯一写命令为 `approve`；status/check 保持检测式只读。
- 独立测试：覆盖 dirty 拒绝、amend 同树、scope 内漂移、越界与批准覆盖正反例。
- 自由组合：pre-commit、Agent 与 CI 消费同一 CLI 结果，不互相复制 policy。
