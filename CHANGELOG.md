# CHANGELOG

本仓库使用 `[Unreleased]` 记录普通开发提交。每条记录必须标注提交类型。

## [Unreleased]

### Commit Type

- 本轮已提交类型：`feat(coding-kit)`、`chore(coding-kit)`。

### Added
- **feat(coding-kit)**：`install-codex-kit.ps1` 现在可创建本地 Marketplace junction，并通过 Codex plugin CLI 注册 `aicoding-platform`、安装 `aicoding@aicoding-platform`；add reproducible local Marketplace link creation and Codex CLI plugin registration/installation.
- **chore(runtime)**：完成本机 runtime 迁移闭环：旧 `.codex\skills` 源码暴露已备份，`.system` 保留，standalone skills 以 junction 从 `F:\Study\AI\Codex-Skills` 暴露，`aicoding-*` 只来自 AiCoding Plugin；complete local runtime migration with backup, preserved `.system`, standalone junctions, and plugin-only `aicoding-*` exposure.
- **feat(coding-kit)**：补全 standalone skill registry，将当前已备份的个人/下载 skill 纳入 `full` profile 安装计划；add a standalone skill registry and include backed-up personal/downloaded skills in the `full` profile plan.
- **feat(tooling)**：`set-codex-skill-profile.ps1` 支持 `-StandaloneRoot agents|codex`、幂等 junction 创建和 dry-run 安装预览；support selectable standalone install roots, idempotent junction creation, and dry-run install previews.
- **docs**：新增 `README_CN.md`，并把 README、CHANGELOG、Tag、Release 维护策略调整为中英双语闭环；add Chinese README documentation and align README, CHANGELOG, Tag, and Release governance with bilingual operation.

- **chore(coding-kit)**：更新 `CodingKit/agents/skills` submodule 到 `61d2176`，包含 `aicoding-kit-maintenance`、`aicoding-user-skill-creator` 以及 `fix(tooling)` 的 BUILDINFO 非自引用漂移检查修复。
- **docs**：明确 `aicoding-git-governance` 负责 Git/README/CHANGELOG/发布治理，`aicoding-kit-maintenance` 负责 kit 生命周期；新增 `aicoding-user-skill-creator`（User-Skill-Creator）与系统 `skill-creator` 的共存边界。
- **feat**：新增 runtime skill exposure 配置和 `audit-runtime-skills.ps1`、`set-codex-skill-profile.ps1`、`migrate-skill-root.ps1`、`restore-legacy-skill-root.ps1`，用于审计重复 Skill、预演 Profile 切换、迁移和回滚。
- **docs**：补充 Runtime Skill Exposure Policy，明确 `Codex-Skills` 源码仓库不得作为用户级 Skill Root，正常模式只通过 AiCoding Plugin 暴露 `aicoding-*`。
- **docs**：新增 `AGENTS.md`、`CodingKit/AGENTS.md`、`docs/CODEX_KIT_ARCHITECTURE.md` 和 `docs/MAINTENANCE_METHOD.md`，形成“AGENTS 边界 → 维护 Skill → docs → config/scripts → CI/Git hooks”的后续维护管理方法。
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
