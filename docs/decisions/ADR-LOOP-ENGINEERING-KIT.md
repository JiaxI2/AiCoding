# ADR: Loop Engineering 作为 AiCoding 核心 Kit

Status: Proposed
PrimitiveReview: required

## Context

AiCoding 已有稳定的 snapshot、plan、runner、adapter、report 和 domain-owned state，但缺少跨多次 Attempt 的任务控制契约。项目级开发与仓库维护需要共享执行原语，同时保持任务来源、权限、验证与停止规则独立。

## Decision

增加 `loop-engineering-kit`，位于平台层，以 WorkSpec、WorkProfile、EvidenceReceipt 和 TransitionDecision 为第一阶段 Primitive。Kit 不拥有第二 CLI、第二 lifecycle、第二 runner、第二 report 或第二 test engine。

## Consequences

- 可以统一表达 turn/goal/time/proactive；
- 项目开发和仓库维护保持解耦；
- Karpathy-style 优化只作为 goal mode 的一种策略；
- Proactive 仅在权限与证据边界明确后启用；
- 需要新增 typed command catalog 条目时，仍进入现有 CLI。

## Rejected

- 单一 `LoopManager`；
- 把项目开发建模为 lifecycle scope；
- 所有任务默认 goal loop；
- Skill 自己判定完成；
- 无限循环自动执行；
- 默认自动 merge/release。

## §12 Checklist 自评

- 单一职责：每个初始包只负责 WorkSpec、ControlMode、Evidence、Transition 或 Profile。
- 执行成本：Phase 1 不扫描仓库、不启动外部进程、不调用 Agent。
- Fast Path：WorkSpec 校验和状态转移为内存纯函数。
- 确定性：规范化 JSON 排序后计算 digest；无时间戳和绝对路径进入 digest。
- 稳定接口：schemaVersion=1，只允许附加兼容字段。
- 可测试：每个包都有独立 Go test。
- 可组合：CLI/Skill/未来 Trigger 仅组合这些 Primitive。
