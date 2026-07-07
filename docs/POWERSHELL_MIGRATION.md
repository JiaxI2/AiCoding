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
- Planned Cleanup: old hot-path references to remove or mark; no implementation in this round.

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

## Planned Cleanup

| Candidate | Cleanup target | Notes |
|---|---|---|
| `scripts/aicoding-kit.ps1 list` summary | Link to `bin\aicoding.exe status --all --json` where only repo summary is needed | Keep PowerShell for kit lifecycle detail |
| `scripts/aicoding-kit.ps1 verify-skills -All -Json` summary | Keep as explicit compatibility lane | Skill semantics remain PowerShell-owned |
| `scripts/check-documentation-sync.ps1` hook mode | Keep as hook fallback only | Default hook path is Go; full docsync stays PowerShell |
| `scripts/aicoding-skill.ps1 sources -Json` | Keep as skill-source workflow owner | Do not add a Go command in this round |
| `scripts/status-*-kit.ps1` detailed status | Mark as slow-path status detail | Current `status --all` covers repository summary only |

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
