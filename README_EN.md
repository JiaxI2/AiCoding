# AiCoding

[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=release)](https://github.com/JiaxI2/AiCoding/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/go1.22)
[![Go toolchain](https://img.shields.io/badge/Go%20toolchain-1.26.5-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/devel/release#go1.26.5)
[![Staticcheck](https://img.shields.io/badge/Staticcheck-2026.1-5C2D91)](https://github.com/dominikh/go-tools/releases/tag/2026.1)
[![govulncheck](https://img.shields.io/badge/govulncheck-1.6.0-00ADD8?logo=go&logoColor=white)](https://go.googlesource.com/vuln/+/refs/tags/v1.6.0)
[![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://github.com/PowerShell/PowerShell/releases/tag/v7.0.0)
[![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://docs.python.org/3.10/whatsnew/3.10.html)
[![Taskfile](https://img.shields.io/badge/Taskfile-optional-29BEB0?logo=task&logoColor=white)](https://taskfile.dev/)
[![clang-format](https://img.shields.io/badge/clang--format-17.0.2-5C2D91?logo=llvm&logoColor=white)](https://github.com/llvm/llvm-project/releases/tag/llvmorg-17.0.2)
[![C UserStyle Kit](https://img.shields.io/badge/C%20UserStyle%20Kit-1.2.0-00599C?logo=c&logoColor=white)](docs/guides/C99_STANDARD_C_SKILL.md)
[![License](https://img.shields.io/github/license/JiaxI2/AiCoding)](LICENSE)

AiCoding is the platform integration, installation, governance, and CodingKit asset repository for the local AI coding workflow. It owns kit registration, hooks, verification entrypoints, release governance, and the Go CLI control plane. It does not own embedded skill source code.

[中文](README_CN.md) | [English](README_EN.md)

## Project Boundary

- Platform repository: integrates CodingKit assets, kit registry, local hooks, Taskfile routing, release governance, and Go CLI gates.
- Source boundary: authoritative skill/plugin source lives in the `CodingKit/agents/skills` submodule and generated package assets.
- Runtime boundary: installed plugin/runtime state is managed through install, update, and verify workflows, not direct Codex cache edits.
- Release boundary: platform, kit/component, and milestone tags use separate namespaces.

## Current Architecture

The Go CLI is the single formal product control plane. The product workflow is
`bootstrap` → `lifecycle` → `doctor --all` / `verify --profile` →
`test --profile` → `release verify|gate`. Domain hooks, governance, DocSync,
Skill, MCP, export, and fresh-clone commands remain subcommands or specialty
diagnostics rather than parallel product entrypoints.

Taskfile is routing only. Business logic lives in Go packages under `internal/*`. PowerShell/Python remains for specialty quality, safety, Plan Mode helpers, external skill workflows, tag planning / overlay compatibility, and hardware or toolchain-specific flows.

## Git Governance Standard

AiCoding uses the repository Git Governance Standard.

- Commit type taxonomy: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.
- Branch naming and environment mapping: `main`, `develop`, `feature`, `test`, `release`, `hotfix`.
- Issue managed lifecycle: structured forms, `type/area/priority/status/resolution` label axes, and human-reviewed closure evidence; stale age never auto-closes an Issue.
- Release typed notes: release notes are grouped by primary type and validated through `.github/RELEASE_TEMPLATE.md` and `bin/aicoding.exe verify release-notes --json`.

## Quick Start

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe lifecycle plan --action install --scope all --runtime-profile runtime --json
bin\aicoding.exe doctor --all --json
bin\aicoding.exe verify --profile Smoke --json
bin\aicoding.exe test --profile Smoke --json
```

## Common Entrypoints

| Scenario | Command | Notes |
|---|---|---|
| Bootstrap | `go run ./cmd/aicoding bootstrap --json` | Builds `bin/aicoding.exe` |
| Lifecycle plan | `bin\aicoding.exe lifecycle plan --action install --scope kit --all --json` | `--scope` is always explicit; cross-domain work uses `--scope all` |
| Product doctor | `task doctor` | Routes to `doctor --all` |
| Product verify | `task verify` | Routes to `verify --profile Smoke` |
| Smoke / Full / Release | `task smoke` / `task full` / `task release` | Routes to the single `test --profile` engine |
| Latest test report | `bin\aicoding.exe test latest` | Shows the latest official test summary |

## Architecture Diagram

```text
User / Agent
  -> Go CLI
     -> lifecycle -> Kit / MCP / runtime Skill
     -> doctor / verify -> shared report schema
     -> test profiles -> one test engine / content evidence
     -> hooks -> governed commit / push gates
     -> release -> verify / gate
  -> Taskfile / CI -> short routes to Go CLI
  -> specialty tools -> quality / safety / Plan Mode / toolchain
```

## Documentation Index

| Topic | Document |
|---|---|
| Architecture overview | [docs/ARCHITECTURE_OVERVIEW.md](docs/ARCHITECTURE_OVERVIEW.md) |
| Command matrix | [docs/COMMANDS.md](docs/COMMANDS.md) |
| C99 / C UserStyle Kit | [docs/guides/C99_STANDARD_C_SKILL.md](docs/guides/C99_STANDARD_C_SKILL.md) |
| Official testing | [docs/operations/testing/GLOBAL_TEST_PLAN.md](docs/operations/testing/GLOBAL_TEST_PLAN.md) |
| PowerShell boundary | [docs/architecture/POWERSHELL_BOUNDARY.md](docs/architecture/POWERSHELL_BOUNDARY.md) |
| Issue governance | [docs/governance/ISSUE_GOVERNANCE.md](docs/governance/ISSUE_GOVERNANCE.md) |
| Release governance overlay | [docs/governance/RELEASE_GOVERNANCE_OVERLAY.md](docs/governance/RELEASE_GOVERNANCE_OVERLAY.md) |
| Tag policy | [docs/governance/TAGGING_POLICY.md](docs/governance/TAGGING_POLICY.md) |
| Release policy | [docs/governance/RELEASE_POLICY.md](docs/governance/RELEASE_POLICY.md) |

## Tag Rules Summary

- Platform release tags: `vMAJOR.MINOR.PATCH`.
- Kit/component release tags: `kit/<kit-id>/vMAJOR.MINOR.PATCH`.
- Milestone tags: `milestone/YYYY.MM.DD-<name>`.
- Do not move, overwrite, or reuse immutable release-bound tags.
