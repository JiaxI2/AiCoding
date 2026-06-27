# AiCoding

AiCoding 是本地 AI 辅助嵌入式开发平台仓库。它不直接维护 Skill 源码，而是通过 `CodingKit/agents/skills` submodule 锁定 `Codex-Skills` 的已验证版本，并提供安装、更新、状态、卸载、运行时审计和 CodingKit 资产入口。

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
