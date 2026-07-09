# Commands

This document keeps the command matrix out of the README. Taskfile is the recommended human and agent entrypoint; it routes to the Go CLI and does not own business logic.

C/H formatting commands are documented in [C Style Format Kit](C_STYLE_FORMAT_KIT.md).

## Default Local Commands

| Purpose | Command | Lane |
|---|---|---|
| Bootstrap Go CLI | `go run ./cmd/aicoding bootstrap --json` | Go |
| Smoke | `task smoke` | Go |
| Smart verify | `bin\aicoding.exe workflow smart-verify --json` | Go |
| DocSync CI | `bin\aicoding.exe docsync ci --json` | Go |
| Skill verification | `bin\aicoding.exe skill verify --all --profile Full --json` | Go |
| Lifecycle plan | `bin\aicoding.exe lifecycle plan --action install --all --json` | Go |
| Lifecycle install/update/uninstall | `bin\aicoding.exe lifecycle install --all --json` | Go |
| Lifecycle rollback | `bin\aicoding.exe lifecycle rollback --last --json` | Go |
| Export | `bin\aicoding.exe export --all --zip --json` | Go |
| Fresh clone | `bin\aicoding.exe fresh-clone --profile Smoke --json` | Go |
| Full aggregate | `bin\aicoding.exe full --json` | Go |
| Release gate | `bin\aicoding.exe release gate --json` | Go |

## Go Native Checks

| Purpose | Command |
|---|---|
| Bootstrap binary | `bin\aicoding.exe bootstrap --json` |
| Smart verify plan + selected checks | `bin\aicoding.exe workflow smart-verify --json` |
| DocSync staged | `bin\aicoding.exe docsync staged --json` |
| DocSync all | `bin\aicoding.exe docsync all --json` |
| DocSync CI | `bin\aicoding.exe docsync ci --json` |
| DocSync release | `bin\aicoding.exe docsync release --json` |
| Skill Smoke verification | `bin\aicoding.exe skill verify --all --profile Smoke --json` |
| Skill Full verification | `bin\aicoding.exe skill verify --all --profile Full --json` |
| Skill Release verification | `bin\aicoding.exe skill verify --all --profile Release --json` |
| Kit Smoke | `bin\aicoding.exe kit verify --all --profile Smoke --json` |
| Kit Lifecycle structure verify | `bin\aicoding.exe kit verify --all --profile Lifecycle --json` |
| Lifecycle install plan | `bin\aicoding.exe lifecycle plan --action install --all --json` |
| Lifecycle update plan | `bin\aicoding.exe lifecycle plan --action update --all --json` |
| Lifecycle install | `bin\aicoding.exe lifecycle install --all --json` |
| Lifecycle update | `bin\aicoding.exe lifecycle update --all --json` |
| Lifecycle uninstall | `bin\aicoding.exe lifecycle uninstall --all --json` |
| Rollback last lifecycle snapshot | `bin\aicoding.exe lifecycle rollback --last --json` |
| Export release bundle | `bin\aicoding.exe export --all --zip --json` |
| Fresh clone Smoke | `bin\aicoding.exe fresh-clone --profile Smoke --json` |
| Fresh clone Release | `bin\aicoding.exe fresh-clone --profile Release --json` |
| Full aggregate | `bin\aicoding.exe full --json` |
| Release aggregate | `bin\aicoding.exe release gate --json` |
| Governance lint | `bin\aicoding.exe governance lint --json` |
| C style formatter status | `bin\aicoding.exe cstyle status --json` |
| C style comment template validation | `bin\aicoding.exe cstyle templates --json` |
| C style format changed files | `bin\aicoding.exe cstyle fmt --scope changed --json` |
| C style check changed files | `bin\aicoding.exe cstyle check --scope changed --json` |
| Hook verification | `bin\aicoding.exe verify hooks --json` |
| Repo text verification | `bin\aicoding.exe verify repo-text --json` |
| Release notes/overlay verification | `bin\aicoding.exe verify release-notes --json` |
| Performance probes | `bin\aicoding.exe doctor perf --json` |
| PowerShell inventory | `bin\aicoding.exe doctor pwsh --json` |
| PowerShell budget | `bin\aicoding.exe doctor pwsh-budget --json` |
| PowerShell regex lint | `bin\aicoding.exe powershell regex-lint --staged --json` |
| Tag namespace audit | `bin\aicoding.exe tag audit --json` |
| Release structural verify | `bin\aicoding.exe release verify --json` |

## Current CI Smoke

`.github/workflows/fast-path.yml` uses this Go-native chain on Windows for PR and push:

```powershell
go test ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin\aicoding.exe docsync ci --json
bin\aicoding.exe skill verify --all --profile Smoke --json
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe full --json
bin\aicoding.exe export --all --zip --json
```

The export step removes generated `dist/aicoding-kit-*` artifacts after the check. Fresh-clone Release stays on manual or scheduled release jobs to avoid slowing every push.

DocSync CI also builds the Go CLI and runs:

```powershell
bin\aicoding.exe docsync ci --json
```

## Taskfile Routes

| Task | Meaning | Lane |
|---|---|---|
| `task setup` | Bootstrap the Go CLI binary | Go |
| `task smoke` | Fast local Smoke gate | Go |
| `task perf` | Go-native performance probes | Go |
| `task full` | Full aggregate validation | Go |
| `task release` | Release gate with release-only checks | Go |
| `task skills` | Skill verification | Go |
| `task rollback` | Roll back last lifecycle state snapshot | Go |
| `task tag:audit` | Tag namespace audit | Go |
| `task style:c:status` | C style formatter status | Go |
| `task fmt:c` | Format changed C/H files | Go |
| `task fmt-check:c` | Check changed C/H file formatting | Go |
| `task tag:plan` | Non-destructive tag correction plan | PowerShell compatibility |
| `task tag:verify` | Release governance overlay compatibility check | PowerShell compatibility |

## Export Artifacts

`bin\aicoding.exe export --all --zip --json` writes:

- `dist/aicoding-kit-<version>.zip`
- `dist/aicoding-kit-<version>.zip.manifest.json`
- `dist/aicoding-kit-<version>.zip.sha256`

The export manifest records stable relative paths, file sizes, SHA-256 hashes, generated time, version, branch, and commit. Generated export artifacts are ignored by Git.

## Explicit PowerShell Parity Checks

PowerShell remains only for workflows not fully replaced in Go, including:

- tag migration planning;
- release governance overlay compatibility;
- PowerShell regex quality;
- external third-party skill install/audit;
- Plan Mode helper scripts;
- safety-specific tooling.

## Link Checks

Default maintained-docs link check:

```powershell
apatch links --mode offline --include-fragments full `
  --input README.md `
  --input README_CN.md `
  --input README_EN.md `
  --input CHANGELOG.md `
  --input "docs/*.md" `
  --input ".github/workflows/*.yml"
```

Full repository link audit remains explicit:

```powershell
apatch links --mode offline --include-fragments full
```

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

Do not use repository commands to perform DSS/XDS/reset/halt/run/flash/erase/write-memory actions.
Hardware-related code and fixtures are documentation or test assets unless a separate approved hardware workflow exists.
