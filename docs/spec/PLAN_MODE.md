# 计划模式会话（Plan Mode Session）：issue-lifecycle-governance

Mode: Plan
Plan Status: Approved
Created: 2026-07-16 11:35:08 +08:00
Feature Slug: issue-lifecycle-governance

## 需求 / Request

将 Issue 创建、分类、流转与关闭标准落地为 AiCoding 仓库级 Git governance policy，并保持现有 Skill source 与 submodule 只读边界

## 必须执行的顺序

1. 澄清模糊点。
2. 明确用户意图和约束。
3. 生成实现计划。
4. 如果存在多条架构路线，先请求用户选择。
5. 将已选择计划拆分为任务。
6. 只有在决策和计划门禁通过后才能实现。
7. 按需要执行 Smoke / schema / golden / doc sync 验证。

## 当前决策状态

需要用户决策：False

## 已确认决策

- 不创建独立 Issue Skill；本轮只增加 AiCoding 仓库策略、表单、label workflow 与 Go lint。
- Codex-Skills 的 `platform/aicoding-git-governance` 保持已发布只读依赖，本轮不宣称修改 canonical source 或 generated plugin。
- AiCoding 增加 Issue Forms、label manifest、生命周期 workflow 和 Go governance lint。
- AiCoding 在独立干净发布 worktree 复现并验证完整修改集，不覆盖既有脏工作树。
- `CodingKit/agents/skills` 只更新到本轮已发布的 Visio Skill commit，不与 Issue policy 建立未发布的反向绑定。
