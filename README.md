# AiCoding

<p align="center">
  <a href="README_CN.md">中文 README_CN.md</a> |
  <a href="#english">English</a> |
  <a href="CHANGELOG.md">更新日志 / CHANGELOG</a> |
  <a href="#environment-preview">环境预览 / Environment</a>
</p>

[![Version](https://img.shields.io/badge/Version-0.1.0-2ea44f)](config/codex-kit.json)
[![Verify](https://img.shields.io/badge/verify--codex--kit-required-2ea44f)](#maintenance-commands)
[![PowerShell](https://img.shields.io/badge/PowerShell-7-5391FE)](#environment-preview)
[![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB)](#environment-preview)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue)](LICENSE)

AiCoding 是本地 AI 辅助嵌入式开发平台仓库。它集成 CodingKit 资产、仓库治理、版本锁定的 Codex plugin kit、Agent Patch Kit 和 AI Debug Repair Kit，用于更可控的 Agent 编辑、更清晰的 Git 同步规则，以及默认非侵入式的嵌入式调试辅助。

<a id="environment-preview"></a>
## 环境预览 / Environment Preview

| 项目 / Item | 当前规则 / Current rule | 快速跳转 / Link |
|---|---|---|
| 运行 Shell / Shell | 默认 PowerShell 7；Windows PowerShell 5.1 仅做兼容性门禁 / PowerShell 7 by default; Windows PowerShell 5.1 only for compatibility gates | [维护命令](#maintenance-commands) |
| Plugin 安装 / Plugin install | 通过本地 Marketplace 安装 `aicoding@aicoding-platform` / install through local Marketplace | [快速开始](#quick-start) |
| Agent Patch Kit | `apatch` 安全补丁、扫描、事务快照和 Markdown 链接检查 / safe patching, scan, transaction snapshots, Markdown checks | [本地 Agent Kit](#local-agent-kits) |
| AI Debug Repair Kit | `airepair` build/test repair 与 TI DSS 只读 scaffold / build-test repair plus TI DSS read-only scaffold | [本地 Agent Kit](#local-agent-kits) |
| Git 治理 / Git governance | README、CHANGELOG、Tag、Release、About 默认中文在前、英文在后 / Chinese first, English second | [Git 治理标准](#git-governance-standard) |

<a id="quick-start"></a>
## 快速开始 / Quick Start

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1
```

`install-codex-kit.ps1` 会创建本地 Marketplace 链接 `plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding`，在 Codex plugin CLI 可用时注册 `aicoding-platform`，并安装 `aicoding@aicoding-platform`。该链接是本机生成状态，已被 Git 忽略。

<a id="local-agent-kits"></a>
## 本地 Agent Kit / Local Agent Kits

AiCoding 通过本地 Marketplace 发布仓库级 Agent Kit：

- Agent Patch Kit：`aicoding-agent-patch-kit`，来源为 `dist/agent-patch-kit/plugins/AiCodingAgentPatch`，提供 `apatch` 安全补丁流程、状态门禁、固定字符串扫描/替换、事务快照、Markdown 链接检查和 patch summary。
- AI Debug Repair Kit：`aicoding-ai-debug-repair-kit`，来源为 `dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit`，提供 `airepair`，用于有边界的 build/test repair loop 和只读嵌入式 debug 辅助。v0.4.0 包含 `ti_dss` TI XDS/CCS DSS scaffold，以及受 policy 限制的 J-Link 侵入式操作 stub。

环境要求：

- 默认使用 PowerShell 7（`pwsh`）执行仓库安装、验证、状态、更新和文档检查；Windows PowerShell 5.1 只用于明确的兼容性门禁。同时需要 Git、Python 3.10+ 和 Codex plugin Marketplace 流程。
- Agent Patch Kit 使用用户态 `apatch` CLI。验证命令：`apatch install doctor`、`apatch brief --format md`、`apatch state status`。
- AI Debug Repair Kit 使用用户态 `ai-debug-repair-kit` Python 包。验证命令：`python -m ai_debug_repair.cli version --output json`、`python -m ai_debug_repair.cli doctor --output json`、`pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-ai-debug-repair-kit.ps1 -Json`。
- TI DSP debug 需要 TI CCS/DSS，例如 `C:\ti\ccs1281\ccs\ccs_base\scripting\bin\dss.bat`，还需要 XDS 仿真器和目标 `.ccxml` 后才能执行真实硬件访问。默认 profile 保持非侵入式：不 reset、不 halt、不 run、不 flash、不写内存/寄存器/表达式。

常用命令：

```powershell
apatch status
apatch scan "README.md" --fixed
apatch summary

python -m ai_debug_repair.cli dss capabilities --output json
python -m ai_debug_repair.cli dss profile-template --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss validate-profile --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss doctor --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
```

`.ai-debug-repair/` 下的 profile、run script、session log 属于本机运行状态，默认不提交到 Git。只有明确作为测试 fixture 时，才应单独纳入版本管理。

## 仓库角色 / Repository Roles

- `CodingKit/agents/skills` 是指向 `https://github.com/JiaxI2/Codex-Skills.git` 的 submodule。
- `CodingKit/agents/skills/plugins/AiCoding` 是安装用的 Codex plugin package。
- `aicoding-user-skill-creator` 打包在 AiCoding plugin 中，显示为 User-Skill-Creator；系统 `skill-creator` 保持独立。
- `.agents/plugins/marketplace.json` 是 AiCoding platform Marketplace entry。
- `config/codex-kit.json` 定义 CodingKit 资产发现和安装规则。
- `.githooks/` 是仓库级 Git hook；Codex hook 位于 plugin 内，需要通过 `/hooks` 审核。
- AiCoding plugin 内置 SDD、MVP、BDD、架构优先、TDD fallback 和文档同步 workflow skills；Superpowers 可复用但不是硬依赖。

## CodingKit 资产 / CodingKit Assets

```text
CodingKit/examples
CodingKit/modules
CodingKit/platforms
CodingKit/tests
CodingKit/tools
```

这些目录是平台资产，不复制进 Codex plugin。Skill 和工具通过 `config/codex-kit.json`、`AICODING_HOME`、安装状态、PATH、项目发现或 MCP 发现它们。

## Standalone Skills

AiCoding 区分 plugin bundled skills 和个人 standalone skills：

- Bundled `aicoding-*` skills 通过 AiCoding Codex Plugin 安装，由 Codex plugin cache 管理。
- 个人或下载的 standalone skills 备份在 `Codex-Skills`，默认按 profile 以 junction 方式安装到 `%USERPROFILE%\.agents\skills`。
- `scripts/set-codex-skill-profile.ps1 -Profile full -DryRun -Json` 可查看完整 standalone skill 安装计划。
- 仅当兼容流程明确需要 `%USERPROFILE%\.codex\skills` 时才使用 `-StandaloneRoot codex`；默认是 `-StandaloneRoot agents`。
- 兼容运行时可以保留 `%USERPROFILE%\.codex\skills\.system` 和部分 standalone junction，但 `aicoding-*` 只能来自已安装的 AiCoding plugin。

<a id="git-governance-standard"></a>
## Git 治理标准 / Git Governance Standard

所有 AiCoding 管理的 Git 仓库都必须在 README 或等价治理文档中写明分支、环境、提交类型、Release 说明和双语文档规则。

- 分支：`main` 或 `master` 是稳定生产分支，除批准的 release/hotfix 集成外不得直接改代码；`develop` 是 DEV 集成分支；`feature/<scope>` 从 `develop` 创建；存在共享测试环境时 `test` 对应 FAT；`release/<version>` 对应 UAT/预上线；`hotfix/<scope>` 从 `main` 创建，并回合到 `main` 和 `develop`。
- 环境：`DEV` 用于开发调试，`FAT` 用于功能验收测试，`UAT` 用于用户验收/预生产，`PRO` 用于生产。
- 提交类型：`feat` 新增功能，`fix` 修复 bug，`docs` 仅文档变更，`style` 仅格式/空白等不影响语义的变更，`refactor` 既不修 bug 也不加功能的代码重构，`perf` 性能改进，`test` 添加或修正测试，`build` 构建或打包行为，`ci` 自动化变更，`chore` 辅助工具或维护文件变更。
- 单次提交：一个 commit 只放一类变更，议题不超过 3 个，并使用 `feat(scope): summary` 这类 typed subject。
- 双语规则：README 默认中文优先，顶部必须保留可见的 `README_CN.md` 与 English 快速切换；CHANGELOG、Tag、GitHub Release 和 GitHub About 描述默认中文在前、英文在后。
- Release：Tag 和 GitHub Release 必须按类型汇总本次包含的全部提交，说明本次 release 主类型，并写清具体影响。

<a id="maintenance-commands"></a>
## 维护命令 / Maintenance Commands

```powershell
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/status-codex-kit.ps1 -Json
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/update-codex-kit.ps1 -DryRun
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all
```

不要在 AiCoding submodule 内重建 `plugins/AiCoding`。只有在 Codex-Skills 已构建、验证、提交并推送 plugin package 后，AiCoding 才更新 submodule 指针。

## 文档 / Documentation

- [中文 README](README_CN.md)
- [Codex Kit Architecture](docs/CODEX_KIT_ARCHITECTURE.md)
- [Maintenance Method](docs/MAINTENANCE_METHOD.md)
- [CodingKit](CodingKit/README.md)
- [CHANGELOG](CHANGELOG.md)
- [Repository Governance](.github/repository-governance.toml)

<a id="english"></a>
## English

AiCoding is a platform repository for local AI-assisted embedded development. It integrates CodingKit assets, repository governance, a version-locked Codex plugin kit, Agent Patch Kit, and AI Debug Repair Kit.

The default documentation order is Chinese first and English second. Use the language switch at the top for the dedicated Chinese README, the English section, changelog, and environment preview. PowerShell 7 is the default shell for install, verify, status, update, and documentation checks; Windows PowerShell 5.1 is reserved for explicit compatibility gates.

The installed AiCoding plugin exposes `aicoding-*` workflow skills through the Codex plugin cache. CodingKit external assets remain outside the plugin and are discovered through `AICODING_HOME`, install state, PATH, project discovery, or MCP.
