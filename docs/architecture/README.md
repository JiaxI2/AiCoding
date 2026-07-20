# 架构文档阅读路径

Status: Accepted

本文只提供阅读顺序和文档分层，不定义新的架构契约。主题冲突时，以“必读”中的总体契约
及对应领域的 Accepted/Frozen 文档为准。

## 必读（改任何代码之前，共约 755 行）

1. [AICODING_CORE_ARCHITECTURE.md](AICODING_CORE_ARCHITECTURE.md)：系统是什么，稳定内核与总体契约事实。
2. [PRIMITIVE_CONSTITUTION.md](PRIMITIVE_CONSTITUTION.md)：Primitive 应该长什么样，如何判断新增是否必要。
3. [FREEZE_AND_ACQUISITION_BOUNDARY.md](FREEZE_AND_ACQUISITION_BOUNDARY.md)：哪些契约已冻结，哪里不能随意修改。
4. [EXTENSION_ADAPTER_CONTRACT.md](EXTENSION_ADAPTER_CONTRACT.md)：新能力如何通过 adapter 进入系统。

## 按需（做特定领域时）

- [CLI_MCP_CONTROL_PLANE.md](CLI_MCP_CONTROL_PLANE.md)：CLI 与 MCP 控制面权威。
- [KIT_LIFECYCLE_ARCHITECTURE.md](KIT_LIFECYCLE_ARCHITECTURE.md)：Kit 生命周期与状态收敛。
- [GIT_REUSE_BOUNDARY.md](GIT_REUSE_BOUNDARY.md)：Git 内容身份与复用边界。
- [POWERSHELL_BOUNDARY.md](POWERSHELL_BOUNDARY.md)：PowerShell 专项保留面。
- [DOC_SYNC_PLUS_SPEC.md](DOC_SYNC_PLUS_SPEC.md)：DocSync Plus 合同。
- [GRAPH_FIRST.md](GRAPH_FIRST.md)：图优先的依赖表达。
- [LOOP_ENGINEERING_ARCHITECTURE.md](LOOP_ENGINEERING_ARCHITECTURE.md)：有界迭代的下一步裁决。
- [PLAN_MODE_ARCHITECTURE.md](PLAN_MODE_ARCHITECTURE.md)：批准 Tree、scope 漂移和 pre-commit 门禁。
- [CODEX_KIT_ARCHITECTURE.md](CODEX_KIT_ARCHITECTURE.md)：Codex Kit 平台集成边界。

## 派生视图（不定义契约，可不读）

- [00-vision.md](00-vision.md) 至 [07-roadmap.md](07-roadmap.md) 的编号系列；
- [ARCHITECTURE_HANDBOOK.md](ARCHITECTURE_HANDBOOK.md)；
- [MCP_CONTROL_PLANE.md](MCP_CONTROL_PLANE.md)（MCP 主题权威仍是 `CLI_MCP_CONTROL_PLANE.md`）。

## Kit 架构文档模板骨架（约定）

新增 Kit 架构文档按以下顺序组织：

```text
结论
→ 系统位置图
→ 邻居边界判据
→ 数据契约
→ 可靠性与安全表
→ 演进边界（明确不做）
```

模板是写作约定，不要求为每个标题建立新的代码抽象。
