# AiCoding

[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=release)](https://github.com/JiaxI2/AiCoding/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://learn.microsoft.com/powershell/)
[![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://www.python.org/)
[![Taskfile](https://img.shields.io/badge/Taskfile-optional-29BEB0?logo=task&logoColor=white)](https://taskfile.dev/)
[![clang-format](https://img.shields.io/badge/clang--format-17.0.2-blue)](https://github.com/llvm/llvm-project/releases/tag/llvmorg-17.0.2)
[![License](https://img.shields.io/github/license/JiaxI2/AiCoding)](LICENSE)

AiCoding is the platform integration, installation, governance, and CodingKit asset repository for the local AI coding workflow. It owns kit registration, hooks, verification entrypoints, release governance, and the Go CLI control plane. It does not own embedded skill source code.

[中文](README_CN.md) | [English](README_EN.md)

## Project Positioning / 项目定位

- Platform repository: integrates CodingKit assets, kit registry, local hooks, Taskfile routing, release governance, and Go CLI checks.
- Source boundary: authoritative skill/plugin source lives in the `CodingKit/agents/skills` submodule and generated package assets.
- Runtime boundary: installed plugin/runtime state is managed through install, update, and verify workflows, not direct Codex cache edits.
- Release boundary: platform, kit/component, and milestone tags use separate namespaces.

## Current Architecture / 当前架构

AiCoding uses the Go CLI as the default local control plane:

- Go CLI: owns bootstrap, smart-verify, Smoke, hooks, status, repo text, release notes, tag/release structural checks, governance lint, DocSync, skill verify, lifecycle, export, fresh-clone, Full, and Release gate.
- PowerShell/Python: retained for compatibility, specialty quality gates, safety checks, Plan Mode helpers, external skill workflows, tag planning, release overlay compatibility, and hardware/toolchain-specific flows.

The Go lane reduces repeated PowerShell cold starts, emits stable JSON, and now owns the Full/Release aggregate gates.

## Quick Start / 快速开始

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe workflow smart-verify --json
task smoke
bin\aicoding.exe docsync ci --json
bin\aicoding.exe skill verify --all --profile Smoke --json
bin\aicoding.exe release gate --json
```

Complete local validation and formal release gates route through `task full` and `task release`, both backed by the Go CLI.

## Architecture Diagram / 架构图

```mermaid
flowchart TD
  User["User / Agent"] --> Taskfile["Taskfile<br/>routing"]
  Taskfile --> GoCLI["Go CLI<br/>bin/aicoding.exe"]
  GoCLI --> FastChecks["bootstrap / smart-verify<br/>smoke / hooks / status<br/>verify / lint / doctor"]
  GoCLI --> Full["full / release<br/>lifecycle / export / rollback<br/>fresh clone / docsync / skills"]
  GoCLI --> Registry["Kit registry<br/>config/kit-registry.json<br/>config/kits/*.json"]
  Registry --> CodingKit["CodingKit assets<br/>skill submodule"]
```

## Documentation Index / 文档索引

| Need | Document |
|---|---|
| Architecture overview | [docs/ARCHITECTURE_OVERVIEW.md](docs/ARCHITECTURE_OVERVIEW.md) |
| Fast Path commands | [docs/FAST_PATH_COMMANDS.md](docs/FAST_PATH_COMMANDS.md) |
| Full command matrix | [docs/COMMANDS.md](docs/COMMANDS.md) |
| C Style Format Kit | [docs/C_STYLE_FORMAT_KIT.md](docs/C_STYLE_FORMAT_KIT.md) |
| PowerShell migration map | [docs/POWERSHELL_MIGRATION.md](docs/POWERSHELL_MIGRATION.md) |
| Release governance overlay | [docs/RELEASE_GOVERNANCE_OVERLAY.md](docs/RELEASE_GOVERNANCE_OVERLAY.md) |
| Tag policy | [docs/TAGGING_POLICY.md](docs/TAGGING_POLICY.md) |
| Release policy | [docs/RELEASE_POLICY.md](docs/RELEASE_POLICY.md) |

## Git Governance Standard / Git 治理标准

Commit type taxonomy: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.

Branch naming and environment mapping: `main` is the platform baseline; `develop`, `feature/*`, `test/*`, `release/*`, and `hotfix/*` describe integration, feature, test, release, and hotfix work.

Release notes must be typed by primary change type, and platform Tag/Release notes default to Chinese-first bilingual text.

## Release / Tag Short Rules / Release / Tag 简短规则

- Platform release tags: `vMAJOR.MINOR.PATCH`, for example `v0.2.0`.
- Kit/component release tags: `kit/<kit-id>/vMAJOR.MINOR.PATCH`.
- Milestone tags: `milestone/YYYY.MM.DD-<name>`.
- Do not publish component versions as pseudo platform tags such as `v1.3.0-powershell-skill-kit`.
- Do not move, overwrite, or reuse immutable release-bound tags.
