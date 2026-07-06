# AiCoding

<p align="center">
  <a href="README_CN.md">中文 README_CN.md</a> |
  <a href="README_EN.md">English README_EN.md</a> |
  <a href="CHANGELOG.md">CHANGELOG / 更新日志</a> |
  <a href="#environment-preview">Environment / 环境预览</a>
</p>

[![Version](https://img.shields.io/badge/Version-0.1.0-2ea44f)](config/codex-kit.json)
[![Verify](https://img.shields.io/badge/verify--codex--kit-required-2ea44f)](#maintenance-commands)
[![PowerShell](https://img.shields.io/badge/PowerShell-7-5391FE)](https://learn.microsoft.com/powershell/scripting/install/install-powershell)
[![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB)](https://www.python.org/downloads/)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8)](https://go.dev/doc/install)
[![License](https://img.shields.io/badge/License-Apache--2.0-blue)](LICENSE)

AiCoding is a platform repository for local AI-assisted embedded development. It integrates CodingKit assets, repository governance, a version-locked Codex plugin kit, Agent Patch Kit, AI Debug Repair Kit, and AiCoding Agent Dev Kit for safer agent editing, clearer Git synchronization rules, and default non-invasive embedded debug assistance.

<a id="environment-preview"></a>
## Environment Preview / 环境预览

| Item / 项目 | Current rule / 当前规则 | Link / 快速跳转 |
|---|---|---|
| Shell / 运行 Shell | PowerShell 7 by default; Windows PowerShell 5.1 only for compatibility gates / 默认 PowerShell 7；Windows PowerShell 5.1 仅做兼容性门禁 | [Maintenance Commands](#maintenance-commands) |
| Go fast path / Go 快路径 | Go 1.22+ builds `bin/aicoding.exe` for hooks, Smoke checks, governance lint, and perf probes / Go 1.22+ 构建 `bin/aicoding.exe`，接管 Git hook、Smoke 检查、治理 lint 和性能探针 | [Fast Path V1](#fast-path-v1) |
| Plugin install / Plugin 安装 | Install `aicoding@aicoding-platform` through the local Marketplace / 通过本地 Marketplace 安装 | [Quick Start](#quick-start) |
| Agent Patch Kit | Safe `apatch` patching, fixed-string scans, transaction snapshots, and Markdown checks / 安全补丁、扫描、事务快照和 Markdown 链接检查 | [Local Agent Kits](#local-agent-kits) |
| AI Debug Repair Kit | `airepair` build/test repair and TI DSS read-only scaffold / build-test repair 与 TI DSS 只读 scaffold | [Local Agent Kits](#local-agent-kits) |
| AiCoding Agent Dev Kit | `aicoding-agent-kit` clarification, option matrix, Plan Mode, Spec/TDD, sequential loading, and progress tracking / 需求澄清、方案矩阵、Plan Mode、Spec/TDD、顺序加载和进度监控 | [Local Agent Kits](#local-agent-kits) |
| Kit Lifecycle v2.0 | Unified `scripts/aicoding-kit.ps1` Kit lifecycle and skill routing entrypoint / 统一 registry/manifest adapter 入口 | [Maintenance Commands](#maintenance-commands) |
| Git governance / Git 治理 | README, CHANGELOG, Tag, Release, and About are Chinese first, English second / 默认中文在前、英文在后 | [Git Governance Standard](#git-governance-standard) |

<a id="quick-start"></a>
## Quick Start / 快速开始

```powershell
git clone --recurse-submodules https://github.com/JiaxI2/AiCoding.git
cd AiCoding
git submodule update --init --recursive
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1
```

`install-codex-kit.ps1` creates the local Marketplace link `plugins/AiCoding -> CodingKit/agents/skills/plugins/AiCoding`, registers `aicoding-platform` when the Codex plugin CLI is available, and installs `aicoding@aicoding-platform`. The link is machine-local generated state and is intentionally ignored by Git.

<a id="local-agent-kits"></a>
## Local Agent Kits / 本地 Agent Kit

AiCoding publishes repository-scoped Agent Kits through the local Marketplace:

- Agent Patch Kit: `aicoding-agent-patch-kit`, sourced from `dist/agent-patch-kit/plugins/AiCodingAgentPatch`, provides the `apatch` safe patch workflow, state gates, fixed-string scan/replace, transaction snapshots, Markdown link checks, and patch summaries.
- AI Debug Repair Kit: `aicoding-ai-debug-repair-kit`, sourced from `dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit`, provides `airepair` for bounded build/test repair loops and default non-invasive TI DSS/XDS read-only debug helpers. v0.4.1 fixes the `airepair dss` workflow around `connect-test`, `core-list`, `monitor-address`, `monitor-symbol`, `find-changing-symbol`, and `report`, while keeping policy-gated J-Link invasive-operation stubs.
- AiCoding Agent Dev Kit: `aicoding-agent-dev-kit`, sourced from `dist/aicoding-agent-dev-kit/plugins/AiCodingAgentDevKit`, provides `aicoding-agent-kit` for requirement clarification, technical option matrices, Plan Mode overlay, Spec Pack, TDD planning, sequential context loading, lightweight decision memory, hook bridge, and MVP progress tracking. Plan Mode overlay documentation: [Agent Dev Kit Plan Mode](docs/AGENT_DEV_KIT_PLAN_MODE.md), [Spec Kit Adaptation](docs/SPEC_KIT_ADAPTATION.md), and [Superpower Skill Adaptation](docs/SUPERPOWER_SKILL_ADAPTATION.md).

Environment expectations:

- PowerShell 7 (`pwsh`) is the default shell for repository install, verify, status, update, and documentation checks; Windows PowerShell 5.1 is reserved for explicit compatibility gates. Git, Python 3.10+, and the Codex plugin Marketplace flow are also required.
- Agent Patch Kit uses the user-mode `apatch` CLI. Validate it with `apatch install doctor`, `apatch brief --format md`, and `apatch state status`.
- AI Debug Repair Kit installs the user-mode `ai-debug-repair-kit` Python package. Validate it with `python -m ai_debug_repair.cli version --output json`, `python -m ai_debug_repair.cli doctor --output json`, and `pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-ai-debug-repair-kit.ps1 -Json`.
- AiCoding Agent Dev Kit installs the user-mode `aicoding-agent-dev-kit` Python package. Validate it with `aicoding-agent-kit status --repo .`, `pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-aicoding-agent-dev-kit.ps1 -Json`, `pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/test-aicoding-agent-dev-kit.ps1 -Json`, `pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-dev-kit-plan-mode.ps1 -Json`, and `pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-engineering-foundation.ps1 -Json`.
- TI DSP debug flows require TI CCS/DSS, such as `C:\ti\ccs1281\ccs\ccs_base\scripting\bin\dss.bat`, plus an XDS probe and a target `.ccxml` before real hardware access. The default profile remains non-invasive: no reset, halt, run, flash, or writes.

Unified lifecycle entrypoint:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 list
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 status -All -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 verify -All
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 test -All -Profile Smoke -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 skills -All -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 verify-skills -All -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-common-code.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-hooks.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-agent-dev-kit-plan-mode.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\hooks\aef\plan-mode-gate.ps1 -Event manual -Mode warn -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\hooks\aef\spec-artifact-gate.ps1 -Event manual -Mode warn -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-agent-engineering-foundation.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-kit.ps1 export -All -Zip -DryRun
```

<a id="fast-path-v1"></a>
Fast Path V1 (Go native hot path):

Local high-frequency paths (Git hooks, Smoke checks, governance lint, performance probes) are served by the Go native CLI `bin/aicoding.exe`. `.githooks` prefer the fast CLI and fall back to PowerShell automatically; Full/Release still use the legacy entrypoints above. See `docs/AICODING_FAST_PATH_ARCHITECTURE_V1.md` and `docs/ROLLBACK_FAST_PATH_V1.md`.

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\install-fast-path-v1.ps1   # build bin/aicoding.exe and enable .githooks
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe doctor perf --json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\test-fast-path-v1.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\measure-fast-path-v1.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\rollback-fast-path-v1.ps1 -UnsetHooksPath -RemoveBinary
```

Kit Lifecycle v2.0 freezes the registry + manifest + adapter layer, skill routing, common registry, hook registry, package boundary, state boundary, and third-party/user-created skill policy. It does not rewrite legacy scripts and does not add `install-all.ps1`, `verify-all.ps1`, `test-all.ps1`, `update-all.ps1`, `uninstall-all.ps1`, or `export-all.ps1`. `-All` only walks enabled Kits in `config/kit-registry.json` and reuses the same action dispatch path as single-Kit execution. Smoke remains the default gate. `verify -All` and `test -All` default to Smoke; use `-Profile Full` only for explicit legacy full verification. Full and Release are explicit.

Typical usage:

```powershell
apatch status
apatch scan "README.md" --fixed
apatch summary

python -m ai_debug_repair.cli dss capabilities --output json
python -m ai_debug_repair.cli dss profile-template --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss validate-profile --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss connect-test --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss core-list --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --output json
python -m ai_debug_repair.cli dss monitor-address --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --address 0xB4C0 --samples 10 --output json
python -m ai_debug_repair.cli dss monitor-symbol --profile .ai-debug-repair\profiles\ti-dss-f28388d-readonly.json --out "<app.out>" --symbol "<symbol>" --samples 10 --output json
python -m ai_debug_repair.cli dss report --workspace . --output md

pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\install-aicoding-agent-dev-kit.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\status-aicoding-agent-dev-kit.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-aicoding-agent-dev-kit.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\test-aicoding-agent-dev-kit.ps1 -Json
aicoding-agent-kit clarify init --repo . --requirement "Describe the unclear requirement"
aicoding-agent-kit load --repo . --auto
```

Machine-local AI Debug Repair profiles, run scripts, session logs, DSS session evidence, and Markdown reports under `.ai-debug-repair/` are ignored by Git unless a specific test fixture is intentionally added. The default AI Debug Repair Kit policy denies reset, halt, run, loadProgram, flash, erase, write-memory, expression writes, and register writes.

## Repository Roles / 仓库角色

- `CodingKit/agents/skills` is the Git submodule pointing to `https://github.com/JiaxI2/Codex-Skills.git`.
- `CodingKit/agents/skills/plugins/AiCoding` is the installable Codex plugin package.
- `aicoding-user-skill-creator` is bundled in the AiCoding plugin as User-Skill-Creator; the system `skill-creator` remains separate.
- `.agents/plugins/marketplace.json` is the AiCoding platform Marketplace entry.
- `config/codex-kit.json` defines CodingKit asset discovery and installation rules.
- `.githooks/` contains repository Git hooks; Codex hooks live inside the plugin and require `/hooks` review.
- The AiCoding plugin bundles SDD, MVP, BDD, architecture-first, TDD fallback, and documentation synchronization workflow skills. Superpowers can be reused when installed but is not required.

## CodingKit Assets / CodingKit 资产

```text
CodingKit/examples
CodingKit/modules
CodingKit/platforms
CodingKit/tests
CodingKit/tools
```

These directories are platform assets and are not copied into the Codex plugin. Skills and tools discover them through `config/codex-kit.json`, `AICODING_HOME`, install state, PATH, project discovery, or MCP.

## Standalone Skills

AiCoding separates bundled plugin skills from personal standalone skills:

- Bundled `aicoding-*` skills are installed through the AiCoding Codex Plugin and managed by the Codex plugin cache.
- Personal or downloaded standalone skills are backed up in `Codex-Skills` and installed by profile as junctions into `%USERPROFILE%\.agents\skills` by default.
- `scripts/set-codex-skill-profile.ps1 -Profile full -DryRun -Json` shows the complete standalone skill install plan.
- Use `-StandaloneRoot codex` only when a compatibility workflow explicitly needs `%USERPROFILE%\.codex\skills`; the default is `-StandaloneRoot agents`.
- A clean compatibility runtime may keep `%USERPROFILE%\.codex\skills\.system` and selected standalone skill junctions, but `aicoding-*` must come only from the installed AiCoding plugin.

<a id="git-governance-standard"></a>
## Git Governance Standard / Git 治理标准

All AiCoding-governed Git repositories must document branch, environment, commit type, release-note, and bilingual documentation rules in README or an equivalent governance file.

- Branches: `main` or `master` is the stable production branch and must not receive direct code edits except approved release or hotfix integration; `develop` is the DEV integration branch; `feature/<scope>` branches start from `develop`; `test` maps to FAT when a shared test environment exists; `release/<version>` maps to UAT/pre-production; `hotfix/<scope>` starts from `main` and is merged back to `main` and `develop`.
- Environments: `DEV` is developer debugging, `FAT` is functional acceptance testing, `UAT` is user acceptance/pre-production, and `PRO` is production.
- Commit types: `feat` adds functionality, `fix` repairs bugs, `docs` changes documentation only, `style` changes formatting without behavior impact, `refactor` restructures code without feature or bug-fix intent, `perf` improves performance, `test` adds or corrects tests, `build` changes build or packaging behavior, `ci` changes automation, and `chore` changes supporting tools or maintenance files.
- Single commits: one commit should contain one category of change, no more than three tightly related topics, and a typed subject such as `feat(scope): summary`.
- Bilingual rule: README defaults to Chinese first and must keep visible file-level switches to `README_CN.md` and `README_EN.md`; CHANGELOG, Tag, GitHub Release, and GitHub About descriptions are Chinese first, English second.
- Releases: Tag and GitHub Release notes must group every included commit by type, state the primary release type, and include `摘要 / Summary`, `变更内容 / What's Changed`, `兼容性 / Compatibility`, `废弃项 / Deprecations`, `发布说明 / Release Notes`, `完整变更 / Full Changelog`, `新贡献者 / New Contributors`, `已知问题 / Known Issues`, `可追溯性 / Traceability`, and `资产 / Assets`.

<a id="maintenance-commands"></a>
## Maintenance Commands / 维护命令

Taskfile is the recommended human and Agent entrypoint. It only routes commands; real business logic stays in the Go CLI, PowerShell scripts, or Python scripts. The Go CLI `bin/aicoding.exe` is the local Fast Path; PowerShell/Python remain the Full/Release slow path. Full and Release must be run explicitly and are not triggered by Smoke.

Tag namespaces are separate: platform releases use `vMAJOR.MINOR.PATCH`, Kit/component releases use `kit/<kit-id>/vMAJOR.MINOR.PATCH`, and date milestones use `milestone/YYYY.MM.DD-<name>`.

```powershell
task setup
task smoke
task perf
task tag:audit
task tag:plan
task tag:verify
task full
task release
task skills
task rollback
```

When Task is unavailable, continue using the Fast Path `bin\aicoding.exe` commands and the full `scripts/aicoding-kit.ps1` / `scripts/test-kit-fresh-clone.ps1` paths.

```powershell
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/status-codex-kit.ps1 -Json
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/update-codex-kit.ps1 -DryRun
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
& "C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all
```

Do not rebuild `plugins/AiCoding` inside the AiCoding submodule. Update the submodule only after Codex-Skills has built, verified, committed, and pushed the plugin package.

## Documentation / 文档

- [中文 README](README_CN.md)
- [English README](README_EN.md)
- [Codex Kit Architecture](docs/CODEX_KIT_ARCHITECTURE.md)
- [Kit Lifecycle Architecture](docs/KIT_LIFECYCLE_ARCHITECTURE.md)
- [Kit Lifecycle Test Profiles](docs/KIT_LIFECYCLE_TEST_PROFILES.md)
- [Kit Skill Routing](docs/KIT_SKILL_ROUTING.md)
- [Common Code Management](docs/COMMON_CODE_MANAGEMENT.md)
- [Hook System](docs/HOOK_SYSTEM.md)
- [Third-Party Skill Policy](docs/THIRD_PARTY_SKILL_POLICY.md)
- [User-Created Skill Policy](docs/USER_CREATED_SKILL_POLICY.md)
- [Kit Export And Release](docs/KIT_EXPORT_AND_RELEASE.md)
- [Kit Install State](docs/KIT_INSTALL_STATE.md)
- [Agent Dev Kit Plan Mode](docs/AGENT_DEV_KIT_PLAN_MODE.md)
- [Spec Kit Adaptation](docs/SPEC_KIT_ADAPTATION.md)
- [Superpower Skill Adaptation](docs/SUPERPOWER_SKILL_ADAPTATION.md)
- [Maintenance Method](docs/MAINTENANCE_METHOD.md)
- [CodingKit](CodingKit/README.md)
- [CHANGELOG](CHANGELOG.md)
- [Repository Governance](.github/repository-governance.toml)