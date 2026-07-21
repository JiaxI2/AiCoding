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

AiCoding 是本地 AI coding 工作流的平台集成、安装、治理和 CodingKit 资产仓库。它负责 kit 注册表、hook、验证入口、发布治理和 Go CLI 控制面，不拥有嵌入式 skill 源码。

[中文](README_CN.md) | [English](README_EN.md)

## 项目边界

- 平台仓库：集成 CodingKit 资产、kit registry、本地 hook、Taskfile 路由、发布治理和 Go CLI 门禁。
- 源码边界：权威 skill/plugin 源码位于 `CodingKit/agents/skills` 子模块和对应生成资产。
- 运行边界：插件 runtime 状态通过安装、更新和验证流程管理，不直接改 Codex cache。
- 发布边界：平台版本、kit/component 版本和 milestone tag 使用独立命名空间。

## 当前架构

Go CLI 是唯一正式产品控制面。产品工作流固定为 `bootstrap` → `lifecycle` →
`doctor --all` / `verify --profile` → `test --profile` → `release verify|gate`。
领域级 hook、governance、DocSync、Skill、MCP、export 和 fresh-clone 命令仍作为该控制面下的
子命令或专项诊断存在，不再形成第二套产品入口。

Taskfile 只做短路由，业务逻辑在 Go 的 `internal/*` 包中。PowerShell/Python 只保留专项质量、安全、计划模式（Plan Mode）、外部 skill、tag planning / overlay compatibility 和硬件/工具链专项流程。

## Git Governance Standard

AiCoding 使用仓库内置 Git Governance Standard。

- commit type taxonomy：`feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`。
- branch naming and environment mapping：`main`, `develop`, `feature`, `test`, `release`, `hotfix`。
- Issue managed lifecycle：使用结构化表单、`type/area/priority/status/resolution` label 轴和人工确认的关闭证据；不按 stale 时间自动关闭。
- Release typed notes：发布说明按主类型汇总，并由 `.github/RELEASE_TEMPLATE.md` 和 `bin/aicoding.exe verify release-notes --json` 验证。

## 快速开始

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe lifecycle plan --action install --scope all --runtime-profile runtime --json
bin\aicoding.exe doctor --all --json
bin\aicoding.exe verify --profile Smoke --json
bin\aicoding.exe test --profile Smoke --json
```

## 常用入口

| 场景 | 命令 | 说明 |
|---|---|---|
| 初始化 | `go run ./cmd/aicoding bootstrap --json` | 构建 `bin/aicoding.exe` |
| 生命周期计划 | `bin\aicoding.exe lifecycle plan --action install --scope kit --all --json` | `--scope` 始终显式；跨域使用 `--scope all` |
| 产品诊断 | `task doctor` | 路由到 `doctor --all` |
| 产品验证 | `task verify` | 路由到 `verify --profile Smoke` |
| Smoke / Full / Release | `task smoke` / `task full` / `task release` | 路由到唯一 `test --profile` 引擎 |
| 最近测试报告 | `bin\aicoding.exe test latest` | 查看最近一次官方测试摘要 |

## 架构图

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

## 文档索引

| 主题 | 文档 |
|---|---|
| 架构总览 | [docs/ARCHITECTURE_OVERVIEW.md](docs/ARCHITECTURE_OVERVIEW.md) |
| 命令矩阵 | [docs/COMMANDS.md](docs/COMMANDS.md) |
| C99 / C UserStyle Kit | [docs/guides/C99_STANDARD_C_SKILL.md](docs/guides/C99_STANDARD_C_SKILL.md) |
| 官方测试 | [docs/operations/testing/GLOBAL_TEST_PLAN.md](docs/operations/testing/GLOBAL_TEST_PLAN.md) |
| PowerShell 当前边界 | [docs/architecture/POWERSHELL_BOUNDARY.md](docs/architecture/POWERSHELL_BOUNDARY.md) |
| Issue governance | [docs/governance/ISSUE_GOVERNANCE.md](docs/governance/ISSUE_GOVERNANCE.md) |
| Release governance overlay | [docs/governance/RELEASE_GOVERNANCE_OVERLAY.md](docs/governance/RELEASE_GOVERNANCE_OVERLAY.md) |
| Tag policy | [docs/governance/TAGGING_POLICY.md](docs/governance/TAGGING_POLICY.md) |
| Release policy | [docs/governance/RELEASE_POLICY.md](docs/governance/RELEASE_POLICY.md) |

## Tag 规则摘要

- 平台发布 tag：`vMAJOR.MINOR.PATCH`。
- Kit/component 发布 tag：`kit/<kit-id>/vMAJOR.MINOR.PATCH`。
- Milestone tag：`milestone/YYYY.MM.DD-<name>`。
- 不移动、不覆盖、不复用已经绑定 release 的 immutable tag。
