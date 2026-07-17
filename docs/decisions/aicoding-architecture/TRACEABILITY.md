# 可追溯性：AiCoding 内核与扩展图架构

| 需求 / 决策 | 架构章节 | 验证 |
|---|---|---|
| Git 式稳定基础 | 稳定内核、控制面边界 | Go tests、contract tests |
| 扩展建立在基础上 | 扩展契约、Porcelain | dependency governance、adapter tests |
| 可玩性 | Porcelain 与可玩性 | profile/plan tests |
| 高性能 | 性能模型 | `doctor perf`、benchmarks |
| 实现 identity 不编码版本 | 架构结论、明确拒绝 | dependency governance |
| 单一控制面 | 控制面边界 | Smoke / Full / Release |
| source/runtime 分离 | Source、distribution 与 runtime | runtime Skill audit |
| ExecutionPlan 成为核心对象 | 稳定内核、已落地的核心对象 | `internal/runner` unit tests、pre-commit contract |
| Registry Snapshot + Digest | 稳定内核、Registry 与命令目录边界 | `internal/registry`、Kit/MCP loader tests、MCP inventory JSON |
| Typed Command Catalog | 控制面边界、当前热点 | CLI catalog/contract tests、`aicoding --help` |

## 公共契约审查

- `README.md`、`README_CN.md`、`README_EN.md`：已审查；本批没有新增顶层产品入口、默认运行体系或 badge authority，不需要改动。
- `docs/COMMANDS.md`：记录 typed catalog 权威与 MCP `registryDigest` 加法字段。
- `docs/ARCHITECTURE_OVERVIEW.md`：同步三个已落地对象及尚未完成的边界。
- 测试文档：同步 plan/digest/catalog 单元契约与 Full/Release 验证范围。

## 实际验证记录

- `go test ./...`、`go vet ./...`：通过。
- `go test -race ./internal/runner ./internal/registry ./internal/kit ./internal/mcpcontrol ./internal/cli`：通过。
- 真实二进制：`--help` 由 catalog 渲染；`version` 读取 manifest 元数据；`mcp list --json` 输出 `registryDigest`；`hook pre-commit --json` 使用 `ExecutionPlan` 并通过。
- 产品门禁：Smoke 38 PASS / 0 WARN / 0 FAIL / 16 SKIP；Full 52 / 0 / 0 / 2；Release 53 / 0 / 0 / 1。
- `verify-codex-kit.ps1 -Json`：通过，复用 Full 得到 52 / 0 / 0 / 2；唯一 warning 为兼容入口的 `CLI_DEPRECATED`。
- Doctor、Verify Smoke、DocSync CI、dependency/layout/lint governance、lifecycle install/update plan、status、Skill Smoke、Hook/repo-text/release-notes、Plan Mode 24 checks：通过。
- Markdown links：116 个目标、50 个唯一链接、63 OK、0 error、53 个 offline excluded。
- 同机交替 A/B：candidate `version` p50 48.2 ms / p95 78.5 ms，`main` 为 49.2 / 77.8；candidate `kit list` 43.6 / 97.1，`main` 为 44.5 / 115.3；优化后 help p50 50.8 vs 53.9，p95 138.8 vs 122.0，相对回退 13.8%，未超过 20% 阻断线。该轮 host 抖动使两者绝对 p95 同时超过 100 ms，因此只用配对相对值判定回退。
- 稳定 identity 审计：Fast Path cache 已收敛为 `.aicoding/cache/fast-path`；实现范围内不再匹配 `fast-path-vN`、`fast_path_vN` 或 `fastpathvN`。
