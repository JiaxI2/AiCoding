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

C UserStyle Kit 位于 `CodingKit/tools/c-userstyle-kit`，包含黄金 Demo、高级规则覆盖样例、
139 条规则目录、VS Code 兼容 snippets、lint、主机编译与行为测试。华为 C 语言编程规范
DKBA 2826-2011.5 的 PDF 和 Markdown 参考副本随该首方资产发布。

用户保持使用统一 Go CLI 入口执行秒级快速验证：

```powershell
bin/aicoding.exe skill c99-standard-c verify --json
```

该验证仅使用主机工具链和临时测试程序，不接入或修改固件工程构建。

## Windows Automation MCP

`tools/windows-automation/visio-mcp` 与 `tools/windows-automation/ppt-mcp` 分别是通用 Visio、PowerPoint capability。AiCoding 只在上层 MCP registry 和 Go 控制面中登记、安装、诊断和验证它们；组件的 package、module、环境变量、schema、示例和测试不观察 AiCoding。

组件提供 Diagram IR、Visio tools、快照、检查、质量检测、有限 repair 和 VSDX/PNG/SVG/PDF 导出。简单同质图可使用全局统一尺寸；复杂图按 `sizeClass`、文字安全区、端口密度和容器职责确定有界尺寸，并检查中心轴、绝对端口、紧凑度、箭头净空和文字碰撞。

`ppt-mcp` 以普通源码 vendor 到本仓库，作为 AiCoding 自行维护的 canonical source；它不保留上游 remote、submodule 或自动更新关系。组件提供 PowerPoint COM tools，Smoke/Full 只运行 mock 回归，Release 才显式启动可见 PowerPoint COM 冒烟。

MCP 不注册工作流 prompt。完整画图流程属于上层通用 `visio-diagram` Skill；控制面与运维入口见 [MCP Control Plane](../docs/architecture/MCP_CONTROL_PLANE.md) 和 [MCP Components](../docs/operations/MCP_COMPONENTS.md)。

## New Machine Setup

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
powershell -NoProfile -ExecutionPolicy Bypass -File tools/specialty/verify-codex-kit.ps1
bin/aicoding.exe lifecycle install --all --json
bin/aicoding.exe doctor --all --json
bin/aicoding.exe verify --profile Smoke --json
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
