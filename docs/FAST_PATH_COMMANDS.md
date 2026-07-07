# Fast Path Commands

Go Fast Path commands are the default local hot path for repeated development checks. Fast Path V2 keeps the checks structural, JSON-readable, and Go-native while preserving PowerShell/Python as the explicit slow path for complete lifecycle and release semantics.

## Bootstrap

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe bootstrap --json
```

`bootstrap` checks repo root, `go.mod`, `.git`, Git, Go, and `bin/`, then builds `bin/aicoding.exe` by default. Use `--no-build` only for diagnostics. The command creates `bin/` when needed and does not call PowerShell.

## Smart Verify

```powershell
bin\aicoding.exe workflow smart-verify --json
```

`workflow smart-verify` reads staged, changed, and untracked files from Git, builds a file-type plan, and executes only Go fast checks for the first V2 loop. It does not call Full, Release, install, uninstall, export, rollback, fresh clone, DSS, or PSScriptAnalyzer paths.

Selected checks include:

- `go test ./...` when Go source or `go.mod` changed;
- kit Smoke manifest verification for kit registry, manifest, Taskfile, or CodingKit surfaces;
- governance lint for README, CHANGELOG, Taskfile, and GitHub metadata surfaces;
- hook, repo-text, and release-notes verification for their matching file types.

## Cache

```powershell
bin\aicoding.exe cache status --json
bin\aicoding.exe cache clean --json
```

The V2 cache is stored under `.aicoding/cache/fast-path-v2`. The parent `.aicoding/cache/` directory is ignored and must not be committed. In this first version it is reporting-only and cleanup-only; cache state never changes pass/fail results.

## Recommended Smoke Chain

`task smoke` remains Go-native:

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
bin\aicoding.exe doctor perf --json
```

CI fast smoke builds the Linux binary and runs Go tests before the same Smoke checks:

```bash
go build -o bin/aicoding ./cmd/aicoding
go test ./...
./bin/aicoding kit verify --all --profile Smoke --json
./bin/aicoding governance lint --json
./bin/aicoding verify hooks --json
./bin/aicoding verify repo-text --json
./bin/aicoding verify release-notes --json
./bin/aicoding doctor perf --json
```

Default Smoke does not call PowerShell. Full and Release remain explicit slow-path tasks.

## Status And Doctor

```powershell
bin\aicoding.exe status --all --json
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe doctor perf --json
```

`doctor pwsh-budget` scans `Taskfile.yml`, `.githooks`, `.github/workflows`, `scripts`, and `docs`, then classifies PowerShell invocation points as `hot-path`, `slow-path`, `fallback`, or `documentation-only`.

## Governance And Release

```powershell
bin\aicoding.exe tag audit --json
bin\aicoding.exe release verify --json
```

`tag audit` classifies local tags into platform, kit, milestone, legacy, and unknown namespaces. Legacy tags are warnings in the JSON payload, not failures.

`release verify` is a structural fast check for CHANGELOG, release template, tag policy docs, overlay files, and malformed release-note text. It does not replace `scripts/verify-release-governance-overlay.ps1` or Release profile gates.

## Verify Commands

```powershell
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
```

- `verify hooks`: checks `.githooks/pre-commit` and `.githooks/commit-msg` exist and prefer the Go fast path before PowerShell fallback.
- `verify repo-text`: checks README, CHANGELOG, and docs text files for conflict markers, empty files, invalid UTF-8, and line-ending warnings.
- `verify release-notes`: checks CHANGELOG, release/tag policy documents, release-governance overlay files, and the release template for malformed Markdown fences or control/replacement characters.

## JSON And Exit Code Contract

All Fast Path V2 commands support `--json` and return the common `report.Result` envelope:

```json
{
  "schemaVersion": 1,
  "command": "...",
  "ok": true,
  "repoRoot": "...",
  "data": {},
  "elapsedMs": 0
}
```

Stable exit code policy:

- `0`: command completed and `ok` is true;
- `1`: command completed with structural errors or failed execution;
- `2`: CLI usage error before command execution.

## Maintained Docs Link Check

Use this as the default Markdown link check after maintained docs change:

```powershell
apatch links --mode offline --include-fragments full --input README.md --input README_CN.md --input README_EN.md --input CHANGELOG.md --input "docs/*.md" --input ".github/workflows/*.yml"
```

Run full repository link audit explicitly with `apatch links --mode offline --include-fragments full` when templates, generated assets, fixtures, and historical archives must be included.

## PowerShell Slow Path Boundary

PowerShell remains the explicit owner for Full/Release profiles, install/update/uninstall/export/rollback, fresh clone validation, skill verification, release overlay compatibility, PSScriptAnalyzer/PowerShell AST gates, and DSS/XDS/hardware-related flows.

No `scripts/*.ps1` file is moved, deleted, or placed under `legacy/` by Fast Path V2.