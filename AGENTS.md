# AiCoding Agent Instructions

## Repository Role

This repository is the platform integration, installation, and CodingKit asset repository.

It does not own Codex skill source code. The authoritative skill and plugin source is the `Codex-Skills` repository mounted at:

```text
CodingKit/agents/skills
```

through a Git submodule.

Before making changes, read:

- `docs/architecture/AICODING_CORE_ARCHITECTURE.md`
- `docs/architecture/CODEX_KIT_ARCHITECTURE.md`
- `docs/operations/MAINTENANCE_METHOD.md`
- `CodingKit/README.md`
- `config/codex-kit.json`
- `CodingKit/AGENTS.md` for CodingKit asset changes

## Dependency Direction And Stable Identity Governance

AiCoding uses the following semantic layers from high to low:

```text
platform -> integration -> capability -> runtime
```

Dependencies may point only to the same layer or a lower layer. A lower layer must not depend on, name, configure, document, or otherwise observe an upper layer.

The executable policy is `config/dependency-governance.json`; validate it with:

```powershell
bin\aicoding.exe governance dependencies --json
```

Required rules:

- `aicoding-*`, `AICODING_*`, `aicoding.local`, and equivalent product namespaces are reserved for platform or integration assets that genuinely depend on AiCoding behavior.
- Reusable Kit, standalone Skill, MCP, module, renderer, schema, environment variable, package, service, example, and test identities must use domain names and remain platform agnostic.
- AiCoding registry or manifest files may bind a platform to a lower capability; the lower capability must not contain the reverse binding.
- Plugin-bundled platform Skills use `aicoding-*`; reusable standalone Skills do not.
- Capability MCP servers expose tools and domain resources. Workflow orchestration, quality procedures, and user intent belong to Skills; capability MCP servers must not own workflow prompt directories or register workflow prompts.
- Stable asset identities must not encode versions in paths, IDs, package/module/service names, C/CMake symbols, model names, or runtime code.
- Asset versions are visible only through manifest metadata, asset documentation, `CHANGELOG.md`, Tag/Release authority, or README badges linked to an exact authority.
- README version badges must be identical across `README.md`, `README_CN.md`, and `README_EN.md`. Third-party versions link to the exact upstream version page; local Kit versions link to the authoritative local Kit document and must match the Kit manifest.
- Implementation directories, filenames, package/module/service names, and stable IDs must not encode versions. README, CHANGELOG, Release, manifest metadata, and explanatory documentation may record versions.
- New product capabilities must compose the stable kernel and capability graph defined by `docs/architecture/AICODING_CORE_ARCHITECTURE.md`; do not add parallel root resolution, registry loading, runner, report, lifecycle, test, or state authorities.

Do not add a time-limited exception for a new reverse dependency or versioned identity. Existing violations must be corrected before integration.

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
5. Run `bin\aicoding.exe test --profile Full --json`.
6. Run install/update dry-runs.
7. Run status JSON validation.
8. Run Markdown, governance, and Git diff checks.
9. Update `CHANGELOG.md`.
10. Commit the AiCoding repository.

Do not update the submodule to a dirty worktree, an unpushed local commit, or an unverified branch tip.

## Required Verification

Before considering AiCoding work complete, run the repository-provided equivalents of:

- `doctor --all`;
- `verify --profile Smoke`;
- `test --profile Smoke|Full|Release` as required by risk;
- `bin\aicoding.exe test --profile Full --json`;
- install dry-run;
- status JSON;
- update dry-run;
- Markdown link validation;
- governance lint;
- `git diff --check`;
- relevant Git hooks.

For an actual release, also test real Marketplace registration, real plugin installation, plugin refresh, Hook review, rollback, and uninstall ownership behavior.

## Validation Evidence Push Rules

- Before pushing `refs/heads/main` or any `refs/tags/*`, require a Release Receipt for the exact
  `local_oid` tree supplied by Git; never substitute the current HEAD or infer profile inheritance.
- Treat a missing/invalid alias, non-fast-forward main update, main deletion, or tag deletion as a
  blocking Context Gate result. Run validation outside the hook on the exact commit, then retry.
- Do not edit Receipt or alias files manually and do not bypass `.githooks/pre-push` to manufacture
  a green result. `--reuse off` remains the explicit full-execution rollback path.
