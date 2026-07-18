# Tasks: Contract Freeze And Acquisition Boundary

执行者按顺序完成并勾选；从最新 origin/main 拉新分支执行（见计划"分支说明"）。

- [x] T1 复核基线：确认 `config/kit-registry.json`、`config/mcp-registry.json`、`config/codex-kit.json` 零 URL；用 cloneableSourcePattern 对 `config/**` 跑语料审计，结果写入交付说明。
- [x] T2 `config/dependency-governance.json`：新增 `acquisitionBoundary` 节。
- [x] T3 `config/schemas/dependency-governance.schema.json`：新增对应 schema 定义。
- [x] T4 `internal/governance/dependencies.go`：实现 `activation manifests URL-free` 与 `cloneable sources registry` 两个 check。
- [x] T5 `internal/governance/dependencies_test.go`：两个 check 的正/负用例。
- [x] T6 `internal/cli/dependency_governance_fixture_test.go`：CLI 层 fixture 回归。
- [x] T7 交叉引用：COMMANDS.md、测试文档、ARCHITECTURE_HANDBOOK.md §8 各加一行指向新边界文档；运行 docsync 完成登记（如需要）。
- [x] T8 `CHANGELOG.md` 条目（Unreleased 段）。
- [x] T9 全门禁链通过：`go build`、包测试、`governance dependencies`、`docsync all`、`test --profile Smoke`、`test --profile Full` 全部 ok=true。
- [x] T10 自查 diff 半径：改动不超出计划文件清单；零行为变化（两个新 check 在当前仓库零违规）。
