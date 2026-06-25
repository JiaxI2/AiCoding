# AiCoding

AiCoding is a platform repository for local AI-assisted embedded development. It integrates CodingKit assets, repository governance, and a version-locked Codex plugin kit.

## Quick Start

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1
```

## Repository Roles

- `CodingKit/agents/skills` is a submodule pointing to `https://github.com/JiaxI2/Codex-Skills.git`.
- `CodingKit/agents/skills/plugins/AiCoding` is the packaged Codex plugin source for installation.
- `.agents/plugins/marketplace.json` is the AiCoding platform Marketplace entry.
- `config/codex-kit.json` defines CodingKit asset discovery and installation rules.
- `.githooks/` contains Git-native hooks for this repository; Codex hooks live inside the plugin.

## CodingKit Assets

```text
CodingKit/examples
CodingKit/modules
CodingKit/platforms
CodingKit/tests
CodingKit/tools
```

These directories are platform assets. They are not copied into the Codex plugin. Skills and tools discover them through `config/codex-kit.json` or `AICODING_HOME`.

## Maintenance Commands

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/status-codex-kit.ps1 -Json
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/update-codex-kit.ps1 -DryRun
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
```

Do not rebuild `plugins/AiCoding` inside the submodule from AiCoding. Update the submodule only after Codex-Skills has built, verified, committed, and pushed the plugin package.

## Documentation

- [CodingKit](CodingKit/README.md)
- [CHANGELOG](CHANGELOG.md)
- [Repository Governance](.github/repository-governance.toml)