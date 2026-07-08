# PowerShell Migration

This document classifies PowerShell entrypoints after Go-native consolidation.
Go-replaced fast-path scripts and lifecycle/export/fresh-clone/DocSync/skill verification entrypoints are no longer default routes.

## Source

The primary inventories are:

```powershell
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
```

Budget categories:

- `hot-path`: default development or Smoke route that should use Go.
- `go-owned`: Full/Release, lifecycle, export, rollback, fresh clone, DocSync, and skill verification paths now owned by Go.
- `compatibility`: retained PowerShell for tag planning, overlay checks, PowerShell-specific quality, external skill workflows, safety, or Plan Mode helpers.
- `documentation-only`: command examples or migration notes in docs.

## Go-Replaced Paths

| Former PowerShell surface | Go replacement | Scope |
|---|---|---|
| Default Smoke kit validation | `bin\aicoding.exe kit verify --all --profile Smoke --json` | Smoke manifest and command envelope checks |
| Full validation route | `bin\aicoding.exe full --json` | Go aggregate gate |
| Release route | `bin\aicoding.exe release gate --json` | Go release aggregate, export, and fresh-clone gate |
| Lifecycle plan | `bin\aicoding.exe lifecycle plan --action install|update|uninstall --all --json` | Registry/manifest lifecycle planning |
| Lifecycle apply | `bin\aicoding.exe lifecycle install|update|uninstall --all --json` | Go lifecycle state and platform install handling |
| Rollback | `bin\aicoding.exe lifecycle rollback --last --json` | Last lifecycle state snapshot restore |
| Export | `bin\aicoding.exe export --all --zip --json` | Native ZIP, manifest, and SHA-256 sidecars |
| Fresh clone | `bin\aicoding.exe fresh-clone --profile Smoke|Full|Release --json` | Temp clone, submodule init, Go build, profile gate |
| Skill verification | `bin\aicoding.exe skill verify --all --profile Smoke|Full|Release --json` | Manifest-declared skill structure and frontmatter checks |
| DocSync staged/all/ci/release | `bin\aicoding.exe docsync staged|all|ci|release --json` | Go DocSync policy and release path checks |
| Removed governance lint script | `bin\aicoding.exe governance lint --json` | Local governance fast lint |
| Removed hook verification script | `bin\aicoding.exe verify hooks --json` | Default hook presence and fast-first verification |
| Removed repo-text script | `bin\aicoding.exe verify repo-text --json` | README, CHANGELOG, and docs text checks |
| Removed release-notes script | `bin\aicoding.exe verify release-notes --json` | CHANGELOG and release policy checks |
| Removed status script | `bin\aicoding.exe status --all --json` | Repo, tool, registry, manifest, required-path summary |

## Keep PowerShell

| PowerShell surface | Reason |
|---|---|
| `scripts/aicoding-tag-governance.ps1 -Action Plan` | Non-destructive legacy tag correction planning |
| `scripts/verify-release-governance-overlay.ps1` | Overlay-specific compatibility check |
| `scripts/aicoding-skill.ps1` external install/audit actions | Third-party skill workflows are broader than skill verification |
| `scripts/lib/AiCoding.SkillAudit.psm1` | External skill audit helper |
| PowerShell AST, PSScriptAnalyzer, and regex optimization gates | PowerShell-specific quality gates |
| Plan Mode and agent hook helper scripts | Agent workflow helpers not covered by lifecycle/export/DocSync migration |
| DSS/XDS/hardware-related scripts, fixtures, or references | Hardware safety boundary; do not run or migrate by default |

The PowerShell Skill Kit pass gate remains scoped to `tools/`, `hooks/`, and `tests/cases/good`.
Negative fixtures remain fixtures and must not be promoted to recursive CI blockers.

## Default Entry Decision

Default local verification should use Go:

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe workflow smart-verify --json
bin\aicoding.exe docsync ci --json
bin\aicoding.exe skill verify --all --profile Smoke --json
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe lifecycle install --all --json
bin\aicoding.exe export --all --zip --json
bin\aicoding.exe fresh-clone --profile Smoke --json
bin\aicoding.exe full --json
bin\aicoding.exe release gate --json
```

PowerShell remains explicit compatibility and specialty tooling only.
Do not delete remaining PowerShell safety, hardware, Plan Mode, external skill, tag planning, or PowerShell-quality scripts without a separate migration plan and validation.
