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
```

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