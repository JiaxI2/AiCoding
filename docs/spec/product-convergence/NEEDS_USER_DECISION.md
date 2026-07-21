# 决策已解决（Decision Resolved）：product-convergence

Plan Status: Resolved

## 选择结果

选择方案 A：兼容优先的统一控制面。

## 选择依据

- 方案 B 保留多个正式入口，违反“产品入口唯一”。
- 方案 C 新增扁平顶层 `install|update` 命令面，超出“收敛而非扩张”的约束。
- 方案 A 复用当前 Go CLI、registry、Runner、Report 与生命周期实现，并提供一个版本的兼容层。

## 实施门禁

`docs/spec/SELECTED_SOLUTION.md`、当前实现计划、任务、可追溯性和
`.aicoding/memory/DECISIONS.md` 完成记录后，允许进入分阶段实现。
