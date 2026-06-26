# Codex Kit Architecture

AiCoding is the platform entrypoint. Codex-Skills is the single source of truth for skills, plugin assembly, and plugin-bundled Codex hooks.

## Layer Model

```text
Codex-Skills canonical sources
-> Codex-Skills/plugins/AiCoding package
-> AiCoding/CodingKit/agents/skills submodule gitlink
-> AiCoding Marketplace install or refresh
-> local Codex plugin cache
-> CodingKit external assets discovered at runtime
```

Each layer has a different synchronization mechanism:

- source to plugin package: `Codex-Skills/scripts/build-plugin.ps1`;
- Codex-Skills to AiCoding: Git submodule gitlink;
- AiCoding to local Codex: Marketplace install or refresh;
- plugin to CodingKit assets: `AICODING_HOME`, install state, `PATH`, project discovery, or MCP.

## Ownership

Codex-Skills owns canonical skills, plugin manifest, plugin hooks, plugin assets, build scripts, verification scripts, and package metadata.

AiCoding owns platform integration, Marketplace, install/update/status/uninstall scripts, CodingKit assets, Git hooks, docs, and the submodule pointer.

AiCoding must not edit or rebuild the submodule. It only selects a verified Codex-Skills commit or tag.

## Generated Output

`CodingKit/agents/skills/plugins/AiCoding/skills` and `BUILDINFO.json` are generated in Codex-Skills and consumed through the submodule.

AiCoding must not regenerate these files.

## Hook Boundary

Codex hooks live in the plugin and are reviewed through `/hooks`. They are auxiliary lifecycle helpers, not a complete safety boundary.

Git hooks live in `.githooks/` and enforce repository-local commit, changelog, and governance rules.

## External Assets

These directories are platform assets and are not copied into the plugin:

- `CodingKit/examples`
- `CodingKit/modules`
- `CodingKit/platforms`
- `CodingKit/tests`
- `CodingKit/tools`

Plugin skills and tools must discover these assets through the approved discovery protocol rather than hard-coded relative paths out of the installed plugin.
## Runtime Skill Layers

Normal runtime separates three sources:

- AiCoding plugin cache exposes bundled `aicoding-*` skills.
- `%USERPROFILE%\.agents\skills` exposes selected standalone personal skills such as Obsidian skills.
- Codex-Skills source checkouts stay outside user-level Skill Roots.

Development mode may expose one selected canonical Skill source, but the same-name plugin skill must be disabled while that development link is active.
## Skill Role Boundaries

`aicoding-git-governance` and `aicoding-kit-maintenance` are both platform skills, but they govern different layers.

`aicoding-git-governance` controls repository operations:

- branch, commit, README, CHANGELOG, tag, release, and GitHub workflow decisions;
- typed commit and changelog rules;
- repository-local Git hooks and release gates;
- firmware artifact and delivery governance when a repository publishes embedded outputs.

`aicoding-kit-maintenance` controls the AiCoding kit lifecycle:

- Codex-Skills and AiCoding repository boundaries;
- plugin packaging, generated output, hooks, and BUILDINFO behavior;
- submodule update order and Marketplace installation flow;
- runtime Skill Root migration, profile switching, and duplicate Skill exposure policy.

Use `aicoding-kit-maintenance` to decide what should change in the kit, then use `aicoding-git-governance` for the actual Git commit, changelog, push, tag, or release workflow.

`aicoding-user-skill-creator` is the user-maintained Skill authoring workflow. It is bundled in the AiCoding plugin under an `aicoding-` name so the system `skill-creator` can remain installed without name conflicts.

