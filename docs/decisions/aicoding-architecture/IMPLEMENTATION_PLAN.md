# 实施计划：正交内核闭环

Plan Status: Implemented and Accepted

## 已完成

1. 固化 `ExecutionPlan` descriptor、不可变选择、snapshot 和 digest，迁移 pre-commit。
2. 建立规范化 Registry Snapshot + Digest，迁移 Kit/MCP loader。
3. 建立 Typed Command Catalog，统一 handler routing、alias、namespace 和 help。
4. 建立通用 `CatalogSnapshot` 内容树，组合 registry 与 referenced manifest digest。
5. 建立 Kit/MCP domain catalog，并让 lifecycle/领域命令消费 detached manifest values。
6. 建立静态 lifecycle Adapter Catalog，声明 input、state owner、entrypoint 与 action effect。
7. 删除 lifecycle scope switch，把 adapter selection 转为 `ExecutionPlan`，成为第二真实消费者。
8. 在 JSON 中暴露 adapter catalog、domain input 与 plan digest。
9. 明确 external Skill/MCP 的 install/update/sync/uninstall、Agent 调用与 state/rollback 边界。
10. 用正交模块、局部测试半径与冻结条件替换无限迁移列表。

## 验收

1. Module contracts：registry、runner、Kit、MCP、lifecycle、report、CLI。
2. Consumer regression：Kit/MCP/runtime Skill lifecycle 与真实 CLI JSON。
3. Repository gates：DocSync、Markdown、dependency/layout/lint、hooks、diff checks。
4. Product gates：doctor、Smoke、Full、Release；子模块保持 clean。
5. 独立提交本批实现，不 push/PR/release。

## 非本计划

- capability graph；
- 全域 journal/atomic rollback；
- HTTP/gRPC/MCP 产品控制 API；
- dynamic plugin ABI；
- C/native core；
- Taskfile/docs 自动生成器、通用 result 大重构或 speculative cache framework。

这些不是“未完成架构项”。只有新的现实证据满足解冻规则时才进入独立 ADR。
