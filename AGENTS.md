# AiCoding Agent Instructions

## Repository Role

This repository is the platform integration, installation, and CodingKit asset repository.

It does not own Codex skill source code. The authoritative skill and plugin source is the `Codex-Skills` repository mounted at:

```text
CodingKit/agents/skills
```

through a Git submodule.

Before making changes, read:

- `docs/CODEX_KIT_ARCHITECTURE.md`
- `docs/MAINTENANCE_METHOD.md`
- `CodingKit/README.md`
- `config/codex-kit.json`
- `CodingKit/AGENTS.md` for CodingKit asset changes

## Submodule Policy

Treat `CodingKit/agents/skills` as a read-only released dependency.

Allowed operations:

- initialize;
- fetch;
- checkout an approved commit or tag;
- inspect;
- validate;
- update the parent repository gitlink.

Prohibited operations:

- editing skill source;
- rebuilding the plugin;
- regenerating plugin skills;
- modifying `BUILDINFO.json`;
- formatting files inside the submodule;
- automatically committing inside the submodule.

After running AiCoding install, update, status, verify, or uninstall scripts, the submodule must remain clean.

## Repository Ownership

AiCoding owns:

- `.agents/plugins/marketplace.json`;
- `config/codex-kit.json`;
- installation, update, status, verification, and uninstall scripts;
- `CodingKit/examples/`;
- `CodingKit/modules/`;
- `CodingKit/platforms/`;
- `CodingKit/tests/`;
- `CodingKit/tools/`;
- project-level `.githooks/`;
- platform integration documentation.

AiCoding does not own:

- embedded Skill source;
- platform Skill source;
- Plugin-generated Skill packages;
- Plugin-bundled Hook source.

## Plugin Installation

The Marketplace source must point to:

```text
./CodingKit/agents/skills/plugins/AiCoding
```

Do not copy the plugin to another source directory.

Do not directly modify the Codex plugin cache.

A submodule update does not automatically update the installed local plugin. Use the repository update workflow to verify the new submodule package, detect package drift, preserve the enabled state, refresh the plugin through Marketplace, update installation state, and report whether Hook review is required.

## External CodingKit Assets

The following directories are external platform assets and must not be copied into the plugin:

- `CodingKit/examples/`
- `CodingKit/modules/`
- `CodingKit/platforms/`
- `CodingKit/tests/`
- `CodingKit/tools/`

Expose them to installed plugin workflows through:

- `AICODING_HOME`;
- installation state;
- `PATH`;
- approved project discovery;
- MCP.

## Cross-Repository Upgrade Workflow

To upgrade the Codex kit:

1. Confirm the target Codex-Skills commit or tag exists remotely.
2. Confirm that commit contains a fully generated and validated plugin.
3. Update `CodingKit/agents/skills` to the approved commit.
4. Stage only the submodule gitlink and intended AiCoding changes.
5. Run `verify-codex-kit`.
6. Run install/update dry-runs.
7. Run status JSON validation.
8. Run Markdown, governance, and Git diff checks.
9. Update `CHANGELOG.md`.
10. Commit the AiCoding repository.

Do not update the submodule to a dirty worktree, an unpushed local commit, or an unverified branch tip.

## Required Verification

Before considering AiCoding work complete, run the repository-provided equivalents of:

- `verify-codex-kit`;
- install dry-run;
- status JSON;
- update dry-run;
- Markdown link validation;
- governance lint;
- `git diff --check`;
- relevant Git hooks.

For an actual release, also test real Marketplace registration, real plugin installation, plugin refresh, Hook review, rollback, and uninstall ownership behavior.

## Prohibited Actions

Do not:

- build inside the submodule;
- copy Skill source into AiCoding;
- copy CodingKit asset directories into the plugin;
- treat a submodule update as an installed-plugin update;
- directly edit plugin cache files;
- delete user-managed standalone skills;
- delete unknown Codex configuration;
- silently overwrite an existing `AICODING_HOME`;
- use destructive Git cleanup without explicit authorization.

## Definition Of Done

Work is complete only when the submodule points to an existing verified Codex-Skills commit, the submodule remains clean, Marketplace paths resolve, scripts pass validation and dry-runs, documentation and changelog are updated, and no Skill source has been duplicated into AiCoding.
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


## SDD/BDD/TDD Workflow Policy

AiCoding workflow skills for SDD, MVP, BDD, architecture-first scaffolding, TDD fallback, and documentation synchronization are bundled through the AiCoding plugin only. Superpowers is optional acceleration, not a required dependency.

Documentation synchronization is enforced by `bin/aicoding.exe docsync`, `.githooks/pre-commit`, and `.github/workflows/docs-sync.yml`. Code, script, config, hook, CI, or CodingKit changes must include a documentation update or an explicit no-doc-change review note with a meaningful reason; see `docs/DOC_SYNC_PLUS_SPEC.md` for the marker format.

## 语言策略 / Language Policy

- 本仓库默认中文优先。
- 面向用户的执行计划、解释、权限请求摘要、验证结果、风险说明、rollback/handoff 说明必须使用中文。
- 英文术语可以保留，但应作为括号说明，例如：计划模式（Plan Mode）、规格驱动开发（SDD）、注册表（registry）。
- JSON 字段名、命令、路径、参数、文件名不翻译。
- 如果 Codex 需要请求用户授权执行命令，必须用中文说明“为什么要执行这个命令”。
- 不要生成英文权限摘要；应写成“读取 Plan Mode registry，用于验证前检查”。
