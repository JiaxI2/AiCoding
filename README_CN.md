# AiCoding

<p align="center">
  <a href="README_CN.md">中文 README_CN.md</a> |
  <a href="README_EN.md">English README_EN.md</a> |
  <a href="CHANGELOG.md">更新日志 / CHANGELOG</a> |
  <a href="README.md#environment-preview">环境预览 / Environment</a>
</p>

[![Version](https://img.shields.io/badge/Version-0.1.0-2ea44f)](config/codex-kit.json)
[![PowerShell](https://img.shields.io/badge/PowerShell-7-5391FE)](README.md#environment-preview)
[![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB)](README.md#environment-preview)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue)](LICENSE)

AiCoding 是本地 AI 辅助嵌入式开发平台仓库。它不直接维护 Skill 源码，而是通过 `CodingKit/agents/skills` submodule 锁定 `Codex-Skills` 的已验证版本，并提供安装、更新、状态、卸载、运行时审计、CodingKit 资产入口、Agent Patch Kit、AI Debug Repair Kit 和 AiCoding Agent Dev Kit。

## 快速环境预览 / Environment Preview

| 项目 | 当前规则 | 跳转 |
|---|---|---|
| Shell | 默认 PowerShell 7；Windows PowerShell 5.1 只做兼容性门禁 | [维护命令](#commands) |
| Plugin | 通过本地 Marketplace 安装 `aicoding@aicoding-platform` | [快速开始](#quick-start) |
| Agent Patch Kit | `apatch` 安全补丁、扫描、事务快照、Markdown 链接检查 | [本地 Agent Kit](#local-agent-kits) |
| AI Debug Repair Kit | `airepair` repair loop、TI DSS 只读 scaffold、J-Link policy stubs | [本地 Agent Kit](#local-agent-kits) |
| AiCoding Agent Dev Kit | `aicoding-agent-kit` 需求澄清、方案矩阵、Plan Mode、Spec/TDD、顺序上下文加载和进度监控 | [本地 Agent Kit](#local-agent-kits) |
| Kit Lifecycle v2.0 | `scripts/aicoding-kit.ps1` 统一 Kit lifecycle 与 skill routing 入口 | [常用命令](#commands) |
| 文档治理 | README/CHANGELOG/Tag/Release/About 默认中文在前、英文在后 | [Git 治理标准](#git-governance-standard) |

<a id="quick-start"></a>
## 快速开始 / Quick Start

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1 -DryRun
```

真实安装 Plugin 时优先使用 Codex 的 Marketplace/plugin 机制。不要手工修改 Codex plugin cache。`install-codex-kit.ps1` 会创建本地 Marketplace 需要的 `plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding` junction，然后通过 Codex plugin CLI 注册 `aicoding-platform` 并安装 `aicoding@aicoding-platform`。`plugins/` 是本机生成状态，不提交到 Git。

<a id="local-agent-kits"></a>
## 本地 Agent Kit / Local Agent Kits

AiCoding 通过本地 Marketplace 发布三套仓库级 Agent Kit：

- Agent Patch Kit：Marketplace 名称为 `aicoding-agent-patch-kit`，来源为 `dist/agent-patch-kit/plugins/AiCodingAgentPatch`，提供 `apatch` 安全补丁流程、状态门禁、固定字符串扫描/替换、事务快照、Markdown 链接检查和 patch summary。
- AI Debug Repair Kit：Marketplace 名称为 `aicoding-ai-debug-repair-kit`，来源为 `dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit`，提供 `airepair`，用于有边界的 build/test repair loop 和默认非侵入式 TI DSS/XDS 只读 debug 辅助。v0.4.1 固定 `airepair dss` 工作流，提供 `connect-test`、`core-list`、`monitor-address`、`monitor-symbol`、`find-changing-symbol` 和 `report`，并保留受 policy 限制的 J-Link 侵入式操作 stub。
- AiCoding Agent Dev Kit：Marketplace 名称为 `aicoding-agent-dev-kit`，来源为 `dist/aicoding-agent-dev-kit/plugins/AiCodingAgentDevKit`，提供 `aicoding-agent-kit`，用于需求澄清、技术方案矩阵、Plan Mode overlay、Spec Pack、TDD 计划、顺序上下文加载、轻量决策记忆、Hook bridge 和 MVP 进度监控。Plan Mode overlay 文档见 [Agent Dev Kit Plan Mode](docs/AGENT_DEV_KIT_PLAN_MODE.md)、[Spec Kit Adaptation](docs/SPEC_KIT_ADAPTATION.md) 和 [Superpower Skill Adaptation](docs/SUPERPOWER_SKILL_ADAPTATION.md)。

环境要求：

- 默认使用 PowerShell 7（`pwsh`）执行仓库安装、验证、状态、更新和文档检查；Windows PowerShell 5.1 只用于明确的兼容性门禁。同时需要 Git、Python 3.10+ 和 Codex plugin Marketplace 流程。
- Agent Patch Kit 使用用户态 `apatch` CLI。验证命令：`apatch install doctor`、`apatch brief --format md`、`apatch state status`。
- AI Debug Repair Kit 使用用户态 `ai-debug-repair-kit` Python 包。验证命令：`python -m ai_debug_repair.cli version --output json`、`python -m ai_debug_repair.cli doctor --output json`、`pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-ai-debug-repair-kit.ps1 -Json`。
- AiCoding Agent Dev Kit 使用用户态 `aicoding-agent-dev-kit` Python 包。验证命令：`aicoding-agent-kit status --repo .`、`pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-aicoding-agent-dev-kit.ps1 -Json`、`pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/test-aicoding-agent-dev-kit.ps1 -Json`、`pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-dev-kit-plan-mode.ps1 -Json`、`pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-engineering-foundation.ps1 -Json`。
- TI DSP debug 需要 TI CCS/DSS，例如 `C:\ti\ccs1281\ccs\ccs_base\scripting\bin\dss.bat`，还需要 XDS 仿真器和目标 `.ccxml` 后才能执行真实硬件访问。默认 profile 保持非侵入式：不 reset、不 halt、不 run、不 flash、不写内存/寄存器/表达式。

统一生命周期入口：
推荐验证矩阵：

| 场景 | 命令 |
|---|---|
| 默认开发 | `pwsh scripts/verify-codex-kit.ps1` |
| 快速生命周期 | `pwsh scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json` |
| Skill 路由 | `pwsh scripts/aicoding-kit.ps1 verify-skills -All -Json` |
| 手动完整 | `pwsh scripts/aicoding-kit.ps1 test -All -Profile Full -Json` |
| 发布前 | `pwsh scripts/test-kit-fresh-clone.ps1 -Profile Release -Json` |
| 打包发布 | `pwsh scripts/aicoding-kit.ps1 export -All -Zip -Json` |
```powershell
pwsh scripts/aicoding-kit.ps1 list
pwsh scripts/aicoding-kit.ps1 status -All -Json
pwsh scripts/aicoding-kit.ps1 verify -All
pwsh scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json
pwsh scripts/aicoding-kit.ps1 skills -All -Json
pwsh scripts/aicoding-kit.ps1 verify-skills -All -Json
pwsh scripts/verify-common-code.ps1 -Json
pwsh scripts/verify-hooks.ps1 -Json
pwsh scripts/verify-agent-dev-kit-plan-mode.ps1 -Json
pwsh scripts/hooks/aef/plan-mode-gate.ps1 -Event manual -Mode warn -Json
pwsh scripts/hooks/aef/spec-artifact-gate.ps1 -Event manual -Mode warn -Json
pwsh scripts/verify-agent-engineering-foundation.ps1 -Json
pwsh scripts/aicoding-skill.ps1 sources -Json
pwsh scripts/aicoding-kit.ps1 export -All -Zip -DryRun
pwsh scripts/aicoding-kit.ps1 export -Kit aicoding-agent-dev-kit -Zip -Json
pwsh scripts/aicoding-kit.ps1 export -All -Zip -Json
pwsh scripts/test-kit-fresh-clone.ps1 -Profile Smoke -Json
```

Kit Lifecycle v2.0 使用 registry + manifest + adapter 固化统一平台入口，不重写旧脚本，不新增 `install-all.ps1`、`verify-all.ps1`、`test-all.ps1`、`export-all.ps1`、`update-all.ps1` 或 `uninstall-all.ps1`。`-All` 只遍历 `config/kit-registry.json` 中启用的 Kit，并复用单 Kit 的同一条 action 调度路径。v2.0 还固化 `skills`/`verify-skills`、Common registry、Hook registry、第三方 Skill source policy、自建 Skill 草稿/验证流程、真实 export/bundle 和 install-state 边界。Smoke 仍是默认 gate；`verify -All` 和 `test -All` 默认等价 Smoke，旧脚本完整 verify 需要显式 `-Profile Full`；Full/Release 必须显式传入。

Legacy adapter entry / 旧脚本入口：

以下旧脚本保留为 manifest adapter command 和 legacy adapter entry。推荐入口是上面的 `scripts/aicoding-kit.ps1`，旧脚本不再作为首选生命周期入口。

```powershell
apatch status
apatch scan "README.md" --fixed
apatch summary

python -m ai_debug_repair.cli dss capabilities --output json
python -m ai_debug_repair.cli dss profile-template --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss validate-profile --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss connect-test --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss core-list --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss monitor-address --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --address 0xB4C0 --samples 10 --output json
python -m ai_debug_repair.cli dss monitor-symbol --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --out "<app.out>" --symbol "<symbol>" --samples 10 --output json
python -m ai_debug_repair.cli dss report --workspace . --output md

pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\install-aicoding-agent-dev-kit.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\status-aicoding-agent-dev-kit.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-aicoding-agent-dev-kit.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\test-aicoding-agent-dev-kit.ps1 -Json
aicoding-agent-kit clarify init --repo . --requirement "Describe the unclear requirement"
aicoding-agent-kit load --repo . --auto
```

`.ai-debug-repair/` 下的 profile、run script、session log、DSS session evidence 和 Markdown report 属于本机运行状态，默认不提交到 Git。只有明确作为测试 fixture 时，才应单独纳入版本管理。AI Debug Repair Kit 默认不 reset、不 halt、不 run、不 loadProgram、不 flash、不 erase、不 write-memory、不写表达式、不写寄存器。

## Skill 安装边界

AiCoding 把 Skill 分成两类运行入口：

1. **AiCoding Plugin skills**
   - 名称为 `aicoding-*`。
   - 来源于 `Codex-Skills/embedded` 和 `Codex-Skills/platform`。
   - 由 `Codex-Skills/plugins/AiCoding` 打包。
   - 安装后进入 Codex 自己管理的 plugin cache。
   - 不作为 standalone skill 手工链接。

2. **Standalone personal skills**
   - 例如 `obsidian-markdown`、`drawio`、`frontend-design`、`webapp-testing` 等。
   - 源码和备份归 `Codex-Skills` 远程仓库。
   - 不进入 AiCoding Plugin。
   - 由 profile 脚本按 `config/codex-kit.json` 的 `standaloneSkillRegistry` 创建 junction，默认安装到 `%USERPROFILE%\.agents\skills`。

## AiCoding 工作流

AiCoding Plugin 现在内置可独立运行的 SDD、MVP、BDD、架构优先、TDD fallback 和文档同步 workflow。Superpowers 可作为增强能力复用，但不是运行 AiCoding 工作流的硬依赖。

<a id="commands"></a>
## 常用命令 / Commands

查看安装计划：

```powershell
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/set-codex-skill-profile.ps1 -Profile full -DryRun -Json
```

选择兼容安装到 `.codex\skills` 时必须显式指定：

```powershell
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/set-codex-skill-profile.ps1 -Profile full -StandaloneRoot codex -DryRun -Json
```

运行审计：

```powershell
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/audit-runtime-skills.ps1 -Json
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all
```

<a id="git-governance-standard"></a>
## Git 治理标准 / Git Governance Standard

所有 AiCoding 管理的 Git 仓库都必须在 README 或等价治理文档中写明分支、环境、提交类型、Release 说明和双语文档规则。

- 分支：`main` 或 `master` 是稳定生产分支，除批准的 release/hotfix 集成外不得直接改代码；`develop` 是 DEV 集成分支；`feature/<scope>` 从 `develop` 创建；存在共享测试环境时 `test` 对应 FAT；`release/<version>` 对应 UAT/预上线；`hotfix/<scope>` 从 `main` 创建，并回合到 `main` 和 `develop`。
- 环境：`DEV` 用于开发调试，`FAT` 用于功能验收测试，`UAT` 用于用户验收/预生产，`PRO` 用于生产。
- 提交类型：`feat` 新增功能，`fix` 修复 bug，`docs` 仅文档变更，`style` 仅格式/空白等不影响语义的变更，`refactor` 既不修 bug 也不加功能的代码重构，`perf` 性能改进，`test` 添加或修正测试，`build` 构建或打包行为，`ci` 自动化变更，`chore` 辅助工具或维护文件变更。
- 单次提交：一个 commit 只放一类变更，议题不超过 3 个，并使用 `feat(scope): summary` 这类 typed subject。
- 双语规则：README 默认中文优先，顶部必须保留可见的 `README_CN.md` 与 English 快速切换；CHANGELOG、Tag、GitHub Release 和 GitHub About 描述默认中文在前、英文在后。
- Release：Tag 和 GitHub Release 必须按类型汇总本次包含的全部提交，说明本次 release 主类型，并包含 `摘要 / Summary`、`变更内容 / What's Changed`、`兼容性 / Compatibility`、`废弃项 / Deprecations`、`发布说明 / Release Notes`、`完整变更 / Full Changelog`、`新贡献者 / New Contributors`、`已知问题 / Known Issues`、`可追溯性 / Traceability` 和 `资产 / Assets`。

## 维护规则

- 不在 AiCoding submodule 内构建 Plugin。
- 不复制 Skill 源码到 AiCoding。
- 不直接修改 Codex plugin cache。
- 新增或下载 standalone skill 时，先进入 `Codex-Skills` 备份，再加入 `config/codex-kit.json` 的 standalone 清单。
- 新增 `aicoding-*` 成组能力时，先在 `Codex-Skills` 修改 canonical source 和打包清单，再更新 AiCoding submodule。
- 兼容模式下可以保留 `%USERPROFILE%\.codex\skills\.system` 和 standalone skill junction，但 `aicoding-*` 只能来自已安装的 AiCoding Plugin。

## 相关文档

- [English README](README_EN.md)
- [CodingKit 架构](docs/CODEX_KIT_ARCHITECTURE.md)
- [Kit Lifecycle Architecture](docs/KIT_LIFECYCLE_ARCHITECTURE.md)
- [Kit Lifecycle Test Profiles](docs/KIT_LIFECYCLE_TEST_PROFILES.md)
- [Kit Skill Routing](docs/KIT_SKILL_ROUTING.md)
- [Common Code Management](docs/COMMON_CODE_MANAGEMENT.md)
- [Hook System](docs/HOOK_SYSTEM.md)
- [Third-Party Skill Policy](docs/THIRD_PARTY_SKILL_POLICY.md)
- [User-Created Skill Policy](docs/USER_CREATED_SKILL_POLICY.md)
- [Kit Export And Release](docs/KIT_EXPORT_AND_RELEASE.md)
- [Kit Install State](docs/KIT_INSTALL_STATE.md)
- [Agent Dev Kit Plan Mode](docs/AGENT_DEV_KIT_PLAN_MODE.md)
- [Spec Kit Adaptation](docs/SPEC_KIT_ADAPTATION.md)
- [Superpower Skill Adaptation](docs/SUPERPOWER_SKILL_ADAPTATION.md)
- [维护方法](docs/MAINTENANCE_METHOD.md)
- [CodingKit](CodingKit/README.md)
- [更新日志 / CHANGELOG](CHANGELOG.md)
