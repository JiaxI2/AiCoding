# AiCoding

[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=release)](https://github.com/JiaxI2/AiCoding/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://learn.microsoft.com/powershell/)
[![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://www.python.org/)
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

The Go CLI is the default control plane. It owns bootstrap, Smoke, CI, official test profiles, hooks, status, repo text, release notes, tag/release structural checks, governance lint, DocSync, skill verify, lifecycle, export, and fresh-clone.

Taskfile is routing only. Business logic lives in Go packages under `internal/*`. PowerShell/Python remains for specialty quality, safety, Plan Mode helpers, external skill workflows, tag planning / overlay compatibility, and hardware or toolchain-specific flows.

## Git Governance Standard

AiCoding uses the repository Git Governance Standard.

- Commit type taxonomy: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.
- Branch naming and environment mapping: `main`, `develop`, `feature`, `test`, `release`, `hotfix`.
- Release typed notes: release notes are grouped by primary type and validated through `.github/RELEASE_TEMPLATE.md` and `bin/aicoding.exe verify release-notes --json`.

## Quick Start

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe smoke --json
bin\aicoding.exe ci --profile Smoke --json
task smoke
bin\aicoding.exe test full --json
bin\aicoding.exe test release --json
```

## Common Entrypoints

| Scenario | Command | Notes |
|---|---|---|
| Bootstrap | `go run ./cmd/aicoding bootstrap --json` | Builds `bin/aicoding.exe` |
| Local Smoke | `task smoke` | Routes to `bin/aicoding.exe smoke --json` |
| CI Smoke | `bin\aicoding.exe ci --profile Smoke --json` | Go tests and default aggregate gates |
| Full | `task full` | Routes to the official Full test profile |
| Release | `task release` | Routes to the official Release test profile |
| Latest test report | `bin\aicoding.exe test latest` | Shows the latest official test summary |

## Architecture Diagram

```text
User / Agent
  -> Taskfile routing
     -> Go CLI (bin/aicoding.exe)
        -> runner plans -> smoke / ci
        -> test profiles -> full / release / latest
        -> kit registry -> CodingKit assets + skill submodule
     -> specialty scripts -> quality / safety / Plan Mode / toolchain
```

## Documentation Index

| Topic | Document |
|---|---|
| Architecture overview | [docs/ARCHITECTURE_OVERVIEW.md](docs/ARCHITECTURE_OVERVIEW.md) |
| Command matrix | [docs/COMMANDS.md](docs/COMMANDS.md) |
| C99 / C UserStyle Kit | [docs/guides/C99_STANDARD_C_SKILL.md](docs/guides/C99_STANDARD_C_SKILL.md) |
| Official testing | [docs/operations/testing/GLOBAL_TEST_PLAN.md](docs/operations/testing/GLOBAL_TEST_PLAN.md) |
| PowerShell boundary | [docs/architecture/POWERSHELL_BOUNDARY.md](docs/architecture/POWERSHELL_BOUNDARY.md) |
| Release governance overlay | [docs/governance/RELEASE_GOVERNANCE_OVERLAY.md](docs/governance/RELEASE_GOVERNANCE_OVERLAY.md) |
| Tag policy | [docs/governance/TAGGING_POLICY.md](docs/governance/TAGGING_POLICY.md) |
| Release policy | [docs/governance/RELEASE_POLICY.md](docs/governance/RELEASE_POLICY.md) |

## Tag Rules Summary

- Platform release tags: `vMAJOR.MINOR.PATCH`.
- Kit/component release tags: `kit/<kit-id>/vMAJOR.MINOR.PATCH`.
- Milestone tags: `milestone/YYYY.MM.DD-<name>`.
- Do not move, overwrite, or reuse immutable release-bound tags.
