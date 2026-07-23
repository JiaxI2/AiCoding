<p align="center">
  <img src="docs/assets/aicoding-banner-light.svg#gh-light-mode-only" width="100%" alt="AiCoding — Verifiable AI Engineering">
  <img src="docs/assets/aicoding-banner-dark.svg#gh-dark-mode-only" width="100%" alt="AiCoding — Verifiable AI Engineering">
</p>

<p align="center"><a href="README_CN.md">中文</a> · <a href="README_EN.md">English</a></p>

[![Release](https://img.shields.io/github/v/release/JiaxI2/AiCoding?label=Release&color=181717&logo=github&logoColor=white)](https://github.com/JiaxI2/AiCoding/releases/latest) [![CI](https://img.shields.io/github/actions/workflow/status/JiaxI2/AiCoding/aicoding-ci.yml?branch=main&label=CI&logo=githubactions&logoColor=white)](https://github.com/JiaxI2/AiCoding/actions/workflows/aicoding-ci.yml) [![License](https://img.shields.io/github/license/JiaxI2/AiCoding?label=License&color=D22128&logo=apache&logoColor=white)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/go1.22) [![Go toolchain](https://img.shields.io/badge/Go%20toolchain-1.26.5-00ADD8?logo=go&logoColor=white)](https://go.dev/doc/devel/release#go1.26.5) [![Staticcheck](https://img.shields.io/badge/Staticcheck-2026.1-00ADD8?logo=go&logoColor=white)](https://github.com/dominikh/go-tools/releases/tag/2026.1) [![Govulncheck](https://img.shields.io/badge/Govulncheck-1.6.0-00ADD8?logo=go&logoColor=white)](https://go.googlesource.com/vuln/+/refs/tags/v1.6.0) [![PowerShell](https://img.shields.io/badge/PowerShell-7%2B-5391FE?logo=powershell&logoColor=white)](https://github.com/PowerShell/PowerShell/releases/tag/v7.0.0) [![Python](https://img.shields.io/badge/Python-3.10%2B-3776AB?logo=python&logoColor=white)](https://docs.python.org/3.10/whatsnew/3.10.html) [![Taskfile](https://img.shields.io/badge/Taskfile-optional-29BEB0?logo=task&logoColor=white)](https://taskfile.dev/) [![Clang-format](https://img.shields.io/badge/Clang--format-17.0.2-262D3A?logo=llvm&logoColor=white)](https://github.com/llvm/llvm-project/releases/tag/llvmorg-17.0.2) [![C UserStyle Kit](https://img.shields.io/badge/C%20UserStyle%20Kit-1.2.0-A8B9CC?logo=c&logoColor=black)](docs/guides/C99_STANDARD_C_SKILL.md)

AiCoding 是让 AI 编码工作流可验证、可复用、可审计的本地工程平台：同一 Git 内容完整验证约 150 秒，重复检查约 424 毫秒，每个绿灯都能追溯到它验证的内容。

## 状态 / Status

面向 Windows 与自动化调用，所有正式入口都提供 JSON 结果；[CI](https://github.com/JiaxI2/AiCoding/actions/workflows/aicoding-ci.yml) 持续验证主线，[CHANGELOG](CHANGELOG.md) 与 [Releases](https://github.com/JiaxI2/AiCoding/releases) 记录可交付变化。

## 一张图看懂

<p align="center">
  <img src="docs/assets/aicoding-overview-light.svg#gh-light-mode-only" width="100%" alt="AiCoding 一张图看懂">
  <img src="docs/assets/aicoding-overview-dark.svg#gh-dark-mode-only" width="100%" alt="AiCoding 一张图看懂">
</p>

- **只有一个入口**：所有能力都从 aicoding CLI 进入，没有第二控制面。
- **能力分五层**：上层组合下层，下层永不反向依赖。
- **证据形成闭环**：验证结论绑定 Git 内容身份，同一内容可审计复用。

## 快速开始 / Quick Start

在递归包含子模块的 clean clone 根目录，用 PowerShell 逐行复制这三行：

```powershell
go run ./cmd/aicoding bootstrap --json && .\bin\aicoding.exe provision --json
.\bin\aicoding.exe verify --profile Smoke --json
.\bin\aicoding.exe test --profile Smoke --json
```

第一行会从源码构建未入仓的本地二进制并完成仓库初始化；随后你应看到 `ok: true`，最终测试摘要应为 `conclusion: PASS`、`fail: 0`。任一步失败，先运行 `.\bin\aicoding.exe doctor --all --json`，再到[命令矩阵](docs/COMMANDS.md)按错误类别定位。

## 发展路线

静态方向见[架构路线图](docs/architecture/07-roadmap.md)；活的 roadmap 可直接用 `.\bin\aicoding.exe todolist --json` 查询，机器与人看到的是同一队列。

## 按角色进入

| 我是谁 | 我要什么 | 从这里开始 |
|---|---|---|
| 新用户 | 跑通第一个绿灯并继续探索 | 上面的三行 → [命令矩阵](docs/COMMANDS.md) |
| Agent / 自动化 | 稳定命令与 JSON 结果契约 | [命令矩阵](docs/COMMANDS.md) → [报告 schema](docs/operations/testing/REPORT_SCHEMA.md) |
| 贡献者 | 改代码而不越过架构红线 | [架构必读路径](docs/architecture/README.md) → [贡献指南](CONTRIBUTING.md) |
| Kit / 扩展作者 | 先确认 Skill、Kit、MCP、Hook 的权威归属，再进入创作流程 | [创作指引](docs/guides/AUTHORING.md) |

## 内核与 Kit

冻结的六模块内核给出稳定底座，内容寻址证据让结论绑定 Git 内容，裁决式 loop 只决定下一步而不再造执行器；三者分别由[核心架构](docs/architecture/AICODING_CORE_ARCHITECTURE.md)、[验证证据](docs/decisions/0007-validation-evidence.md)与[循环工程架构](docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md)约束。

下表严格投影 `config/kit-registry.json` 当前全部 enabled Kit；能力句来自各 manifest，详情保持一行一链接。

| Kit | 一句话核心能力 | 详情 |
|---|---|---|
| `aicoding-platform` | AiCoding platform integration, Codex plugin marketplace registration, CodingKit asset discovery, and submodule validation. | [Kit / Plugin 视图](docs/reference/KIT_PLUGIN_VIEW.md) |
| `docsync-plus` | Semantic documentation drift detection kit for AiCoding repositories. | [DocSync Plus](docs/architecture/DOC_SYNC_PLUS_SPEC.md) |
| `reuse-governance` | Declarative governance for independently integrated reusable modules. | [复用治理](docs/operations/THIRD_PARTY_REUSE_GOVERNANCE.md) |
| `common-control-kit` | Reusable C99 control modules under CodingKit/modules/common/controller. | [控制模块](CodingKit/modules/common/controller/foc/README.md) |
| `c-userstyle-kit` | First-party C99 style, comment, lint, host-compile, and behavior verification assets backed by the Huawei DKBA 2826-2011&#46;5 reference. | [C UserStyle Kit](docs/guides/C99_STANDARD_C_SKILL.md) |
| `release-governance-overlay-kit` | Tag/release namespace governance, Taskfile entry, and performance-loop overlay for AiCoding. | [发布治理](docs/governance/RELEASE_GOVERNANCE_OVERLAY.md) |

## 平台能力索引

<!-- BEGIN GENERATED: CAPABILITIES -->

> 此区由 `config/internal-capabilities.json` 生成（`sha256:cdc4c7c38b0f0acb4d33282e0e84baa64a2964441c37d3becf4c168fc8a3f1d5`）。完整的 29 项能力见 [能力索引](docs/CAPABILITIES.md)。

| 可直接使用的能力 | 核心职责 | 快速入口 | 使用闭环 | 架构 |
|---|---|---|---|---|
| `bootstrap` Bootstrap | 检查并构建 AiCoding Go CLI 的最小本地启动路径。 | `aicoding bootstrap` | [describe](docs/CAPABILITIES.md#capability-bootstrap) | [文档](docs/architecture/AICODING_CORE_ARCHITECTURE.md) |
| `c-style` C99 Style Control | 统一 C99 风格、注释、格式化与宿主验证入口。 | `aicoding skill c99-standard-c check` | [describe](docs/CAPABILITIES.md#capability-c-style) | [文档](docs/architecture/C_USERSTYLE_KIT_ARCHITECTURE.md) |
| `cache` Local Artifact Retention | 观测并按证据保护规则回收已注册的本地生成物与临时资源。 | `aicoding cache status` | [describe](docs/CAPABILITIES.md#capability-cache) | [文档](docs/architecture/01-system-architecture.md) |
| `capability` Capability Discoverability | 把 internal 包投影为可查询、可生成且可治理的单一能力目录。 | `aicoding capability list` | [describe](docs/CAPABILITIES.md#capability-capability) | [文档](docs/architecture/01-system-architecture.md) |
| `cli` Typed CLI Control Plane | 拥有 typed command catalog、参数解析、帮助、JSON stdout 与退出码。 | `aicoding --help` | [describe](docs/CAPABILITIES.md#capability-cli) | [文档](docs/architecture/AICODING_CORE_ARCHITECTURE.md) |
| `docsync` DocSync | 检测源码、配置与权威文档之间的同步漂移。 | `aicoding docsync all` | [describe](docs/CAPABILITIES.md#capability-docsync) | [文档](docs/architecture/DOC_SYNC_PLUS_SPEC.md) |
| `governance` Repository Governance | 执行提交、依赖方向、目录布局与能力孤儿门禁。 | `aicoding governance lint` | [describe](docs/CAPABILITIES.md#capability-governance) | [文档](docs/architecture/GRAPH_FIRST.md) |
| `kit` Kit Management | 加载、投影、验证并脚手架化 Kit 能力。 | `aicoding kit list` | [describe](docs/CAPABILITIES.md#capability-kit) | [文档](docs/architecture/KIT_LIFECYCLE_ARCHITECTURE.md) |
| `lifecycle` Lifecycle Composition | 以统一 adapter catalog 编排 Kit、MCP、runtime Skill 与 repo-context 生命周期。 | `aicoding lifecycle plan` | [describe](docs/CAPABILITIES.md#capability-lifecycle) | [文档](docs/architecture/KIT_LIFECYCLE_ARCHITECTURE.md) |
| `loop-engineering` Loop Engineering | 校验有界 WorkSpec、裁决下一步并追加记录尝试，不执行循环。 | `aicoding work validate` | [describe](docs/CAPABILITIES.md#capability-loop-engineering) | [文档](docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md) |
| `mcp-control` MCP Control Plane | 读取 MCP 注册表并执行状态、诊断、验证与生命周期动作。 | `aicoding mcp list` | [describe](docs/CAPABILITIES.md#capability-mcp-control) | [文档](docs/architecture/MCP_CONTROL_PLANE.md) |
| `plan-mode` Plan Mode | 校验计划产物、批准绑定与 Git Tree 漂移。 | `aicoding plan check` | [describe](docs/CAPABILITIES.md#capability-plan-mode) | [文档](docs/architecture/PLAN_MODE_ARCHITECTURE.md) |
| `powershell-regex` PowerShell Regex Lint | 对 PowerShell 正则高风险写法执行 Go-native 快速检查。 | `aicoding powershell regex-lint` | [describe](docs/CAPABILITIES.md#capability-powershell-regex) | [文档](docs/architecture/POWERSHELL_BOUNDARY.md) |
| `release-gate` Release Gate | 执行发布结构验证并组合正式 Release 门禁。 | `aicoding release verify` | [describe](docs/CAPABILITIES.md#capability-release-gate) | [文档](docs/governance/RELEASE_POLICY.md) |
| `repo-context` Repository Context | 构建并同步仓库事实快照，供生命周期只读使用。 | `aicoding lifecycle status` | [describe](docs/CAPABILITIES.md#capability-repo-context) | [文档](docs/architecture/02-context-architecture.md) |
| `repo-health` Repository Health | 聚合产品 doctor 与确定性 verify 检查。 | `aicoding doctor --all` | [describe](docs/CAPABILITIES.md#capability-repo-health) | [文档](docs/architecture/01-system-architecture.md) |
| `repo-init` Repository Provisioning | 幂等初始化 Git 本地设置、Hook、状态根与文档骨架。 | `aicoding provision` | [describe](docs/CAPABILITIES.md#capability-repo-init) | [文档](docs/decisions/0005-repo-init.md) |
| `reuse-governance` Reuse Governance | 验证可复用模块边界与既有复用证据。 | `aicoding governance reuse` | [describe](docs/CAPABILITIES.md#capability-reuse-governance) | [文档](docs/architecture/GIT_REUSE_BOUNDARY.md) |
| `tag-policy` Tag Policy | 只读审计 Git tag 命名空间与发布标签策略。 | `aicoding tag audit` | [describe](docs/CAPABILITIES.md#capability-tag-policy) | [文档](docs/governance/TAGGING_POLICY.md) |
| `test-engine` Global Test Engine | 拥有 Smoke、Full、Release 测试注册、执行、超时与报告。 | `aicoding test` | [describe](docs/CAPABILITIES.md#capability-test-engine) | [文档](docs/architecture/AICODING_CORE_ARCHITECTURE.md) |
| `todolist` Todolist Projection | 只读投影 docs/todolist 的状态、标题与验证入口。 | `aicoding todolist` | [describe](docs/CAPABILITIES.md#capability-todolist) | [文档](docs/decisions/0004-todolist-primitive.md) |
| `validation-evidence` Validation Evidence | 把测试结论绑定到 Git 内容身份，同一内容可审计复用。 | `aicoding validation status` | [describe](docs/CAPABILITIES.md#capability-validation-evidence) | [文档](docs/decisions/0007-validation-evidence.md) |

<!-- END GENERATED: CAPABILITIES -->

## 为什么这个仓库越用越值钱

- **地基复利**：冻结内核只接受向上组合，新能力不推倒旧边界（[愿景与四象限](docs/architecture/00-vision.md)）。
- **证据复利**：同一内容完整验证一次，之后可跨 worktree 在约 424 毫秒内复用（[实测基线](docs/operations/VALIDATION_EVIDENCE_BUDGET.md)）。
- **能力复利**：loop、plan、验证证据和 Kit 生命周期共享既有 Primitive，而不是各建一套事实源（[Primitive Constitution](docs/architecture/PRIMITIVE_CONSTITUTION.md)）。
- **知识复利**：四象限中的每类不确定性都有对应沉淀位置，经验会变成下一次工作的输入（[愿景 §3](docs/architecture/00-vision.md)）。

## 当前架构

`Go CLI` 是唯一正式产品入口：`lifecycle` 管理能力，`doctor --all` / `verify --profile` / `test --profile` 产生分级结论，`release verify|gate` 守住发布。Taskfile 只做短路由，PowerShell/Python 只保留专项边界。完整分层见[架构阅读路径](docs/architecture/README.md)。

## Git 工作流 / Git Workflow

仓库遵循 [Git Governance Standard](docs/governance/RELEASE_POLICY.md)：commit type 为 `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`；分支映射为 `main`, `develop`, `feature`, `test`, `release`, `hotfix`；Release typed notes 按主类型汇总并由[发布说明门禁](.github/RELEASE_TEMPLATE.md)验证。

## 仓库导航 / Repository Navigation

面向人的主题入口在 [docs/README.md](docs/README.md)，下面的目录地图由机器配置生成。

<!-- AICODING:REPOSITORY_MAP:START -->
## Repository map

> Generated from `config/repository-navigation.json`. Edit the configuration, not this block.

| Area | Purpose | Audience | Entry |
|---|---|---|---|
| `CodingKit/` | Authoritative skill/plugin assets and submodule boundary. | maintainer, agent | `CodingKit/agents/skills` |
| `cmd/` | Go executable entry points only. | developer | `cmd/aicoding` |
| `config/` | Machine-readable platform configuration, registries, policies and schemas. | maintainer, agent | `config/README.md` |
| `docs/` | Canonical human documentation. | user, contributor, maintainer | `docs/README.md` |
| `internal/` | Go platform implementation packages. | developer | `internal/README.md` |
| `testdata/` | Fixtures and sample repositories; no executable business logic. | developer | `testdata` |
| `tools/` | Standalone specialty, migration, testing and template tooling. | maintainer, agent | `tools/README.md` |

### Common routes

| Need | Start here | Command |
|---|---|---|
| 初始化或构建 AiCoding | `cmd/aicoding` | `go run ./cmd/aicoding bootstrap --json` |
| 查找平台命令 | `docs/COMMANDS.md` | `bin/aicoding.exe governance layout --json` |
| 修改架构或包边界 | `docs/architecture` | — |
| 修改 Hook、发布或 Git 治理 | `docs/governance` | `aicoding governance layout --json` |
| 维护 kit 注册表或生命周期 | `config` | `bin/aicoding.exe lifecycle status --scope all --json` |
| 运行完整验证 | `docs/operations` | `bin/aicoding.exe test --profile Full --json` |
| 维护专项工具 | `tools` | — |
| 维护 Skill 权威源码 | `CodingKit/agents/skills` | — |
<!-- AICODING:REPOSITORY_MAP:END -->

## Star History

[在 Star History 查看项目趋势](https://www.star-history.com/?repos=JiaxI2%2FAiCoding&type=date&legend=top-left)。
