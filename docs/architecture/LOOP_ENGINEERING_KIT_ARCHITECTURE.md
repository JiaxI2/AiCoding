# Loop Engineering Kit 架构

Status: Proposed

## 1. 核心结论

Loop Engineering 作为 AiCoding 核心 Kit 开发，但不进入六模块稳定内核，也不形成第二控制面。

架构表达：

```text
Work Domain × Control Mode × Execution Policy
```

### Work Domain

- `project-development`
- `repository-maintenance`
- `ci-repair`
- `performance-experiment`
- `documentation-maintenance`
- `architecture-evolution`

### Control Mode

- `turn`：人启动，人决定下一步。
- `goal`：人定义目标，Evaluator 与停止规则决定是否继续。
- `time`：时钟触发一次有界工作。
- `proactive`：事件或观察器发现候选工作，但仍受权限与人工检查点限制。

### Execution Policy

- workspace policy
- write-scope policy
- verification policy
- budget policy
- human-checkpoint policy
- persistence policy

## 2. 与 AiCoding 现有架构的关系

```text
Loop Policy / Work Profile
        ↓
Existing typed CLI catalog
        ↓
Existing runner / gitx / report
        ↓
Domain capability and domain-owned state
```

复用：

- `internal/runner`：单次有界执行；
- `internal/report`：唯一 JSON 证据外壳；
- `internal/gitx`：Git 操作边界；
- `internal/registry`：规范化输入与 digest；
- `repo-context`：按域加载上下文；
- `testengine`：AiCoding 仓库维护的唯一测试权威。

禁止：

- runner 决定是否继续 Loop；
- report 自动触发修复；
- Skill 修改 WorkState；
- Adapter 同时拥有项目开发和仓库维护策略；
- 新建全域 LoopManager。

## 3. 核心对象

### WorkSpec

不可变工作契约：目标、验收、控制模式、权限、预算和停止规则。规范化后生成 `workDigest`。

### Attempt

一次有界执行。每次 Attempt 必须绑定：

- Work digest；
- 输入事实 digest；
- source HEAD；
- workspace lease；
- agent/tool invocation；
- evidence receipt。

### EvidenceReceipt

Evaluator 输出的机器证据。Agent 的完成声明不是证据。

### TransitionDecision

纯函数：

```text
WorkSpec + PreviousState + Evidence + Budget
→ NextState + Reason
```

### WorkProfile

领域绑定，不保存运行状态：

- 允许的 Control Mode；
- 默认写范围；
- Gate policy；
- 人工检查点；
- 推荐 Skill。

## 4. 项目开发与仓库维护解耦

### Project Development

- 任务来源：用户、Issue、Spec；
- 评价：项目自身 build/test/acceptance；
- 默认控制模式：turn、goal；
- 架构选择和产品语义通常需人工判断；
- 不能硬编码 AiCoding 的 `doctor/test/release`。

### Repository Maintenance

- 任务来源：CI、Todo、漂移、依赖、周期审计；
- 评价：仓库治理、健康检查、测试和 release gate；
- 可使用 time/proactive；
- 默认不自动 merge 或 release；
- AiCoding 自维护 Profile 使用已有正式命令，不复制检查实现。

## 5. 推荐包边界

```text
internal/loopkit/
  workspec/      规范化与 digest
  controlmode/   四类模式的结构约束
  evidence/      证据模型与校验
  transition/    纯状态转移
  profile/       静态 Work Profile catalog
```

后续真实需求出现后再增加：

```text
workspace/       worktree lease
attempt/         一次执行记录
agenthost/       Codex/Claude/manual adapter
trigger/         定时或事件标准化
```

不得提前增加 `loopmanager`、全域 journal 或自动调度服务。

## 6. 状态机

```text
DRAFT
  → READY
  → RUNNING
  → VERIFYING
      ├→ VERIFIED
      ├→ CONTINUE
      ├→ BLOCKED
      ├→ NEEDS_HUMAN
      └→ EXHAUSTED
```

终止态：`VERIFIED`、`BLOCKED`、`NEEDS_HUMAN`、`EXHAUSTED`。

`CONTINUE` 不是完成，只表示允许下一次 Attempt。

## 7. Gate Vector

软件工程不使用单一总分替代正确性：

```text
required gates   任一 FAIL 即不能 VERIFIED
non-regression   必须满足基线或显式豁免
advisory gates   只告警，不掩盖 required gate
```

典型维度：

- correctness
- compatibility
- safety
- performance
- maintainability
- scope compliance
- documentation

## 8. 分阶段落地

### Phase 1：契约与纯函数

- WorkSpec schema；
- Profile schema；
- Evidence schema；
- Work digest；
- Transition evaluator；
- 示例 Profile；
- 单元测试。

不调用 Agent，不创建 Worktree。

### Phase 2：Workspace 与 Attempt

- worktree lease；
- source state snapshot；
- 单次 Attempt；
- evidence freshness；
- `work prepare/step/status`。

### Phase 3：两个真实消费者

- `project-development`；
- `aicoding-repository-maintenance`。

没有这两个真实消费者，不允许修改稳定内核。

### Phase 4：有界 Goal Loop

- `work run --max-iterations N`；
- no-progress；
- repeated-failure；
- input-drift；
- budget exhausted。

### Phase 5：Time / Proactive

只有 Trigger、权限和停止规则稳定后再加入。调度器不进入仓库核心进程。

## 9. 验收标准

- 项目开发与仓库维护 Profile 不共享状态文件；
- 同一 WorkSpec 规范化结果和 digest 稳定；
- required gate FAIL 时绝不返回 VERIFIED；
- Agent claim 不能直接改变状态；
- 达到预算后返回 EXHAUSTED；
- architecture-evolution 默认不能使用 proactive 自动关闭；
- 所有 CLI 结果继续使用现有 `report.Result`；
- 不增加第二 test engine 或 report schema authority。
