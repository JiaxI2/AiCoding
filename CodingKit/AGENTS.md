# CodingKit Asset Instructions

## Directory Role

CodingKit contains platform assets used by AiCoding workflows.

`agents/skills` is a read-only Git submodule. The remaining directories are external assets, not plugin package content.

## Asset Classification

- `examples/`: example projects and demonstrations
- `modules/`: reusable engineering modules
- `platforms/`: MCU, DSP, board, SDK, or platform-specific support
- `tests/`: CodingKit integration and compatibility tests
- `tools/`: deterministic CLI tools and utility programs
- `agents/skills/`: released Codex-Skills dependency

## Packaging Rules

Do not copy `examples`, `modules`, `platforms`, `tests`, or `tools` wholesale into the AiCoding plugin.

A small file may be moved into plugin assets only when it is:

- required for plugin execution;
- small;
- dependency-light;
- cross-project;
- versioned together with the plugin.

Large tools, SDK support, example projects, platform files, and shared modules remain in CodingKit.

## Discovery Contract

Plugin workflows must locate CodingKit through:

- `AICODING_HOME`;
- the installation-state file;
- `PATH`;
- approved project discovery;
- MCP.

Do not make plugin workflows depend on the source checkout being adjacent to the plugin cache.

## Submodule Rule

Do not modify, build, format, or generate files under `agents/skills` from AiCoding scripts.

Any required Skill or Plugin change must be made and released in Codex-Skills first.
## Runtime Skill Exposure Policy

`Codex-Skills` is a source and build repository, not a user-level skill discovery root.

Do not place or link the whole repository under:

- `%USERPROFILE%\.agents\skills`
- `%USERPROFILE%\.codex\skills`

Normal runtime mode must expose bundled `aicoding-*` skills only through the installed AiCoding plugin.

Canonical `embedded/` and `platform/` skill sources may be exposed only in an explicit `skill-development` profile. The AiCoding plugin must be disabled while a same-name canonical skill is linked.

Generated plugin skills under `plugins/AiCoding/skills/` must never be linked as standalone user skills.

Before completing installation, upgrade, migration, profile switching, or uninstall work, run the runtime Skill audit and reject duplicate active Skill names.

Source skills may have generated package copies in the repository, but a Skill name must have only one active runtime source.

