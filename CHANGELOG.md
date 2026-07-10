# Changelog

## [Unreleased]

- **ci**: 修复 Windows GitHub Actions 的相对 CLI 路径，避免 `cmd` 将 `bin/aicoding.exe` 解析为命令加参数。

## [0.6.0] - 2026-07-10

- **refactor(layout)**: 收敛文档分类、Plan Mode 产物路径与工具路径，新增 IA 导航配置和生成的目录导航 hub。

- **feat(test)**: 新增全局测试器，并提供 `test full`、`test release` 与 `test latest` 的结构化验证和报告。

- **docs(readme)**: README 只保留平台/kit/plugin/skill 母级架构入口，具体 leaf skill 命令下沉到命令文档；补充 clang-format 17.0.2 badge 和 README 可见性规则。
- **refactor(cli)**: 默认用户入口统一为 `bin/aicoding.exe smoke|ci|full|release gate` 和 `skill c99-standard-c ...`。
- **feat(runner)**: 新增 `internal/runner` 并发 Plan，支持按任务 ID 快速新增、移除和组合只读验证任务。
- **docs**: README、命令文档、架构文档、PowerShell 边界文档、Tag policy 和 Release policy 只描述当前 main 的可观测标准。
- **chore(pwsh)**: Go 默认控制面之外只保留 PowerShell 专项质量、安全、Plan Mode、外部 skill、tag planning / overlay compatibility 和硬件/工具链边界脚本。
