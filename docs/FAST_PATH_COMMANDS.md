# Fast Path Commands

Go Fast Path commands are the default local hot path. They are designed for Smoke, status, verify, lint, and doctor checks where repeated PowerShell startup is unnecessary.

## Build

```powershell
go build -o bin/aicoding.exe ./cmd/aicoding
```

The binary is a local build artifact and must not be committed.

## Recommended Smoke Chain

`task smoke` routes to these Go commands:

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
bin\aicoding.exe doctor perf --json
```

The PR/push fast workflow uses the same Go-native smoke lane after `go test ./...` and `go build -o bin/aicoding ./cmd/aicoding`. It does not call the legacy PowerShell fast-path scripts as default CI.

## Status And Doctor

```powershell
bin\aicoding.exe status --all --json
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor perf --json
```

`status --all` summarizes repo root, branch, Go/Git availability, tool discovery, kit registry, manifests, and required paths. It does not run PowerShell workflow scripts.

`doctor pwsh` scans configured repository surfaces for PowerShell invocation points and returns category plus migration advice.

`doctor perf` measures local Fast Path probes. It is not a Full/Release benchmark.

## Verify Commands

```powershell
bin\aicoding.exe verify hooks --json
bin\aicoding.exe verify repo-text --json
bin\aicoding.exe verify release-notes --json
```

- `verify hooks`: checks `.githooks/pre-commit` and `.githooks/commit-msg` exist and prefer the Go fast path before PowerShell fallback.
- `verify repo-text`: checks README, CHANGELOG, and docs text files for conflict markers, empty files, invalid UTF-8, and line-ending warnings.
- `verify release-notes`: checks CHANGELOG, release/tag policy documents, and release-governance overlay files exist.

## Governance And Kit Smoke

```powershell
bin\aicoding.exe governance lint --json
bin\aicoding.exe kit verify --all --profile Smoke --json
```

These are the default local checks for high-frequency development. Full and Release still use explicit slow-path tasks.

## PowerShell Regex Fast Lint

```powershell
bin\aicoding.exe powershell regex-lint --staged --json
bin\aicoding.exe powershell regex-lint --path scripts --json
```

This is a fast lint surface for common PowerShell regex hazards. Full PowerShell AST/PSScriptAnalyzer gates remain slow-path tooling.

The PowerShell Skill Kit compatibility gate remains PowerShell-owned. Its default pass gate is scoped to `tools/`, `hooks/`, and `tests/cases/good`; `tests/cases/bad` and `tests/cases/rewrite` are negative fixtures and must not be treated as recursive CI blockers.

## Out Of Scope For This Round

- No smart verify implementation.
- No cache implementation.
- No Full/Release semantic rewrite.
- No install/uninstall/export/fresh clone migration.
- No hardware action execution.
