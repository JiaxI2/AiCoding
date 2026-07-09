# Fast Path Commands

Go Fast Path commands are the default local hot path for repeated development checks. After Go-native consolidation, the same Go CLI also owns the default Full, Release gate, lifecycle, export, fresh-clone, DocSync, and skill verification routes. PowerShell/Python remains explicit compatibility and specialty tooling.

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

`workflow smart-verify` reads staged, changed, and untracked files from Git, builds a file-type plan, and executes selected Go checks. It stays fast and does not run release export, fresh clone, DSS/XDS, or PSScriptAnalyzer paths.

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

The V2 cache is stored under `.aicoding/cache/fast-path-v2`. The parent `.aicoding/cache/` directory is ignored and must not be committed. Cache state is reporting-only and cleanup-only; cache state never changes pass/fail results.

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

The default CI smoke workflow on Windows builds the Go CLI and runs:

```powershell
go test ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin\aicoding.exe docsync ci --json
bin\aicoding.exe skill verify --all --profile Smoke --json
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe full --json
bin\aicoding.exe export --all --zip --json
```

Default Smoke does not call PowerShell. Full and Release are explicit Go aggregate tasks.

## Kit Lifecycle Structure Verify

```powershell
bin\aicoding.exe kit verify --all --profile Lifecycle --json
```

`kit verify --profile Lifecycle` is the Go-native default for codex-kit and kit lifecycle structural verification. It checks `config/codex-kit.json`, the kit registry, manifests, command envelopes, required paths, generated package warnings, and all-kit dry-run skip policy without running PowerShell adapters or fresh clone gates.

PowerShell `verify-codex-kit.ps1` remains an explicit compatibility check and is not the default gate.

## Lifecycle, Export, And Fresh Clone

```powershell
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe lifecycle plan --action update --all --json
bin\aicoding.exe lifecycle install --all --json
bin\aicoding.exe lifecycle update --all --json
bin\aicoding.exe lifecycle uninstall --all --json
bin\aicoding.exe lifecycle rollback --last --json
bin\aicoding.exe export --all --zip --json
bin\aicoding.exe fresh-clone --profile Smoke --json
```

These are Go CLI default routes. They use registry and manifest data, produce JSON envelopes, and keep Taskfile as routing only. Manifest-declared PowerShell commands may remain for compatibility or specialty workflows, but they are not the default lifecycle/export/fresh-clone entrypoints.

## Status And Doctor

```powershell
bin\aicoding.exe status --all --json
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe doctor perf --json
```

`doctor pwsh-budget` scans `Taskfile.yml`, `.githooks`, `.github/workflows`, `scripts`, and `docs`, then classifies PowerShell invocation points by route budget.

Default `task perf` maps to Go-native `doctor perf` only. Run PowerShell parity comparisons explicitly from [COMMANDS.md](COMMANDS.md#explicit-powershell-parity-checks) when compatibility timing is needed.

## Governance And Release

```powershell
bin\aicoding.exe tag audit --json
bin\aicoding.exe release verify --json
bin\aicoding.exe release gate --json
```

`tag audit` classifies local tags into platform, kit, milestone, legacy, and unknown namespaces. Legacy tags are warnings in the JSON payload, not failures.

`release verify` is a structural check for CHANGELOG, release template, tag policy docs, overlay files, and malformed release-note text. `release gate` is the Go-native release aggregate.

## Verify Commands

```powershell
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
```

- `verify hooks`: checks `.githooks/pre-commit` and `.githooks/commit-msg` exist and prefer the Go fast path before explicit PowerShell compatibility paths.
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

## PowerShell Boundary

PowerShell remains for compatibility and specialty workflows only:

- tag planning and release-governance overlay compatibility;
- PowerShell AST, PSScriptAnalyzer, regex, and PowerShell-specific quality gates;
- external third-party skill install/audit flows;
- Plan Mode helper scripts;
- safety-specific tooling;
- DSS/XDS/hardware or toolchain diagnostics.

Go-replaced Fast Path V1 wrapper/install/test/measure scripts are removed from current source. Remaining PowerShell scripts stay because they map to one of the compatibility or specialty categories above.
