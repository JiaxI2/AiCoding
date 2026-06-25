# CHANGELOG

本仓库使用 `[Unreleased]` 记录普通开发提交。每条记录必须标注提交类型。

## [Unreleased]

### Commit Type

- 本次提交主类型：`chore(git)`。

### Added

- **docs**：新增根 `README.md`，说明 AiCoding 仓库定位、submodule、快速开始和 Git 提交流程。
- **chore**：新增 `.github/repository-governance.toml`，记录分支、README、CHANGELOG、版本、发布、固件制品和自动化策略。
- **chore**：新增 `.githooks/pre-commit`、`.githooks/commit-msg` 和 `scripts/lint-git-governance.ps1`，通过 CLI 配置 hook 并检查提交类型与 CHANGELOG 更新。
- **chore**：新增 `.gitattributes`，固定 `.githooks/*` 使用 LF 行尾，避免 Git Bash 执行 hook 时受 CRLF 影响。
- **chore**：更新 `CodingKit/agents/skills` submodule 到 `Codex-Skills` 的 `Git-Skill` 最新提交。

[Unreleased]: https://github.com/JiaxI2/AiCoding/compare/4f71d521df90461cae2b39fba5bcac47f1b5ad76...HEAD