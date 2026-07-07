# PowerShell Migration

This document classifies PowerShell entrypoints after Go Fast Path V2. It is a routing document only. The Go-replaced fast-path PowerShell scripts now live under `scripts/legacy/fast-path-replaced/` as explicit legacy fallback or historical tooling.

## Source

The primary inventories are:

```powershell
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
```

Budget categories:

- `hot-path`: default development or Smoke route that should prefer Go.
- `slow-path`: complete lifecycle, compatibility, Full/Release, or PowerShell-owned gate.
- `fallback`: retained compatibility path after a Go-first attempt.
- `documentation-only`: command examples or migration notes in docs.

## Go-Replaced Default Paths

| PowerShell surface | Go replacement | Scope |
|---|---|---|
| `task setup` PowerShell installer probe | `go run ./cmd/aicoding bootstrap --json` | Build/check `bin/aicoding.exe` without PowerShell |
| `scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json` default smoke use | `bin\aicoding.exe kit verify --all --profile Smoke --json` | Smoke manifest check only |
| `scripts/legacy/fast-path-replaced/lint-git-governance.ps1` fast lint use | `bin\aicoding.exe governance lint --json` | Local governance fast lint |
| `scripts/legacy/fast-path-replaced/verify-hooks.ps1` | `bin\aicoding.exe verify hooks --json` | Default hook presence and fast-first verification |
| `scripts/legacy/fast-path-replaced/verify-repo-text-format.ps1` | `bin\aicoding.exe verify repo-text --json` | README, CHANGELOG, and docs text checks |
| `scripts/legacy/fast-path-replaced/verify-release-notes.ps1` | `bin\aicoding.exe verify release-notes --json` | CHANGELOG and release policy presence checks |
| `scripts/legacy/fast-path-replaced/status-codex-kit.ps1` summary use | `bin\aicoding.exe status --all --json` | Repo, tool, registry, manifest, required-path summary |
| PowerShell inventory review | `bin\aicoding.exe doctor pwsh-budget --json` | Classifies hot/slow/fallback/docs-only invocation points |
| Tag audit summary | `bin\aicoding.exe tag audit --json` | Structural namespace classification; legacy tags are warnings |
| Release structure summary | `bin\aicoding.exe release verify --json` | Structural release/template/tag-policy fast check |

These replacements remove PowerShell from the default hot path only. The moved legacy scripts remain available as explicit fallback or historical tooling.

## Smart Verify

```powershell
bin\aicoding.exe workflow smart-verify --json
```

`workflow smart-verify` reads Git staged, changed, and untracked files, emits the selected plan, and runs existing Go quick checks. It deliberately excludes Full, Release, install, uninstall, export, rollback, fresh clone, DSS, and PSScriptAnalyzer work.

## Keep PowerShell

| PowerShell surface | Reason |
|---|---|
| `scripts/aicoding-kit.ps1` Full/Release/install/export/uninstall/rollback paths | Complete lifecycle orchestration and compatibility semantics |
| `scripts/test-kit-fresh-clone.ps1` | Fresh clone and Release gate behavior |
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
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe doctor perf --json
bin\aicoding.exe tag audit --json
bin\aicoding.exe release verify --json
```

PowerShell remains the explicit Full/Release, install/uninstall/export/fresh clone, rollback, skill verification, and compatibility lane.

## Legacy Boundary

Go-replaced fast-path PowerShell scripts are already isolated under `scripts/legacy/fast-path-replaced/`. Do not delete remaining PowerShell slow paths or change Full/Release, install/update/uninstall/export/rollback, fresh clone, DSS, or PSScriptAnalyzer semantics without a separate migration plan and user confirmation.