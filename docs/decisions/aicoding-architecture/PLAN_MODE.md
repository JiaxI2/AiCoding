# 计划模式：AiCoding 内核与扩展图架构

Mode: Plan -> Architecture Baseline -> Implement -> Verify
Plan Status: Implemented and Accepted

## 需求

参考 Git、GitHub 和 mattpocock/skills，审查当前 AiCoding，从可扩展性、可玩性和
高性能角度确定一套长期架构。

## 已确认用户决策

- 基础功能优先做到稳定、可组合、可测试和高性能。
- 扩展功能全部建立在基础能力之上。
- 实现目录、文件名和稳定 identity 不编码版本。
- README、CHANGELOG 和其他说明文档可以出现版本信息。
- 尽量一次确定长期方案，不维护平行架构。

## 本轮成功标准

1. 形成唯一权威架构文档。
2. 明确内核、扩展、porcelain、分发和 runtime 边界。
3. 形成 facts -> plan -> adapter -> domain state -> evidence 的可验证闭环。
4. 支持 external Skill/MCP lifecycle 与 Agent CLI/JSON 调用。
5. 定义正交模块、局部验证半径与明确冻结条件。
6. 不修改 Codex-Skills 子模块或 plugin cache。

## 第一批执行范围

用户随后明确选择先实现：

1. `ExecutionPlan` 升级为核心对象；
2. Registry Snapshot + Digest；
3. Typed Command Catalog。

在这三个对象之上，本轮只继续实现已有真实消费者要求的 manifest catalog tree、静态
adapter catalog、lifecycle ExecutionPlan 和 digest evidence。实现保持同一架构和 worktree
边界，不引入 capability graph、全域 journal、动态 plugin、远程控制 API 或 C 加速层。

## 架构停止条件

- Kit/MCP 共用内容树 snapshot；
- pre-commit/lifecycle 共用 ExecutionPlan；
- Kit/MCP/runtime Skill 通过静态 adapter catalog 组合；
- Agent/Skill 通过 CLI/JSON 完成 plan/apply/verify；
- 模块职责与测试影响半径固化。

满足后冻结总体架构；新增 Skill/MCP/component 属于功能扩展。
