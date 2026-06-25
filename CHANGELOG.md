# CHANGELOG

本仓库使用 `[Unreleased]` 记录普通开发提交。每条记录必须标注提交类型。

## [Unreleased]

### Commit Type

- 本次提交主类型：`chore(git)`。

### Added

- **docs**：新增根 `README.md`，说明 AiCoding 仓库定位、submodule、快速开始和 Git 提交流程。
- **docs**：新增 CodingKit/README.md，说明 CodingKit/agents/skills submodule、plugins/AiCoding kit 入口和新电脑安装流程。
- **chore**：新增 scripts/install-aicoding-codex-kit.ps1，把 CodingKit/agents/skills/plugins/AiCoding 链接或复制到本机 Codex skills 插件目录，并配置仓库 Git hooks。
- **chore**：新增 `.github/repository-governance.toml`，记录分支、README、CHANGELOG、版本、发布、固件制品和自动化策略。
- **chore**：新增 `.githooks/pre-commit`、`.githooks/commit-msg` 和 `scripts/lint-git-governance.ps1`，通过 CLI 配置 hook 并检查提交类型与 CHANGELOG 更新。
- **chore**：新增 `.gitattributes`，固定 `.githooks/*` 使用 LF 行尾，避免 Git Bash 执行 hook 时受 CRLF 影响。
- **chore**：更新 `CodingKit/agents/skills` submodule 到包含 `plugins/AiCoding` 的 `Codex-Skills` 最新提交。
- **feat**：新增 `.agents/plugins/marketplace.json` 和 `config/codex-kit.json`，将 AiCoding 定义为 Codex Marketplace 安装入口和 CodingKit 资产发现入口。
- **build**：新增 `scripts/install-codex-kit.ps1`、`scripts/update-codex-kit.ps1`、`scripts/status-codex-kit.ps1`、`scripts/uninstall-codex-kit.ps1`、`scripts/verify-codex-kit.ps1` 和共享 helper，支持安装、更新、状态、卸载和验证流程。
- **docs**：更新 `README.md` 与 `CodingKit/README.md`，明确 `examples/modules/platforms/tests/tools` 为平台资产层，AiCoding 不在 submodule 中重建 Plugin，Hook 只作为辅助约束。

[Unreleased]: https://github.com/JiaxI2/AiCoding/compare/4f71d521df90461cae2b39fba5bcac47f1b5ad76...HEAD