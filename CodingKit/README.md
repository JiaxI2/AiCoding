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

## Codex Kit

The installable Codex plugin is provided by the submodule at:

```text
CodingKit/agents/skills/plugins/AiCoding
```

AiCoding does not rebuild this plugin inside the submodule. Build and verification happen in `Codex-Skills`; AiCoding only locks a verified commit and installs it through its Marketplace.

## Asset Discovery

Plugin skills and hooks discover CodingKit assets by this protocol:

1. use `AICODING_HOME` when it is set;
2. otherwise walk upward from the active repository until `config/codex-kit.json` is found;
3. resolve `examples`, `modules`, `platforms`, `tests`, and `tools` from that manifest;
4. treat missing optional assets as unavailable capability, not as plugin failure.

## New Machine Setup

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1
```

After installing the plugin, open Codex `/hooks` and review/trust the plugin-bundled hooks.
## Runtime Skill Exposure

`CodingKit/agents/skills` is a submodule and must not be linked wholesale into a user Skill Root.

Normal runtime should expose `aicoding-*` skills through the installed AiCoding plugin. Personal standalone skills are linked selectively from `%USERPROFILE%\.agents\skills`.

Run the runtime audit before and after install, update, migration, profile switching, or uninstall work:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/audit-runtime-skills.ps1 -Json
```
