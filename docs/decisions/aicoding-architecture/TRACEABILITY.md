# 可追溯性：正交内核闭环

| 需求 / 决策 | 实现证据 | 验证证据 |
|---|---|---|
| Git 式 facts 与 movable intent 分离 | Registry/manifest `CatalogSnapshot`、`ExecutionPlan` | registry/runner contracts |
| 稳定边界优先于无限优化 | Accepted/Frozen 文档、明确解冻条件 | Plan Mode、architecture docs review |
| No God Core | 六个正交职责、domain-owned state | dependency `goPackageBoundaries` |
| 模块可独立优化 | production import 禁令、模块测试矩阵 | module tests + consumer regressions |
| Kit/MCP 内容树 | `internal/registry` + domain catalog loaders | manifest-only digest tests、detached-value tests |
| ExecutionPlan 第二消费者 | lifecycle adapter selection -> plan | lifecycle plan digest tests |
| 静态扩展 | Adapter Descriptor/Catalog，三领域 static function | lifecycle catalog tests |
| Agent 调用接口 | process + `report.Result` JSON | real CLI plan/list/status probes |
| External Skill lifecycle | runtime registry/source digest + bounded specialty adapter | runtime plan/audit、Full/Release |
| MCP lifecycle | component catalog + MCP domain state/config ownership | MCP tests、plan、Full/Release |
| 安装/更新/同步/卸载 | action/effect 与 domain contract | lifecycle tests、dry-run profiles |
| 可追踪证据 | `catalogDigest`、`inputDigest`、`planDigest` | report/schema/CLI contracts |
| 不引入 speculative graph/journal/API/C | architecture rejection + evidence thresholds | doc review、dependency gates |
| 实现 identity 不编码版本 | stable package/path/ID | dependency governance |

## 公共契约同步

- `docs/architecture/AICODING_CORE_ARCHITECTURE.md`：正交模块、闭环、冻结与测试半径权威。
- `docs/architecture/CLI_MCP_CONTROL_PLANE.md`：仓库生命周期、Agent API 与 MCP/Skill 边界。
- `docs/architecture/EXTENSION_ADAPTER_CONTRACT.md`：真实 descriptor、输入/输出/state contract。
- `docs/ARCHITECTURE_OVERVIEW.md`：同步当前实现权威与内容树/plan/adapter 关系。
- `docs/COMMANDS.md`：同步 Agent lifecycle 用法和 digest 加法字段。
- `docs/operations/testing/REPORT_SCHEMA.md` 与 CLI schema：同步 input/plan digest。
- `config/dependency-governance.json` 与 schema：增加 orthogonal production import gate。
- README 三件套已审查：本批未新增顶层产品入口、默认工具链或 badge authority，无需改动。

## 验收记录

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `go test -race` 覆盖 registry、runner、report、Kit、MCP、lifecycle、governance、
  repohealth 与 CLI：通过。
- 真实 `kit list`、`mcp list`、Kit/MCP/runtime Skill lifecycle plan 和 Kit status：均返回
  `sha256:` input/plan/catalog evidence；所有 plan/status 无写入。
- `governance dependencies`：通过，`orthogonal Go package boundaries` check 为 PASS。
- DocSync CI、governance lint/layout、hooks、repo-text、release-notes：通过。
- 变更 Markdown link validation：15 个文件通过。
- Plan Mode verification：通过。
- Doctor：4 PASS / 1 WARN / 0 FAIL；warning 仅为 worktree 未安装 visio-mcp。
- Verify Smoke：11 PASS / 1 WARN / 0 FAIL；warning 仅为当前 worktree 缺少可选
  `CodingKit/examples`、`CodingKit/platforms` 目录。
- Smoke：38 PASS / 0 WARN / 0 FAIL / 16 SKIP。
- Full：52 PASS / 0 WARN / 0 FAIL / 2 SKIP。
- Release：53 PASS / 0 WARN / 0 FAIL / 1 SKIP。
- `verify-codex-kit.ps1 -Json`：通过，Full 52 PASS / 0 WARN / 0 FAIL / 2 SKIP；唯一 warning
  为兼容入口 `CLI_DEPRECATED`。
- `git diff --check`：通过；Codex-Skills submodule 保持 clean 且 gitlink 未变化。

## 需求结论

用户要求的 Git/正交设计哲学、有限架构闭环、Skill/MCP lifecycle、Agent 接口、模块化局部
验证、worktree 实现与全量验收均有直接实现和运行证据。总体架构满足冻结条件；后续新增
component/Skill 属于功能扩展，模块内部性能优化属于维护，不再保留无限架构迁移清单。
