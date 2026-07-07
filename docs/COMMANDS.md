# Commands

This document keeps the command matrix out of the README. Taskfile is the recommended human and agent entrypoint; it should route commands, not own business logic.

## Default Local Commands

| Purpose | Command | Lane |
|---|---|---|
| Build Go CLI | `go build -o bin/aicoding.exe ./cmd/aicoding` | Go |
| Smoke | `task smoke` | Go default |
| Performance probe | `task perf` | Go plus PowerShell comparison |
| Status summary | `bin\aicoding.exe status --all --json` | Go |
| PowerShell inventory | `bin\aicoding.exe doctor pwsh --json` | Go |

## Go Native Checks

| Purpose | Command |
|---|---|
| Kit Smoke | `bin\aicoding.exe kit verify --all --profile Smoke --json` |
| Governance lint | `bin\aicoding.exe governance lint --json` |
| Hook verification | `bin\aicoding.exe verify hooks --json` |
| Repo text verification | `bin\aicoding.exe verify repo-text --json` |
| Release notes/overlay verification | `bin\aicoding.exe verify release-notes --json` |
| Performance probes | `bin\aicoding.exe doctor perf --json` |
| PowerShell regex lint | `bin\aicoding.exe powershell regex-lint --staged --json` |

## Default CI Smoke

PR/push fast CI builds the Go CLI, runs `go test ./...`, then runs the same Go native Smoke checks listed above. Legacy PowerShell fast-path scripts are retained for fallback or explicit slow-path compatibility, not as the default CI smoke lane.

## Taskfile Routes

| Task | Meaning | Lane |
|---|---|---|
| `task setup` | Build or enable Go Fast Path when installer exists | PowerShell route / Go build |
| `task smoke` | Fast local Smoke gate | Go |
| `task perf` | Fast performance probes and legacy comparison | Go + PowerShell comparison |
| `task full` | Explicit Full validation | PowerShell slow path |
| `task release` | Explicit Release and export gate | PowerShell slow path |
| `task skills` | Skill verification | PowerShell slow path |
| `task rollback` | Roll back Fast Path installation | PowerShell slow path |
| `task tag:audit` | Tag namespace audit | PowerShell slow path |
| `task tag:plan` | Non-destructive tag correction plan | PowerShell slow path |
| `task tag:verify` | Release governance overlay check | PowerShell slow path |

## Explicit Slow Paths

Use these only when the workflow requires complete semantics:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-kit.ps1 test -All -Profile Full -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/test-kit-fresh-clone.ps1 -Profile Release -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-kit.ps1 export -All -Zip -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-kit.ps1 verify-skills -All -Json
```

Install, update, uninstall, rollback, fresh clone, release, and export scripts remain PowerShell/Python-owned.

## Tag Governance

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-tag-governance.ps1 -Action Audit -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/aicoding-tag-governance.ps1 -Action Plan
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-release-governance-overlay.ps1 -Json
```

These commands do not create or push tags unless a separate explicit tag operation is requested and confirmed.

## Safety Boundary

Do not use repository commands to perform DSS/XDS/reset/halt/run/flash/erase/write-memory actions. Hardware-related code and fixtures are documentation or test assets unless a separate approved hardware workflow exists.
