# 计划模式：AiCoding 内核与扩展图架构

Mode: Plan -> Architecture Baseline
Plan Status: Approved

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
3. 给出性能基线、预算和机器门禁。
4. 把迁移拆成可验证提交，但不引入版本化实现路径。
5. 不修改 Codex-Skills 子模块或 plugin cache。

## 第一批执行范围

用户随后明确选择先实现：

1. `ExecutionPlan` 升级为核心对象；
2. Registry Snapshot + Digest；
3. Typed Command Catalog。

实现保持同一架构和 worktree 边界，不提前引入 capability graph、动态 plugin、
state/journal 或 C 加速层。