- Repository hooks may only call the prebuilt Go CLI. They must not run tests/builds, modify the
  worktree, stash/reset/checkout, or invoke a recursive push.
- These rules do not authorize Profile inheritance or Plan Mode integration; both remain outside
  ADR 0007 phase 2.

## Canonical Product Control Plane

The formal product workflow is:

```text
bootstrap
-> lifecycle
-> doctor --all / verify --profile
-> test --profile
-> release verify|gate
```

`internal/lifecycle`, `internal/testengine`, `internal/repohealth`, and `internal/report`
are the sole lifecycle, test, product-check, and report authorities. Do not add a second
aggregator in Taskfile, CI, PowerShell, Python, hooks, or documentation.

Expired compatibility commands must not remain routed or appear in current help. Removed
forms belong only in the migration table or historical decisions; canonical commands own
all current behavior.

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

## External GitHub Skill Policy

All Skills downloaded from GitHub must enter the Codex-Skills source repository through a declared nested Git submodule under `external/`; do not copy their source into either repository. Codex-Skills owns `.gitmodules`, the pinned external gitlink, and `config/external-skill-bindings.json`.

AiCoding may expose an external standalone Skill by keeping its runtime name in `profiles.full.standaloneSkills` and `standaloneSkillRegistry.skills`, then mapping that name to the nested directory containing `SKILL.md` through `standaloneSkillRegistry.sourcePaths`. The runtime junction must target that mapped Skill directory, not the external repository root. New-machine and update flows must initialize submodules recursively.

External updates resolve the highest stable semantic-version tag and advance the pinned gitlink only after review. Runtime uninstall removes only a junction whose target exactly matches the registered source path. Repository de-integration must pair removal of the AiCoding registry/source-path entry with removal of the Codex-Skills binding manifest entry, `.gitmodules` section, and gitlink; orphaned runtime or source links are not allowed.


## SDD/BDD/TDD Workflow Policy

AiCoding workflow skills for SDD, MVP, BDD, architecture-first scaffolding, TDD fallback, and documentation synchronization are bundled through the AiCoding plugin only. Superpowers is optional acceleration, not a required dependency.

Documentation synchronization is enforced by `bin/aicoding.exe docsync`, `.githooks/pre-commit`, and `.github/workflows/aicoding-ci.yml`. Code, script, config, hook, CI, or CodingKit changes must include a documentation update or an explicit no-doc-change review note with a meaningful reason; see `docs/architecture/DOC_SYNC_PLUS_SPEC.md` for the marker format.

`cmd/**/*.go`, `internal/cli/**/*.go`, `internal/testengine/**/*.go`, `Taskfile.yml`, and
`.github/workflows/**` are public command-contract surfaces. Changes to them must review
`README.md`, `README_CN.md`, `README_EN.md`, `docs/COMMANDS.md`,
`docs/ARCHITECTURE_OVERVIEW.md`, relevant test documentation, and `CHANGELOG.md`.

## README 和工具链可见性策略 / README And Toolchain Visibility Policy

- `README.md`、`README_CN.md`、`README_EN.md` 是架构入口，只描述平台、kit、plugin、skill 母级边界，不列具体 leaf skill 命令。
- 具体 skill 命令、专项格式化命令和 Taskfile 子命令放在 `docs/COMMANDS.md` 或对应专项文档中。
- 新增稳定工具链、运行体系或默认验证体系时，必须同步三份 README 顶部 badge，并链接到权威 URL。
- README 中的版本只能通过顶部 badge 展示；第三方版本链接上游准确版本页，本地 Kit 版本链接本仓库权威 Kit 文档。
- README 架构图优先使用短文本图；如果使用 Mermaid，节点标签必须短，不写长路径、不写具体 leaf skill 名，避免渲染裁剪。

## 语言策略 / Language Policy

- 本仓库默认中文优先。
- 面向用户的执行计划、解释、权限请求摘要、验证结果、风险说明、rollback/handoff 说明必须使用中文。
- 英文术语可以保留，但应作为括号说明，例如：计划模式（Plan Mode）、规格驱动开发（SDD）、注册表（registry）。
- JSON 字段名、命令、路径、参数、文件名不翻译。
- 如果 Codex 需要请求用户授权执行命令，必须用中文说明“为什么要执行这个命令”。
- 不要生成英文权限摘要；应写成“读取 Plan Mode registry，用于验证前检查”。
