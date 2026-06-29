# AiCoding

AiCoding is a platform repository for local AI-assisted embedded development. It integrates CodingKit assets, repository governance, a version-locked Codex plugin kit, Agent Patch Kit, and AI Debug Repair Kit for safer agent editing and non-invasive embedded debug workflows.

## Quick Start

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1
```

`install-codex-kit.ps1` creates the local Marketplace link `plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding`, registers this repository as the `aicoding-platform` Marketplace through the Codex plugin CLI when available, and installs `aicoding@aicoding-platform`. The link is local generated state and is intentionally ignored by Git.

## Local Agent Kits

AiCoding also publishes repo-scoped agent kits through the local Marketplace:

- Agent Patch Kit: `aicoding-agent-patch-kit`, installed from `dist/agent-patch-kit/plugins/AiCodingAgentPatch`, provides the `apatch` safe patch workflow, state gates, fixed-string scan/replace, transaction snapshots, Markdown link checks, and patch summaries.
- AI Debug Repair Kit: `aicoding-ai-debug-repair-kit`, installed from `dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit`, provides `airepair` for bounded build/test repair loops and read-only embedded debug helpers. v0.4.0 includes the `ti_dss` TI XDS/CCS DSS scaffold and policy-gated J-Link invasive-operation stubs.

Environment expectations:

- PowerShell 7 (`pwsh`) is the default shell for repository install, verify, status, update, and documentation checks; Windows PowerShell 5.1 is used only for explicit compatibility gates. Git, Python 3.10+, and the Codex plugin Marketplace flow are also required.
- Agent Patch Kit uses the user-mode `apatch` CLI. Validate it with `apatch install doctor`, `apatch brief --format md`, and `apatch state status`.
- AI Debug Repair Kit installs the user-mode `ai-debug-repair-kit` Python package. Validate it with `python -m ai_debug_repair.cli version --output json`, `python -m ai_debug_repair.cli doctor --output json`, and `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-ai-debug-repair-kit.ps1 -Json`.
- TI DSP debug flows require TI CCS/DSS, such as `C:\ti\ccs1281\ccs\ccs_base\scripting\bin\dss.bat`, plus an XDS probe and a target `.ccxml` before real hardware execution. The default profile remains non-invasive: no reset, halt, run, flash, or writes.

Typical usage:

```powershell
apatch status
apatch scan "README.md" --fixed
apatch summary

python -m ai_debug_repair.cli dss capabilities --output json
python -m ai_debug_repair.cli dss profile-template --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss validate-profile --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss doctor --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
```

Machine-local AI Debug Repair state under `.ai-debug-repair/` is ignored by Git. Do not commit generated profiles, run scripts, or session logs unless a specific test fixture is intentionally added.

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

## Git Governance Standard

All AiCoding-governed Git repositories must document branch, environment, commit type, and release-note rules in README or an equivalent governance file.

- Branches: `main` or `master` is the stable production branch and must not receive direct code edits except approved release or hotfix integration; `develop` is the DEV integration branch; `feature/<scope>` branches start from `develop`; `test` maps to FAT when a shared test environment exists; `release/<version>` maps to UAT/pre-production; `hotfix/<scope>` starts from `main` and is merged back to `main` and `develop`.
- Environments: `DEV` is developer debugging, `FAT` is functional acceptance testing, `UAT` is user acceptance/pre-production, and `PRO` is production.
- Commit types: `feat` adds functionality, `fix` repairs bugs, `docs` changes documentation only, `style` changes formatting without behavior impact, `refactor` restructures code without feature or bug-fix intent, `perf` improves performance, `test` adds or corrects tests, `build` changes build or packaging behavior, `ci` changes automation, and `chore` changes supporting tools or maintenance files.
- Single commits: one commit should contain one category of change, no more than three tightly related topics, and a typed subject such as `feat(scope): summary`.
- Releases: Tag and GitHub Release notes must group every included commit by type, state the primary release type, and describe the concrete user-facing or maintenance impact.

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
