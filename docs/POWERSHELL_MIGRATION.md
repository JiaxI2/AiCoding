# PowerShell Migration

This document classifies PowerShell entrypoints after Go Fast Path V2. It is a routing document only. Go-replaced fast-path PowerShell scripts have been removed from the working tree after Go Fast Path V2 reached parity. Historical copies remain available through Git history only.

## Source

The primary inventories are:

```powershell
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
```

Budget categories:

- `hot-path`: default development or Smoke route that should prefer Go.
- `slow-path`: complete lifecycle, compatibility, Full/Release, or PowerShell-owned gate.
- legacy compatibility inventory: retained paths for explicit comparison; not a default route.
- `documentation-only`: command examples or migration notes in docs.

## Go-Replaced Default Paths

| PowerShell surface | Go replacement | Scope |
|---|---|---|
| `task setup` PowerShell installer probe | `go run ./cmd/aicoding bootstrap --json` | Build/check `bin/aicoding.exe` without PowerShell |
| `scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json` default smoke use | `bin\aicoding.exe kit verify --all --profile Smoke --json` | Smoke manifest check only |
| Removed legacy fast lint script (`lint-git-governance.ps1`; Git history only) | `bin\aicoding.exe governance lint --json` | Local governance fast lint |
| Removed legacy hook verification script (`verify-hooks.ps1`; Git history only) | `bin\aicoding.exe verify hooks --json` | Default hook presence and fast-first verification |
| Removed legacy repo-text script (`verify-repo-text-format.ps1`; Git history only) | `bin\aicoding.exe verify repo-text --json` | README, CHANGELOG, and docs text checks |
| Removed legacy release-notes script (`verify-release-notes.ps1`; Git history only) | `bin\aicoding.exe verify release-notes --json` | CHANGELOG and release policy presence checks |
| Removed legacy status script (`status-codex-kit.ps1`; Git history only) | `bin\aicoding.exe status --all --json` | Repo, tool, registry, manifest, required-path summary |
| PowerShell inventory review | `bin\aicoding.exe doctor pwsh-budget --json` | Classifies hot/slow/compatibility/docs-only invocation points |
| Tag audit summary | `bin\aicoding.exe tag audit --json` | Structural namespace classification; legacy tags are warnings |
| Release structure summary | `bin\aicoding.exe release verify --json` | Structural release/template/tag-policy fast check |
| All-kit lifecycle dry-run/status aggregation | `bin\aicoding.exe kit lifecycle --action install|update|uninstall --all --dry-run --json`; `bin\aicoding.exe kit lifecycle --action status --all --json` | Static registry/manifest planner; does not execute lifecycle adapters |
| `scripts/verify-kit-lifecycle.ps1 -Json` structural subset | `bin\aicoding.exe kit verify --all --profile Lifecycle --json` | Go-native registry/manifest/required-path/dry-run policy verification |
| `scripts/verify-codex-kit.ps1 -Json` structural subset | `bin\aicoding.exe kit verify --all --profile Lifecycle --json` | Go-native codex-kit config, Marketplace path, package layout, and lifecycle structure verification |

These replacements remove PowerShell from the default hot path. The Go-replaced legacy scripts have been removed from the working tree and are available only through Git history.

## Smart Verify

```powershell
bin\aicoding.exe workflow smart-verify --json
```

`workflow smart-verify` reads Git staged, changed, and untracked files, emits the selected plan, and runs existing Go quick checks. It deliberately excludes Full, Release, real install/uninstall, export, rollback, fresh clone, DSS, and PSScriptAnalyzer work.

## Keep PowerShell

| PowerShell surface | Reason |
|---|---|
| `scripts/aicoding-kit.ps1` Full/Release and real install/update/export/uninstall/rollback paths | Complete lifecycle orchestration and compatibility semantics |
| `scripts/test-kit-fresh-clone.ps1` | Fresh clone and Release gate behavior |
| `scripts/verify-codex-kit.ps1` | Compatibility/full codex-kit verification, submodule plugin verifier orchestration, and optional fresh clone integration |
| `scripts/verify-kit-lifecycle.ps1` | Compatibility/full lifecycle verification, external PowerShell script probes, and adapter parity checks |
| `scripts/aicoding-kit.ps1 export -All -Zip -Json` | Packaging/export ownership |
| `scripts/install-*.ps1`, `scripts/update-*.ps1`, `scripts/uninstall-*.ps1` | Installer state, Marketplace refresh, and rollback ownership |
| `scripts/rollback-fast-path-v1.ps1` | Explicit rollback workflow |
| `scripts/aicoding-tag-governance.ps1 -Action Plan` | Non-destructive legacy tag correction planning |
| `scripts/verify-release-governance-overlay.ps1` | Overlay-specific compatibility check |
| `scripts/aicoding-kit.ps1 verify-skills -All -Json` | Skill semantics and compatibility verification |
| PowerShell AST, PSScriptAnalyzer, and regex optimization gates | PowerShell-specific quality gates |
| DSS/XDS/hardware-related scripts, fixtures, or references | Hardware safety boundary; do not run or migrate by default |

The PowerShell Skill Kit pass gate remains scoped to `tools/`, `hooks/`, and `tests/cases/good`. `tests/cases/bad` and `tests/cases/rewrite` remain negative fixtures and must not be promoted to recursive CI blockers.

## Cache Boundary

```powershell
bin\aicoding.exe cache status --json
bin\aicoding.exe cache clean --json
```

Fast Path V2 cache state is diagnostic only. It is the base for later incremental verify, but this version never lets cache hits or misses decide pass/fail.

## Default Entry Decision

Default local Smoke/status/verify/lint/doctor entrypoints should use Go:

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe workflow smart-verify --json
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
bin\aicoding.exe status --all --json
bin\aicoding.exe kit verify --all --profile Lifecycle --json
bin\aicoding.exe kit lifecycle --action update --all --dry-run --json
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe doctor perf --json
bin\aicoding.exe tag audit --json
bin\aicoding.exe release verify --json
```

Default `task perf` is Go-only. PowerShell parity timing remains explicit through `scripts/measure-fast-path-v1.ps1 -Json` or lifecycle adapter dry-runs when compatibility comparison is needed.

PowerShell remains the explicit Full/Release, real install/update/uninstall/export/fresh clone, rollback, skill verification, and compatibility lane.

## Legacy Boundary

Go-replaced fast-path PowerShell scripts have been removed from the working tree and are available only through Git history. Do not delete remaining PowerShell slow paths or change Full/Release, real install/update/uninstall/export/rollback, fresh clone, DSS, or PSScriptAnalyzer semantics without a separate migration plan and user confirmation.
