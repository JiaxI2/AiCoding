# Implementation Plan: Dependency Direction And Stable Identity Governance

Plan Status: Approved

1. 新增机器策略与 schema。
2. 新增 Go dependency governance checker，并接入 lint/pre-commit/聚合测试。
3. 约束 Kit/MCP registry binding、Skill 命名、MCP prompt ownership 和 namespace。
4. 增加资产版本不可观察与 README badge authority 校验。
5. 立即清理现有 capability 反向命名和自版本代码。
6. 更新 AGENTS、架构、维护、命令与变更记录。
7. 执行 Go、MCP、Kit、Markdown、Plan Mode 和 Git 门禁。

Rollback: 删除新增 policy/schema/checker/CLI 路由并恢复本次明确列出的资产命名；不触碰用户的其他未提交改动。
