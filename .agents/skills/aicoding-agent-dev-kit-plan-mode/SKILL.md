---
name: aicoding-agent-dev-kit-plan-mode
description: 用于 AiCoding 中非平凡、架构敏感或需求模糊的工作；在实现前执行计划模式（Plan Mode），适配 Spec Kit 阶段，并要求用户先选择模糊架构路线。
---

# AiCoding Agent Dev Kit 计划模式（Plan Mode）

将本 Skill 作为 AiCoding Agent Dev Kit 的路由 overlay 使用。默认中文优先：执行计划、权限摘要、命令目的、验证结果、风险说明、rollback/handoff 都必须中文说明，英文术语只作为括号补充。

## 首要步骤

实现前先声明：

```text
Mode: 计划模式（Plan Mode）
Capability domain: 能力领域
Current context loaded: 已加载上下文
Unknowns: 未知项
Decision required: 是否需要用户决策
Planned artifacts: 计划产物
```

## 阶段顺序

使用以下顺序：

```text
澄清（Clarify） -> 规格化（Specify） -> 计划（Plan） -> 用户决策（User Decision） -> 任务（Tasks） -> 实现（Implement） -> 验证（Verify） -> 交接（Handoff）
```

## 模糊架构处理

如果存在多条可行技术路线，不要直接实现。

先创建或更新：

```text
docs/spec/PRD_OPTIONS.md
docs/spec/NEEDS_USER_DECISION.md
```

然后向用户展示 2-5 个技术路线选项，并请求用户选择。

只有在以下记录存在后才能继续：

```text
docs/spec/SELECTED_SOLUTION.md
.aicoding/memory/DECISIONS.md
```

## Spec Kit 适配

将 Spec Kit 流程作为操作模型：

- constitution：AiCoding 规则和 `AGENTS.md`。
- specify：用户意图和约束。
- clarify：问题澄清和方案选择门禁。
- plan：实现计划。
- tasks：执行任务。
- analyze/checklist：验证计划、可追溯性和门禁。
- implement：仅在计划和决策门禁通过后实现。

## Superpower 风格习惯

使用显式模式、最小上下文加载、进度检查点和停止条件。

不依赖外部 Superpower 包已安装。

## 必要命令

```powershell
pwsh scripts\new-agent-plan-mode-session.ps1 -Feature "<功能名>" -Description "<需求描述>" -NeedsDecision -Json
pwsh scripts\verify-agent-dev-kit-plan-mode.ps1 -Json
pwsh scripts\hooks\aef\plan-mode-gate.ps1 -Event manual -Mode warn -Json
```

## 交接契约

最终回复必须包含：

```text
模式：
已实现：
已验证：
未验证：
决策记录：
回滚：
下一步：
```
