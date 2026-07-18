# Tasks: Git Reuse Boundary

执行者按顺序完成并勾选；每项完成后本地跑对应最小验证，最后统一跑门禁链。

- [x] T1 迁移 `internal/cstyle/cstyle.go` 两处 git 调用到 gitx，保持输出解析等价。
- [x] T2 迁移 `internal/platform/files.go` rev-parse 到 gitx（若产生依赖问题，改为豁免登记并在 PR 说明）。
- [x] T3 迁移 `internal/kit/plugin_runtime.go`（hooksPath）与 `internal/kit/structure.go`（status --short）到 gitx。
- [x] T4 迁移 `internal/lifecycle/runtime_skill.go` 两处 rev-parse 到 gitx。
- [x] T5 `config/dependency-governance.json`：新增 gitx goPackageBoundaries entry 与 `gitProcessBoundary` 节。
- [x] T6 `config/schemas/dependency-governance.schema.json`：为新节增加 schema。
- [x] T7 `internal/governance/dependencies.go`：实现 `git process ownership` 与 `gitx importer allowlist` 两个 check。
- [x] T8 `internal/governance/dependencies_test.go`：两个新 check 的正/负用例。
- [x] T9 `internal/cli/dependency_governance_fixture_test.go`：CLI 层 fixture 回归（两个 check 名出现在 JSON 报告）。
- [x] T10 `internal/cli/catalog_test.go`：新增 `TestCommandCatalogRejectsGitPorcelainVerbs`（完整禁用集合 + 豁免注释）。
- [x] T11 运行 docsync 门禁，按报错完成新文档登记（如需要；覆盖 GIT_REUSE_BOUNDARY.md 与 ARCHITECTURE_HANDBOOK.md 两份新文档）。
- [x] T12 `CHANGELOG.md` 条目。
- [x] T13 全门禁链通过：`go build`、包测试、`governance dependencies`、`docsync`、`test --profile Smoke`、`test --profile Full` 全部 ok=true。
- [x] T14 自查 diff 半径：改动文件不超出 IMPLEMENTATION_PLAN 列出的范围；无新 CLI 命令、无 snapshot/runner/report 契约改动。
