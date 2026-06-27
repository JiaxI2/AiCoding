# AiCoding

Chinese documentation: [README_CN.md](README_CN.md).

AiCoding is a platform repository for local AI-assisted embedded development. It integrates CodingKit assets, repository governance, and a version-locked Codex plugin kit.

## Quick Start

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1
```

`install-codex-kit.ps1` creates the local Marketplace link `plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding`, registers this repository as the `aicoding-platform` Marketplace through the Codex plugin CLI when available, and installs `aicoding@aicoding-platform`. The link is local generated state and is intentionally ignored by Git.

## Repository Roles

- `CodingKit/agents/skills` is a submodule pointing to `https://github.com/JiaxI2/Codex-Skills.git`.
- `CodingKit/agents/skills/plugins/AiCoding` is the packaged Codex plugin source for installation.
- `aicoding-user-skill-creator` is bundled in the AiCoding plugin as User-Skill-Creator; the system `skill-creator` remains separate.
- `.agents/plugins/marketplace.json` is the AiCoding platform Marketplace entry.
- `config/codex-kit.json` defines CodingKit asset discovery and installation rules.
- `.githooks/` contains Git-native hooks for this repository; Codex hooks live inside the plugin.
- The AiCoding plugin bundles standalone-capable SDD, MVP, BDD, architecture-first, TDD fallback, and documentation synchronization workflow skills; Superpowers can be reused when installed but is not required.

## CodingKit Assets

```text
CodingKit/examples
CodingKit/modules
CodingKit/platforms
CodingKit/tests
CodingKit/tools
```

These directories are platform assets. They are not copied into the Codex plugin. Skills and tools discover them through `config/codex-kit.json` or `AICODING_HOME`.


## Standalone Skills

AiCoding separates bundled plugin skills from personal standalone skills.

- Bundled `aicoding-*` skills are installed through the AiCoding Codex Plugin and managed by Codex plugin cache.
- Personal or downloaded standalone skills are backed up in `Codex-Skills` and installed by profile as junctions into `%USERPROFILE%\.agents\skills` by default.
- `scripts/set-codex-skill-profile.ps1 -Profile full -DryRun -Json` shows the complete standalone skill install plan.
- Use `-StandaloneRoot codex` only when a compatibility workflow explicitly needs `%USERPROFILE%\.codex\skills`; the default is `-StandaloneRoot agents`.
- A clean compatibility runtime may keep `%USERPROFILE%\.codex\skills\.system` and selected standalone skill junctions, but `aicoding-*` must come only from the installed AiCoding plugin.

## Maintenance Commands

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/status-codex-kit.ps1 -Json
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/update-codex-kit.ps1 -DryRun
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all
```

Do not rebuild `plugins/AiCoding` inside the submodule from AiCoding. Update the submodule only after Codex-Skills has built, verified, committed, and pushed the plugin package.

## Documentation

- [Codex Kit Architecture](docs/CODEX_KIT_ARCHITECTURE.md)
- [Maintenance Method](docs/MAINTENANCE_METHOD.md)
- [CodingKit](CodingKit/README.md)
- [CHANGELOG](CHANGELOG.md)
- [Repository Governance](.github/repository-governance.toml)
