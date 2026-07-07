# Commands

This document keeps the command matrix out of the README. Taskfile is the recommended human and agent entrypoint; it should route commands, not own business logic.

## Default Local Commands

| Purpose | Command | Lane |
|---|---|---|
| Bootstrap Go CLI | `go run ./cmd/aicoding bootstrap --json` | Go |
| Smoke | `task smoke` | Go default |
| Smart verify | `bin\aicoding.exe workflow smart-verify --json` | Go |
| Performance probe | `task perf` | Go plus PowerShell comparison |
| Status summary | `bin\aicoding.exe status --all --json` | Go |
| PowerShell budget | `bin\aicoding.exe doctor pwsh-budget --json` | Go |
| Tag audit | `bin\aicoding.exe tag audit --json` | Go |
| Release structure | `bin\aicoding.exe release verify --json` | Go |

## Go Native Checks

| Purpose | Command |
|---|---|
| Bootstrap binary | `bin\aicoding.exe bootstrap --json` |
| Smart verify plan + selected checks | `bin\aicoding.exe workflow smart-verify --json` |
| Cache status | `bin\aicoding.exe cache status --json` |
| Cache clean | `bin\aicoding.exe cache clean --json` |
| Kit Smoke | `bin\aicoding.exe kit verify --all --profile Smoke --json` |
| Governance lint | `bin\aicoding.exe governance lint --json` |
| Hook verification | `bin\aicoding.exe verify hooks --json` |
| Repo text verification | `bin\aicoding.exe verify repo-text --json` |
| Release notes/overlay verification | `bin\aicoding.exe verify release-notes --json` |
| Performance probes | `bin\aicoding.exe doctor perf --json` |
| PowerShell inventory | `bin\aicoding.exe doctor pwsh --json` |
| PowerShell budget | `bin\aicoding.exe doctor pwsh-budget --json` |
| PowerShell regex lint | `bin\aicoding.exe powershell regex-lint --staged --json` |
| Tag namespace audit | `bin\aicoding.exe tag audit --json` |
| Release structural verify | `bin\aicoding.exe release verify --json` |

## Default CI Smoke

PR/push fast CI builds the Go CLI, runs `go test ./...`, then runs the same Go native Smoke checks listed above. Legacy PowerShell fast-path scripts are retained for fallback or explicit slow-path compatibility, not as the default CI smoke lane.

## Link Checks

Default maintained-docs link check:

```powershell
apatch links --mode offline --include-fragments full --input README.md --input README_CN.md --input README_EN.md --input CHANGELOG.md --input "docs/*.md" --input ".github/workflows/*.yml"
```

Full repository link audit remains explicit:

```powershell
apatch links --mode offline --include-fragments full
```

The default check excludes templates, generated plugin/submodule assets, runtime mirrors, cache/report output, and external fixtures from the blocker path.

## Taskfile Routes

| Task | Meaning | Lane |
|---|---|---|
| `task setup` | Bootstrap the Go Fast Path binary | Go |
| `task smoke` | Fast local Smoke gate | Go |
| `task perf` | Fast performance probes and legacy comparison | Go + PowerShell comparison |
| `task full` | Explicit Full validation | PowerShell slow path |
| `task release` | Explicit Release and export gate | PowerShell slow path |
| `task skills` | Skill verification | PowerShell slow path |
| `task rollback` | Roll back Fast Path installation | PowerShell slow path |
| `task tag:audit` | Tag namespace audit | Go |
| `task tag:plan` | Non-destructive tag correction plan | PowerShell slow path |
| `task tag:verify` | Release governance overlay compatibility check | PowerShell slow path |

## Explicit Slow Paths

Use these only when the workflow requires complete semantics:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-kit.ps1 test -All -Profile Full -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/test-kit-fresh-clone.ps1 -Profile Release -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-kit.ps1 export -All -Zip -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-kit.ps1 verify-skills -All -Json
```

Install, update, uninstall, rollback, fresh clone, release, export, DSS, and PSScriptAnalyzer workflows remain PowerShell/Python-owned.

## Tag Governance

Fast structural audit:

```powershell
bin\aicoding.exe tag audit --json
```

Slow-path planning and overlay compatibility:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-tag-governance.ps1 -Action Plan
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-release-governance-overlay.ps1 -Json
```

These commands do not create or push tags unless a separate explicit tag operation is requested and confirmed.

## Safety Boundary

Do not use repository commands to perform DSS/XDS/reset/halt/run/flash/erase/write-memory actions. Hardware-related code and fixtures are documentation or test assets unless a separate approved hardware workflow exists.