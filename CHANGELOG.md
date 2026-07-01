# CHANGELOG

本仓库使用 `[Unreleased]` 记录普通开发提交。每条记录必须标注提交类型。

## [Unreleased]

### Commit Type

- 本轮已提交类型：`feat(coding-kit)`、`feat(docs-sync)`、`docs(repo)`、`feat(git-governance)`、`feat(ai-debug-repair-kit)`。

### Added
- **feat(coding-kit)**：新增 `CodingKit/modules/common/ring_buffer` C99 环形缓冲区模块，提供外部存储、无动态内存、C28x 16-bit 字节寻址兼容的 `RingBuf_Init/Reset/Used/Free/Write/Read/ReadByte` 接口；add a `CodingKit/modules/common/ring_buffer` C99 ring buffer module with caller-owned storage, no dynamic allocation, C28x 16-bit byte-addressing compatibility, and `RingBuf_Init/Reset/Used/Free/Write/Read/ReadByte` APIs.
- **feat(ai-debug-repair-kit)**：新增严格 DSS `attach-readonly` CLI 模板和授权 `flash-debug` 验证路径，固化 full-core/no-GEL attach、`symbol.load` + `evaluateToString`、地址 `memory.readData` 监控、DSS stderr 异常判定和字符串字面量安全扫描，并完成 F28388D CPU1 Flash 擦除、烧录、verify、复位运行及 `txMsgData` 实时变化验证；add strict DSS `attach-readonly` CLI templates and an authorized `flash-debug` validation path, covering full-core/no-GEL attach, `symbol.load` plus `evaluateToString`, address `memory.readData` monitoring, DSS stderr exception detection, string-literal-aware safety scanning, and verified F28388D CPU1 Flash erase/program/verify/reset/run with live `txMsgData` changes.
- **fix(ai-debug-repair-kit)**：`status-ai-debug-repair-kit.ps1` 在 `airepair.exe` 不在 PATH 时会回退探测 Python user Scripts 目录，避免已安装 CLI 被报告为 `null`；make `status-ai-debug-repair-kit.ps1` fall back to the Python user Scripts directory when `airepair.exe` is not on PATH, so an installed CLI is not reported as `null`.
- **feat(ai-debug-repair-kit)**：升级 AI Debug Repair Kit 到 v0.4.1，新增固定 `airepair dss` connect/core-list/monitor-address/monitor-symbol/find-changing-symbol/report 工作流、DSS session evidence、JSON safety/capability/evidence 输出和 forbidden token 扫描，默认继续禁止 reset/halt/run/loadProgram/flash/erase/write-memory，并通过 v0.4.1 包内 verify、12 项 pytest、部署后 verify 和 Codex plugin validator 验证；upgrade AI Debug Repair Kit to v0.4.1 with fixed `airepair dss` connect/core-list/monitor-address/monitor-symbol/find-changing-symbol/report workflows, DSS session evidence, JSON safety/capability/evidence output, forbidden-token scanning, default denial of reset/halt/run/loadProgram/flash/erase/write-memory, and verified package verify, 12 pytest cases, installed verify, and Codex plugin validation.
- **feat(powershell-skill-kit)**：集成 Codex Agent PowerShell Skill Kit v1.2.1，新增 package-only 子 kit、Marketplace 条目、生命周期脚本、PS7 AST/Safety/PSScriptAnalyzer gate、runtime mirror 所有权标记，并修复 dry-run、PowerShell 7.5 Generic.List 序列化和 `$count:` rewrite 阻塞测试；integrate Codex Agent PowerShell Skill Kit v1.2.1 with package-only sub-kit, Marketplace entry, lifecycle scripts, PS7 AST/Safety/PSScriptAnalyzer gates, runtime-mirror ownership markers, and fixes for dry-run, PowerShell 7.5 Generic.List serialization, and `$count:` rewrite blocking tests.
- **docs(repo)**：新增 `README_EN.md` 独立英文入口，并把 README 顶部 English 快速切换从页内锚点改为文件跳转；add a standalone `README_EN.md` English entry and change the top English switch from an in-page anchor to a file-level README link.
- **docs(repo)**：将 `README.md` 改为中文优先的双语入口，顶部保留 `README_CN.md`/English 快速切换，并新增可点击环境预览；make `README.md` a Chinese-first bilingual entry with top `README_CN.md`/English switching and a clickable environment preview.
- **chore(git-governance)**：把 README 顶部双语切换、CHANGELOG 双语、GitHub About 双语和中文优先顺序写入治理配置、lint 与 Agent Patch Kit 规则；codify README language switch, bilingual CHANGELOG, bilingual GitHub About, and Chinese-first ordering in governance config, lint, and Agent Patch Kit rules.
- **feat(ai-debug-repair-kit)**：更新 README/README_CN 和 Git governance 规则，将 `README_CN.md` 作为 GitHub About/Homepage 中文入口而不是英文 README 顶部链接，并记录 Agent Patch Kit 与 AI Debug Repair Kit 的环境、安装和使用说明；document Agent Patch Kit and AI Debug Repair Kit setup and usage, and route `README_CN.md` through GitHub About/Homepage governance instead of an English README top link.
- **feat(ai-debug-repair-kit)**：集成 AiCoding AI Debug Repair Kit v0.4.0，新增 `ti_dss` 非侵入式 TI XDS/CCS DSS backend scaffold、`airepair dss` 只读命令族、J-Link 侵入式操作 policy-gated stubs、TI DSS/J-Link 安全策略文档与示例 profile，并通过 v0.4.0 包内 pytest、部署后 verify、Codex plugin validator 和 TI DSS capabilities 验证；integrate AiCoding AI Debug Repair Kit v0.4.0 with non-invasive `ti_dss` backend scaffold, `airepair dss` read-only commands, policy-gated J-Link invasive stubs, safety docs, example profiles, and verified package pytest, installed verify, Codex plugin validation, and TI DSS capabilities.
- **feat(ai-debug-repair-kit)**：集成 AiCoding AI Debug Repair Kit v0.3.2，本地 Marketplace 新增 `aicoding-ai-debug-repair-kit`，发布 `airepair` CLI、三项调试/修复 Skill、PowerShell 安装/验证/状态/卸载脚本，并通过 Windows PowerShell 5.1、PowerShell 7、pytest 与 Codex plugin validator 验证；integrate AiCoding AI Debug Repair Kit v0.3.2 with local Marketplace entry, `airepair` CLI, debug/repair skills, PowerShell lifecycle scripts, and verified Windows PowerShell 5.1, PowerShell 7, pytest, and Codex plugin validation.
- **fix(agent-patch-kit)**：升级 Agent Patch Kit 到 v2.2，修复 v2.1 editable pip 安装依赖原始解压目录的问题，改为 non-editable user-mode wheel 安装，并重新部署 repo-scoped Skill 与 marketplace sidecar；upgrade Agent Patch Kit to v2.2, fix the v2.1 editable-install source-directory dependency, use non-editable user-mode wheel install, and redeploy repo-scoped Skill plus marketplace sidecar.
- **feat(agent-patch-kit)**：部署 Agent Patch Kit v2.1 为 repo-scoped Skill，新增项目配置、Agent snippet、AiCoding marketplace sidecar 和本地 plugin 条目，并记录安装前后 Agent 上下文/token 成本对比；deploy Agent Patch Kit v2.1 as a repo-scoped Skill with project config, Agent snippet, AiCoding marketplace sidecar, local plugin entry, and before/after agent context-token evaluation.
- **feat(git-governance)**：将 README 中文链接、Git 治理标准、commit type 和 Release typed summary 规则接入 `scripts/lint-git-governance.ps1`，通过 Git hook 机器检查；enforce README Chinese-link, Git governance, commit type, and release typed-summary rules through the Git hook lint.
- **docs(repo)**：新增 Apache-2.0 `LICENSE`、`CONTRIBUTING.md`、`SECURITY.md` 和 `CITATION.cff`，补齐 GitHub About 侧栏可识别文件；add repository metadata files recognized by GitHub About.
- **docs(repo)**：在 README/README_CN 中写明 AiCoding Git 治理标准，包括分支命名、环境映射、commit type、单次提交约束和 Release typed summary；document branch/environment, commit type, single-commit, and release typed-summary standards in README files.
- **feat(coding-kit)**：更新 `CodingKit/agents/skills` submodule 到 `283b3a0`，包含 AiCoding SDD/MVP/BDD/架构优先/TDD fallback/文档同步 workflow skills，并保持 Superpowers 为可选增强；update the Codex-Skills submodule to `283b3a0` with standalone-capable AiCoding workflow skills while keeping Superpowers optional.
- **feat(docs-sync)**：新增 `config/docs-sync.policy.json`、`scripts/check-documentation-sync.ps1`、`scripts/install-docsync-hook.ps1`、`.github/workflows/docs-sync.yml`，并把 docs-sync 接入 `.githooks/pre-commit`；add documentation synchronization policy, checker, installer, CI workflow, and pre-commit integration.
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
