# 计划模式会话（Plan Mode Session）：product-convergence

Mode: Plan -> Implement
Plan Status: Approved
Feature Slug: product-convergence

## 需求 / Request

将 AiCoding 收敛为一个正式 CLI 入口体系、一个测试引擎、一个报告体系、
一个生命周期和一个 Release 闭环，同时兼容旧入口一个版本。

## 必须执行的顺序

1. 澄清模糊点。
2. 明确用户意图和约束。
3. 生成实现计划。
4. 如果存在多条架构路线，先请求用户选择。
5. 将已选择计划拆分为任务。
6. 只有在决策和计划门禁通过后才能实现。
7. 按需要执行 Smoke / schema / golden / doc sync 验证。

## 当前决策状态

需要用户决策：False；方案 A 已由目标约束确定。

## 已确认决策

- `test --profile Smoke|Full|Release` 是唯一正式测试入口。
- `lifecycle` 是唯一正式产品生命周期命名空间。
- `release gate` 调用同一个测试引擎，不递归调用 CLI 聚合器。
- 兼容入口保留一个版本并输出 `CLI_DEPRECATED`。
- `report.Result` 保持兼容外壳，统一参数错误、退出码、JSON stdout 和测试报告。
- 所有实现只发生在 `codex/product-convergence` worktree。
- `CodingKit/agents/skills` 保持只读和 clean。
