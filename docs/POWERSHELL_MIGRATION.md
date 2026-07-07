# PowerShell Migration

This document classifies PowerShell entrypoints after Go Fast Path expansion. It is a planning and routing document only. No scripts are moved or deleted in this round.

## Source

The classification is based on:

```powershell
bin\aicoding.exe doctor pwsh --json
```

Terms:

- Go-Replaced: deprecated for default Smoke/status/verify/lint hot paths; script remains available.
- Keep PowerShell: explicit slow path or workflow owner; do not migrate in this round.
- Planned Go Migration: possible future orchestration target; no implementation in this round.

## Go-Replaced

| PowerShell surface | Go replacement | Scope |
|---|---|---|
| `scripts/verify-hooks.ps1` | `bin\aicoding.exe verify hooks --json` | Default hook presence and fast-first verification |
| `scripts/verify-repo-text-format.ps1` | `bin\aicoding.exe verify repo-text --json` | README, CHANGELOG, and docs text checks |
| `scripts/verify-release-notes.ps1` | `bin\aicoding.exe verify release-notes --json` | CHANGELOG and release policy presence checks |
| `scripts/status-codex-kit.ps1` status summary use | `bin\aicoding.exe status --all --json` | Repo, tool, registry, manifest, required-path summary |
| `scripts/lint-git-governance.ps1` fast lint use | `bin\aicoding.exe governance lint --json` | Local governance fast lint |
| `scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json` manifest smoke use | `bin\aicoding.exe kit verify --all --profile Smoke --json` | Smoke manifest check only |
| `.githooks/pre-commit` and `.githooks/commit-msg` primary path | `bin/aicoding` or `bin/aicoding.exe` | Hooks prefer Go, then fall back to PowerShell |

These replacements do not remove the original scripts. They only remove those scripts from the default hot path.

## Keep PowerShell

| PowerShell surface | Reason |
|---|---|
| `scripts/aicoding-kit.ps1` Full/Release/install/export/uninstall/rollback paths | Complete lifecycle orchestration and compatibility semantics |
| `scripts/test-kit-fresh-clone.ps1` | Fresh clone and Release gate behavior |
| `scripts/aicoding-kit.ps1 export -All -Zip -Json` | Packaging/export ownership |
| `scripts/install-*.ps1`, `scripts/update-*.ps1`, `scripts/uninstall-*.ps1` | Installer state, Marketplace refresh, and rollback ownership |
| `scripts/rollback-fast-path-v1.ps1` | Explicit rollback workflow |
| `scripts/aicoding-tag-governance.ps1` | Non-destructive release/tag governance audit and plan |
| `scripts/verify-release-governance-overlay.ps1` | Overlay-specific compatibility check |
| PowerShell AST, PSScriptAnalyzer, and regex optimization gates | PowerShell-specific quality gates |
| DSS/XDS/hardware-related scripts, fixtures, or references | Hardware safety boundary; do not run or migrate by default |

## Planned Go Migration

| Candidate | Possible target | Notes |
|---|---|---|
| `scripts/aicoding-kit.ps1 list` summary | `bin\aicoding.exe status --all --json` extension | Plan only; no implementation in this round |
| `scripts/aicoding-kit.ps1 verify-skills -All -Json` summary | Future Go orchestration check | Keep PowerShell until skill semantics are modeled |
| `scripts/check-documentation-sync.ps1` hook mode | Existing `bin\aicoding.exe hook pre-commit --json` path | Keep fallback until docsync parity is explicit |
| `scripts/aicoding-skill.ps1 sources -Json` | Future registry/source status view | Plan only |
| `scripts/status-*-kit.ps1` detailed status | Future status detail subcommands | Current `status --all` covers repository summary only |

## Default Entry Decision

Default local Smoke/status/verify/lint/doctor entrypoints should use Go:

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
bin\aicoding.exe status --all --json
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor perf --json
```

PowerShell remains the explicit Full/Release, install/uninstall/export/fresh clone, rollback, and compatibility lane.

## No-Delete Rule

This round marks deprecated default hot-path usage only. Do not move files to `scripts/legacy/` and do not delete scripts without a separate migration plan and user confirmation.
