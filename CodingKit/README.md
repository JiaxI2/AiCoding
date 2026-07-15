# CodingKit

CodingKit is the platform layer for local AI-assisted embedded development.

## Layout

```text
CodingKit/
├── agents/
│   └── skills/        Git submodule: JiaxI2/Codex-Skills
├── examples/          Example projects and bring-up cases
├── modules/           Reusable embedded modules
├── platforms/         Board, MCU, RTOS, and toolchain templates
├── tests/             Verification assets and regression cases
└── tools/             Local tools and diagnostics
```

`tools/c-userstyle-kit` 是首方 C99 生成与验证资产。它由 kit registry 管理，并通过现有
`skill c99-standard-c` 用户入口调用；不会创建第二套顶层 formatting 命令。

## Codex Kit

The installable Codex plugin is provided by the submodule at:

```text
CodingKit/agents/skills/plugins/AiCoding
```

AiCoding does not rebuild this plugin inside the submodule. Build and verification happen in `Codex-Skills`; AiCoding only locks a verified commit and installs it through its Marketplace.

The bundled AiCoding plugin includes standalone-capable SDD, MVP, BDD, architecture-first, TDD fallback, and documentation synchronization workflow skills. Superpowers remains optional.

## Asset Discovery

Plugin skills and hooks discover CodingKit assets by this protocol:

1. use `AICODING_HOME` when it is set;
2. otherwise walk upward from the active repository until `config/codex-kit.json` is found;
3. resolve `examples`, `modules`, `platforms`, `tests`, and `tools` from that manifest;
4. treat missing optional assets as unavailable capability, not as plugin failure.

## C UserStyle Kit

C UserStyle Kit 1.2.0 位于 `CodingKit/tools/c-userstyle-kit`，包含黄金 Demo、高级规则覆盖样例、
139 条规则目录、VS Code 兼容 snippets、lint、主机编译与行为测试。华为 C 语言编程规范
DKBA 2826-2011.5 的 PDF 和 Markdown 参考副本随该首方资产发布。

用户保持使用统一 Go CLI 入口执行秒级快速验证：

```powershell
bin/aicoding.exe skill c99-standard-c verify --json
```

该验证仅使用主机工具链和临时测试程序，不接入或修改固件工程构建。

## New Machine Setup

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
powershell -NoProfile -ExecutionPolicy Bypass -File tools/specialty/verify-codex-kit.ps1
bin/aicoding.exe lifecycle install --all --json
```

After installing the plugin, open Codex `/hooks` and review/trust the plugin-bundled hooks.

The install script creates the local Marketplace link required by the Codex plugin CLI:

```text
plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding
```

This link is local generated state. It must not be used to copy plugin files into AiCoding.
## Runtime Skill Exposure

`CodingKit/agents/skills` is a submodule and must not be linked wholesale into a user Skill Root.

Normal runtime should expose `aicoding-*` skills through the installed AiCoding plugin. Personal standalone skills are linked selectively from `%USERPROFILE%\.agents\skills` by default. The complete registry lives in `config/codex-kit.json` under `standaloneSkillRegistry`, and compatibility installs can target `%USERPROFILE%\.codex\skills` only when `set-codex-skill-profile.ps1 -StandaloneRoot codex` is explicitly selected.

GitHub-sourced standalone Skills are not copied into AiCoding. They are pinned as nested submodules under `Codex-Skills/external/`, and `standaloneSkillRegistry.sourcePaths` maps each runtime name to the nested directory that contains its `SKILL.md`. Clone and update flows therefore use recursive submodule initialization.

When compatibility mode keeps `%USERPROFILE%\.codex\skills`, keep `.system` and selected standalone links only. Remove source checkout directories such as `embedded`, `platform`, and `plugins/AiCoding/skills` from active runtime exposure after backing them up.

Run the runtime audit before and after install, update, migration, profile switching, or uninstall work:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File tools/specialty/audit-runtime-skills.ps1 -Json
```
