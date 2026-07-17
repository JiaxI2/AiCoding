# 可追溯性（Traceability）：product-convergence

| 需求 / 决策 | 计划章节 | 任务 | 验证 |
|---|---|---|---|
| 产品入口唯一 | 方案 A 正式/兼容入口 | Phase 1、3、6 | CLI help；`CLI_DEPRECATED`；文档扫描 |
| 唯一测试引擎 | `internal/testengine` | Phase 2、3 | test ID 单次执行；无递归调用 |
| 唯一报告体系 | `report.Result` + test report schema | Phase 2、5 | Schema；JSON-only stdout |
| 唯一生命周期 | `internal/lifecycle` 静态 adapter | Phase 4 | dry-run；临时配置；rollback |
| CLI 契约稳定 | help、参数、退出码、JSON | Phase 1、5 | subprocess contract matrix |
| 文档入口唯一 | README -> COMMANDS/Architecture hub | Phase 6 | DocSync；Markdown links |
| Release 闭环 | release gate 复用 Release test plan | Phase 3、5 | Full/Release gate；fresh-clone |
| 兼容一个版本 | 旧入口统一警告和路由 | Phase 1、3 | 兼容命令回归 |
| submodule 只读 | 不修改 `CodingKit/agents/skills` | 全阶段 | submodule status clean |
