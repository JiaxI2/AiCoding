# Changelog

## [Unreleased]

## [0.8.0] - 2026-07-15

- **feat(cstyle)**: 将 C UserStyle Kit 1.2.0 作为 `CodingKit/tools` 自包含 Go module 纳入平台，保留唯一 `skill c99-standard-c` 用户入口，并新增 `fast`/`full` 结构化验证。 / Integrates C UserStyle Kit 1.2.0 through the existing C99 Skill route with structured fast/full verification.

- **test(governance)**: 将真实 C Kit 快速验证加入 Kit registry、Taskfile、全局 Smoke/Full/Release 测试和源码事实检查，同时保持 skills submodule、插件与缓存不变。 / Adds C Kit verification to repository governance without modifying the skills submodule or plugin runtime.

- **fix(pwsh)**: 修复专项脚本从 `tools/specialty` 定位仓库根的旧路径错误，使 Codex Kit 与 runtime Skill 审计可在当前目录架构中真实执行。 / Fixes repository-root discovery for specialty Codex Kit and runtime Skill audits.

- **docs(reference)**: 随 C Kit 发布完整 PDF、规范化 Markdown、raw 转换件、139 条规则目录、黄金 demo、高级可见样例和用户可编辑 VS Code 风格 snippets；以上参考资产按用户明确授权允许公开分发。 / Publishes the complete reference and customization assets under explicit user authorization.

## [0.7.0] - 2026-07-10

- **feat(governance)**: 新增可复用模块登记与证据门禁；以 Go CLI 接入 Skill Verify、hook、CI、DocSync 和 lifecycle，首轮仅采用可回滚的原生实现。 / Adds a reusable-module evidence gate integrated with the Go control plane.

- **ci**: 修复 Windows GitHub Actions 的相对 CLI 路径，避免 `cmd` 将 `bin/aicoding.exe` 解析为命令加参数。 / Fixes Go CLI invocation from Windows CI.

## [0.6.0] - 2026-07-10

- **refactor(layout)**: 收敛文档分类、Plan Mode 产物路径与工具路径，新增 IA 导航配置和生成的目录导航 hub。

- **feat(test)**: 新增全局测试器，并提供 `test full`、`test release` 与 `test latest` 的结构化验证和报告。

- **docs(readme)**: README 只保留平台/kit/plugin/skill 母级架构入口，具体 leaf skill 命令下沉到命令文档；补充 clang-format 17.0.2 badge 和 README 可见性规则。
- **refactor(cli)**: 默认用户入口统一为 `bin/aicoding.exe smoke|ci|full|release gate` 和 `skill c99-standard-c ...`。
- **feat(runner)**: 新增 `internal/runner` 并发 Plan，支持按任务 ID 快速新增、移除和组合只读验证任务。
- **docs**: README、命令文档、架构文档、PowerShell 边界文档、Tag policy 和 Release policy 只描述当前 main 的可观测标准。
- **chore(pwsh)**: Go 默认控制面之外只保留 PowerShell 专项质量、安全、Plan Mode、外部 skill、tag planning / overlay compatibility 和硬件/工具链边界脚本。

[Unreleased]: https://github.com/JiaxI2/AiCoding/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/JiaxI2/AiCoding/compare/v0.7.0...v0.8.0
