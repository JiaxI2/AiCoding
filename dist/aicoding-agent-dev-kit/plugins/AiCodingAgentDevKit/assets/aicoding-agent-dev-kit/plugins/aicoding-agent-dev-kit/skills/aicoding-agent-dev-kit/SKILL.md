---
name: aicoding-agent-dev-kit
description: 用于需求澄清、方案矩阵、Spec Pack、TDD、顺序上下文加载、决策记忆、进度监控和质量门禁的轻量 Agent 入口。
---

# AiCoding Agent Dev Kit Thin Skill

只将本 Skill 作为 Agent 路由层使用。面向用户的计划、权限摘要、验证结果、风险和 rollback/handoff 说明必须中文优先；英文术语可作为括号补充。

## 领域中立规则

不要把特定应用示例写入可复用 Kit。

具体示例应在读取目标仓库上下文后，写入目标仓库自己的文档或规格。

## 如果需求模糊

1. 先不要实现。
2. 创建或更新 `spec/PRD_OPTIONS.md`。
3. 展示 2-5 个技术选项，说明优点、缺点、风险、验证方式、工作量和推荐条件。
4. 请求用户选择或拒绝某个选项。
5. 将用户选择写入 `spec/SELECTED_SOLUTION.md`。
6. 将决策记录到 `.agent-memory/DECISIONS.md`。
7. 再更新 PRD / APP_FLOW / IMPLEMENTATION_PLAN / ADR / traceability。

## 实现前

1. 使用 `aicoding-agent-kit load --repo . --auto` 加载上下文。
2. 阅读 `.agent-dev-kit/context/context-pack.md` 和 manifest。
3. 检查 `spec/IMPLEMENTATION_PLAN.md` 是否包含 Red-Green-Refactor。
4. 初始化或更新进度看板。
5. 在提交或 handoff 前运行质量门禁。
