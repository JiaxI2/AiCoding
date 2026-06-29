# AiCoding

AiCoding 是本地 AI 辅助嵌入式开发平台仓库。它不直接维护 Skill 源码，而是通过 `CodingKit/agents/skills` submodule 锁定 `Codex-Skills` 的已验证版本，并提供安装、更新、状态、卸载、运行时审计、CodingKit 资产入口、Agent Patch Kit 和 AI Debug Repair Kit。

## 快速开始

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1 -DryRun
```

真实安装 Plugin 时优先使用 Codex 的 Marketplace/plugin 机制。不要手工修改 Codex plugin cache。

执行真实安装时，`install-codex-kit.ps1` 会创建本地 Marketplace 需要的 `plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding` junction，然后通过 Codex plugin CLI 注册 `aicoding-platform` 并安装 `aicoding@aicoding-platform`。`plugins/` 是本机生成状态，不提交到 Git。

## 本地 Agent Kit

AiCoding 还通过本地 Marketplace 发布两套仓库级 Agent Kit：

- Agent Patch Kit：Marketplace 名称为 `aicoding-agent-patch-kit`，来源为 `dist/agent-patch-kit/plugins/AiCodingAgentPatch`，提供 `apatch` 安全补丁流程、状态门禁、固定字符串扫描/替换、事务快照、Markdown 链接检查和 patch summary。
- AI Debug Repair Kit：Marketplace 名称为 `aicoding-ai-debug-repair-kit`，来源为 `dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit`，提供 `airepair`，用于有边界的 build/test repair loop 和只读嵌入式 debug 辅助。v0.4.0 包含 `ti_dss` TI XDS/CCS DSS scaffold，以及受 policy 限制的 J-Link 侵入式操作 stub。

环境要求：

- 默认使用 PowerShell 7（`pwsh`）执行仓库安装、验证、状态、更新和文档检查；Windows PowerShell 5.1 只用于明确的兼容性门禁。同时需要 Git、Python 3.10+ 和 Codex plugin Marketplace 流程。
- Agent Patch Kit 使用用户态 `apatch` CLI。验证命令：`apatch install doctor`、`apatch brief --format md`、`apatch state status`。
- AI Debug Repair Kit 使用用户态 `ai-debug-repair-kit` Python 包。验证命令：`python -m ai_debug_repair.cli version --output json`、`python -m ai_debug_repair.cli doctor --output json`、`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-ai-debug-repair-kit.ps1 -Json`。
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

## 常用命令

查看安装计划：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/set-codex-skill-profile.ps1 -Profile full -DryRun -Json
```

选择兼容安装到 `.codex\skills` 时必须显式指定：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/set-codex-skill-profile.ps1 -Profile full -StandaloneRoot codex -DryRun -Json
```

运行审计：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/audit-runtime-skills.ps1 -Json
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all
```

## Git 治理标准

所有 AiCoding 管理的 Git 仓库都必须在 README 或等价治理文档中写明分支、环境、提交类型和 Release 说明规则。

- 分支：`main` 或 `master` 是稳定生产分支，除批准的 release/hotfix 集成外不得直接改代码；`develop` 是 DEV 集成分支；`feature/<scope>` 从 `develop` 创建；存在共享测试环境时 `test` 对应 FAT；`release/<version>` 对应 UAT/预上线；`hotfix/<scope>` 从 `main` 创建，并回合到 `main` 和 `develop`。
- 环境：`DEV` 用于开发调试，`FAT` 用于功能验收测试，`UAT` 用于用户验收/预生产，`PRO` 用于生产。
- 提交类型：`feat` 新增功能，`fix` 修复 bug，`docs` 仅文档变更，`style` 仅格式/空白等不影响语义的变更，`refactor` 既不修 bug 也不加功能的代码重构，`perf` 性能改进，`test` 添加或修正测试，`build` 构建或打包行为，`ci` 自动化变更，`chore` 辅助工具或维护文件变更。
- 单次提交：一个 commit 只放一类变更，议题不超过 3 个，并使用 `feat(scope): summary` 这类 typed subject。
- Release：Tag 和 GitHub Release 必须按类型汇总本次包含的全部提交，说明本次 release 主类型，并写清具体影响。

## 维护规则

- 不在 AiCoding submodule 内构建 Plugin。
- 不复制 Skill 源码到 AiCoding。
- 不直接修改 Codex plugin cache。
- 新增或下载 standalone skill 时，先进入 `Codex-Skills` 备份，再加入 `config/codex-kit.json` 的 standalone 清单。
- 新增 `aicoding-*` 成组能力时，先在 `Codex-Skills` 修改 canonical source 和打包清单，再更新 AiCoding submodule。
- 兼容模式下可以保留 `%USERPROFILE%\.codex\skills\.system` 和 standalone skill junction，但 `aicoding-*` 只能来自已安装的 AiCoding Plugin。

## 相关文档

- [English README](README.md)
- [CodingKit 架构](docs/CODEX_KIT_ARCHITECTURE.md)
- [维护方法](docs/MAINTENANCE_METHOD.md)
- [CodingKit](CodingKit/README.md)
- [更新日志](CHANGELOG.md)
