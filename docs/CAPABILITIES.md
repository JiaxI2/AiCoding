# AiCoding 平台能力索引

> 本文件由 `config/internal-capabilities.json` 生成，请运行 `bin/aicoding.exe capability index --write` 更新。

Registry digest: `sha256:1f915ac583c9dc8d7661c87741942fc68d96c3439dbc797a734f0e8fe7523fdd`

共登记 29 个 `internal/` 一级包；文档义务按公共入口、内部实现域和 Primitive 分级。

- `publicEntries` 非空：必须指向 typed command catalog 中的现存入口，并登记架构文档。
- `stable` 且 `publicEntries` 非空：必须登记 quickstart 与 activation，避免只有命令没有用法。
- `internal-only`：没有公共入口时可不单建架构文档，避免文档剧场。
- `stable`：必须登记至少一条可执行验证命令；`beta`/`experimental` 仍需明确状态。

| ID | Package | Type | Status | Summary | Public entries | Architecture | Verification |
|---|---|---|---|---|---|---|---|
| `adr-review` | `internal/adrreview` | `internal-only` | `stable` | 检查新 Primitive ADR 是否包含必需的自评清单。 | — | — | `go test ./internal/adrreview/...` |
| `bootstrap` | `internal/bootstrap` | `product-workflow` | `stable` | 检查并构建 AiCoding Go CLI 的最小本地启动路径。 | `aicoding bootstrap` | [文档](architecture/AICODING_CORE_ARCHITECTURE.md) | `go test ./internal/bootstrap/...` |
| `c-style` | `internal/cstyle` | `domain-capability` | `stable` | 统一 C99 风格、注释、格式化与宿主验证入口。 | `aicoding skill c99-standard-c check`<br>`aicoding skill c99-standard-c verify` | [文档](architecture/C_USERSTYLE_KIT_ARCHITECTURE.md) | `go test ./internal/cstyle/...` |
| `cache` | `internal/cache` | `domain-capability` | `stable` | 观测并按证据保护规则回收已注册的本地生成物与临时资源。 | `aicoding cache status`<br>`aicoding cache clean` | [文档](architecture/01-system-architecture.md) | `go test ./internal/cache/...` |
| `capability` | `internal/capability` | `domain-capability` | `beta` | 把 internal 包投影为可查询、可生成且可治理的单一能力目录。 | `aicoding capability list`<br>`aicoding capability describe`<br>`aicoding capability index` | [文档](architecture/01-system-architecture.md) | `go test ./internal/capability/...` |
| `cli` | `internal/cli` | `product-workflow` | `stable` | 拥有 typed command catalog、参数解析、帮助、JSON stdout 与退出码。 | `aicoding --help` | [文档](architecture/AICODING_CORE_ARCHITECTURE.md) | `go test ./internal/cli/...` |
| `docsync` | `internal/docsync` | `domain-capability` | `stable` | 检测源码、配置与权威文档之间的同步漂移。 | `aicoding docsync all` | [文档](architecture/DOC_SYNC_PLUS_SPEC.md) | `go test ./internal/docsync/...` |
| `git` | `internal/gitx` | `primitive` | `stable` | 拥有全仓唯一 Git 子进程边界与内容状态读取。 | — | — | `go test ./internal/gitx/...` |
| `governance` | `internal/governance` | `domain-capability` | `stable` | 执行提交、依赖方向、目录布局与能力孤儿门禁。 | `aicoding governance lint`<br>`aicoding governance dependencies`<br>`aicoding governance layout`<br>`aicoding governance capabilities` | [文档](architecture/GRAPH_FIRST.md) | `go test ./internal/governance/...` |
| `kit` | `internal/kit` | `domain-capability` | `stable` | 加载、投影、验证并脚手架化 Kit 能力。 | `aicoding kit list`<br>`aicoding kit describe`<br>`aicoding kit init`<br>`aicoding kit register`<br>`aicoding kit prefetch`<br>`aicoding kit verify` | [文档](architecture/KIT_LIFECYCLE_ARCHITECTURE.md) | `go test ./internal/kit/...` |
| `lifecycle` | `internal/lifecycle` | `product-workflow` | `stable` | 以统一 adapter catalog 编排 Kit、MCP、runtime Skill 与 repo-context 生命周期。 | `aicoding lifecycle plan`<br>`aicoding lifecycle status`<br>`aicoding lifecycle verify` | [文档](architecture/KIT_LIFECYCLE_ARCHITECTURE.md) | `go test ./internal/lifecycle/...` |
| `loop-engineering` | `internal/loopkit` | `domain-capability` | `stable` | 校验有界 WorkSpec、裁决下一步并追加记录尝试，不执行循环。 | `aicoding work validate`<br>`aicoding work next`<br>`aicoding work status`<br>`aicoding work record` | [文档](architecture/LOOP_ENGINEERING_ARCHITECTURE.md) | `go test ./internal/loopkit/...` |
| `mcp-control` | `internal/mcpcontrol` | `domain-capability` | `stable` | 读取 MCP 注册表并执行状态、诊断、验证与生命周期动作。 | `aicoding mcp list`<br>`aicoding mcp status`<br>`aicoding mcp doctor`<br>`aicoding mcp verify` | [文档](architecture/MCP_CONTROL_PLANE.md) | `go test ./internal/mcpcontrol/...` |
| `path-policy` | `internal/pathpolicy` | `primitive` | `stable` | 统一编译、校验并匹配冻结的仓库相对路径 pattern 方言。 | — | — | `go test ./internal/pathpolicy/...` |
| `plan-mode` | `internal/plan` | `domain-capability` | `stable` | 校验计划产物、批准绑定与 Git Tree 漂移。 | `aicoding plan check`<br>`aicoding plan verify`<br>`aicoding plan status`<br>`aicoding plan approve` | [文档](architecture/PLAN_MODE_ARCHITECTURE.md) | `go test ./internal/plan/...` |
| `platform` | `internal/platform` | `primitive` | `stable` | 提供仓库路径、文件事实与临时资源生命周期原语。 | — | — | `go test ./internal/platform/...` |
| `powershell-regex` | `internal/pwshregex` | `domain-capability` | `stable` | 对 PowerShell 正则高风险写法执行 Go-native 快速检查。 | `aicoding powershell regex-lint` | [文档](architecture/POWERSHELL_BOUNDARY.md) | `go test ./internal/pwshregex/...` |
| `registry` | `internal/registry` | `primitive` | `stable` | 提供确定性 snapshot、digest 与防可变泄漏的注册表原语。 | — | — | `go test ./internal/registry/...` |
| `release-gate` | `internal/releasegate` | `product-workflow` | `stable` | 执行发布结构验证并组合正式 Release 门禁。 | `aicoding release verify`<br>`aicoding release gate` | [文档](governance/RELEASE_POLICY.md) | `go test ./internal/releasegate/...` |
| `repo-context` | `internal/repocontext` | `domain-capability` | `stable` | 构建并同步仓库事实快照，供生命周期只读使用。 | `aicoding lifecycle status` | [文档](architecture/02-context-architecture.md) | `go test ./internal/repocontext/...` |
| `repo-health` | `internal/repohealth` | `product-workflow` | `stable` | 聚合产品 doctor 与确定性 verify 检查。 | `aicoding doctor --all`<br>`aicoding verify` | [文档](architecture/01-system-architecture.md) | `go test ./internal/repohealth/...` |
| `repo-init` | `internal/repoinit` | `product-workflow` | `stable` | 幂等初始化 Git 本地设置、Hook、状态根与文档骨架。 | `aicoding provision` | [文档](decisions/0005-repo-init.md) | `go test ./internal/repoinit/...` |
| `report` | `internal/report` | `primitive` | `stable` | 拥有全仓唯一 Result、StandardReport 与共享证据信封。 | — | — | `go test ./internal/report/...` |
| `reuse-governance` | `internal/reuse` | `domain-capability` | `stable` | 验证可复用模块边界与既有复用证据。 | `aicoding governance reuse` | [文档](architecture/GIT_REUSE_BOUNDARY.md) | `go test ./internal/reuse/...` |
| `runner` | `internal/runner` | `primitive` | `stable` | 按确定性顺序执行有界并发任务，不拥有测试语义。 | — | — | `go test ./internal/runner/...` |
| `tag-policy` | `internal/tagpolicy` | `domain-capability` | `stable` | 只读审计 Git tag 命名空间与发布标签策略。 | `aicoding tag audit` | [文档](governance/TAGGING_POLICY.md) | `go test ./internal/tagpolicy/...` |
| `test-engine` | `internal/testengine` | `product-workflow` | `stable` | 拥有 Smoke、Full、Release 测试注册、执行、超时与报告。 | `aicoding test` | [文档](architecture/AICODING_CORE_ARCHITECTURE.md) | `go test ./internal/testengine/...` |
| `todolist` | `internal/todolist` | `domain-capability` | `stable` | 只读投影 docs/todolist 的状态、标题与验证入口。 | `aicoding todolist` | [文档](decisions/0004-todolist-primitive.md) | `go test ./internal/todolist/...` |
| `validation-evidence` | `internal/validationevidence` | `domain-capability` | `stable` | 把测试结论绑定到 Git 内容身份，同一内容可审计复用。 | `aicoding validation status`<br>`aicoding validation check`<br>`aicoding validation explain`<br>`aicoding validation list` | [文档](decisions/0007-validation-evidence.md) | `go test ./internal/validationevidence/...` |

## 公共能力使用闭环

<a id="capability-bootstrap"></a>

### `bootstrap` Bootstrap

- 当前状态：`stable`（`product-workflow`）
- 是什么：检查并构建 AiCoding Go CLI 的最小本地启动路径。
- 架构图：[文档](architecture/AICODING_CORE_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding bootstrap --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding bootstrap --json`。
- 怎么验证：`go test ./internal/bootstrap/...`
- 一次查看：`bin/aicoding.exe capability describe --id bootstrap --json`

<a id="capability-c-style"></a>

### `c-style` C99 Style Control

- 当前状态：`stable`（`domain-capability`）
- 是什么：统一 C99 风格、注释、格式化与宿主验证入口。
- 架构图：[文档](architecture/C_USERSTYLE_KIT_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding skill c99-standard-c status --json`
  2. `aicoding skill c99-standard-c verify --profile fast --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding skill c99-standard-c status --json`。
- 怎么验证：`go test ./internal/cstyle/...`
- 一次查看：`bin/aicoding.exe capability describe --id c-style --json`

<a id="capability-cache"></a>

### `cache` Local Artifact Retention

- 当前状态：`stable`（`domain-capability`）
- 是什么：观测并按证据保护规则回收已注册的本地生成物与临时资源。
- 架构图：[文档](architecture/01-system-architecture.md)
- 怎么用：
  1. `aicoding cache status --json`
  2. `aicoding cache clean --scope temp --dry-run --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding cache status --json`。
- 怎么验证：`go test ./internal/cache/...`
- 一次查看：`bin/aicoding.exe capability describe --id cache --json`

<a id="capability-capability"></a>

### `capability` Capability Discoverability

- 当前状态：`beta`（`domain-capability`）
- 是什么：把 internal 包投影为可查询、可生成且可治理的单一能力目录。
- 架构图：[文档](architecture/01-system-architecture.md)
- 怎么用：
  1. `aicoding capability list --json`
  2. `aicoding capability describe --id capability --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding capability list --json`。
- 怎么验证：`go test ./internal/capability/...`
- 一次查看：`bin/aicoding.exe capability describe --id capability --json`

<a id="capability-cli"></a>

### `cli` Typed CLI Control Plane

- 当前状态：`stable`（`product-workflow`）
- 是什么：拥有 typed command catalog、参数解析、帮助、JSON stdout 与退出码。
- 架构图：[文档](architecture/AICODING_CORE_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding --help`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding --help`。
- 怎么验证：`go test ./internal/cli/...`
- 一次查看：`bin/aicoding.exe capability describe --id cli --json`

<a id="capability-docsync"></a>

### `docsync` DocSync

- 当前状态：`stable`（`domain-capability`）
- 是什么：检测源码、配置与权威文档之间的同步漂移。
- 架构图：[文档](architecture/DOC_SYNC_PLUS_SPEC.md)
- 怎么用：
  1. `aicoding docsync all --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding docsync all --json`。
- 怎么验证：`go test ./internal/docsync/...`
- 一次查看：`bin/aicoding.exe capability describe --id docsync --json`

<a id="capability-governance"></a>

### `governance` Repository Governance

- 当前状态：`stable`（`domain-capability`）
- 是什么：执行提交、依赖方向、目录布局与能力孤儿门禁。
- 架构图：[文档](architecture/GRAPH_FIRST.md)
- 怎么用：
  1. `aicoding governance capabilities --json`
  2. `aicoding governance dependencies --json`
  3. `aicoding governance layout --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding governance capabilities --json`。
- 怎么验证：`go test ./internal/governance/...`
- 一次查看：`bin/aicoding.exe capability describe --id governance --json`

<a id="capability-kit"></a>

### `kit` Kit Management

- 当前状态：`stable`（`domain-capability`）
- 是什么：加载、投影、验证并脚手架化 Kit 能力。
- 架构图：[文档](architecture/KIT_LIFECYCLE_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding kit list --json`
  2. `aicoding kit describe --all --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding kit list --json`。
- 怎么验证：`go test ./internal/kit/...`
- 一次查看：`bin/aicoding.exe capability describe --id kit --json`

<a id="capability-lifecycle"></a>

### `lifecycle` Lifecycle Composition

- 当前状态：`stable`（`product-workflow`）
- 是什么：以统一 adapter catalog 编排 Kit、MCP、runtime Skill 与 repo-context 生命周期。
- 架构图：[文档](architecture/KIT_LIFECYCLE_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding lifecycle status --scope all --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding lifecycle status --scope all --json`。
- 怎么验证：`go test ./internal/lifecycle/...`
- 一次查看：`bin/aicoding.exe capability describe --id lifecycle --json`

<a id="capability-loop-engineering"></a>

### `loop-engineering` Loop Engineering

- 当前状态：`stable`（`domain-capability`）
- 是什么：校验有界 WorkSpec、裁决下一步并追加记录尝试，不执行循环。
- 架构图：[文档](architecture/LOOP_ENGINEERING_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding work validate --file testdata/loopkit/examples/project-development.work.json --json`
  2. `aicoding work next --file testdata/loopkit/examples/project-development.work.json --json`
  3. `aicoding work status --file testdata/loopkit/examples/project-development.work.json --json`
  - 示例输入：`testdata/loopkit/examples/project-development.work.json`
- 怎么进 Agent：`cli-entry`；work 系列命令已在 typed catalog，Agent 直接调用即可，无需 install；loop-engineering-kit 的可选打包状态不影响这些命令。；调用 `aicoding work next --file testdata/loopkit/examples/project-development.work.json --json`。
- 怎么验证：`go test ./internal/loopkit/...`
- 一次查看：`bin/aicoding.exe capability describe --id loop-engineering --json`

<a id="capability-mcp-control"></a>

### `mcp-control` MCP Control Plane

- 当前状态：`stable`（`domain-capability`）
- 是什么：读取 MCP 注册表并执行状态、诊断、验证与生命周期动作。
- 架构图：[文档](architecture/MCP_CONTROL_PLANE.md)
- 怎么用：
  1. `aicoding mcp list --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding mcp list --json`。
- 怎么验证：`go test ./internal/mcpcontrol/...`
- 一次查看：`bin/aicoding.exe capability describe --id mcp-control --json`

<a id="capability-plan-mode"></a>

### `plan-mode` Plan Mode

- 当前状态：`stable`（`domain-capability`）
- 是什么：校验计划产物、批准绑定与 Git Tree 漂移。
- 架构图：[文档](architecture/PLAN_MODE_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding plan check --paths README.md --json`
  2. `aicoding plan status --all --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding plan check --paths README.md --json`。
- 怎么验证：`go test ./internal/plan/...`
- 一次查看：`bin/aicoding.exe capability describe --id plan-mode --json`

<a id="capability-powershell-regex"></a>

### `powershell-regex` PowerShell Regex Lint

- 当前状态：`stable`（`domain-capability`）
- 是什么：对 PowerShell 正则高风险写法执行 Go-native 快速检查。
- 架构图：[文档](architecture/POWERSHELL_BOUNDARY.md)
- 怎么用：
  1. `aicoding powershell regex-lint --staged --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding powershell regex-lint --staged --json`。
- 怎么验证：`go test ./internal/pwshregex/...`
- 一次查看：`bin/aicoding.exe capability describe --id powershell-regex --json`

<a id="capability-release-gate"></a>

### `release-gate` Release Gate

- 当前状态：`stable`（`product-workflow`）
- 是什么：执行发布结构验证并组合正式 Release 门禁。
- 架构图：[文档](governance/RELEASE_POLICY.md)
- 怎么用：
  1. `aicoding release verify --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding release verify --json`。
- 怎么验证：`go test ./internal/releasegate/...`
- 一次查看：`bin/aicoding.exe capability describe --id release-gate --json`

<a id="capability-repo-context"></a>

### `repo-context` Repository Context

- 当前状态：`stable`（`domain-capability`）
- 是什么：构建并同步仓库事实快照，供生命周期只读使用。
- 架构图：[文档](architecture/02-context-architecture.md)
- 怎么用：
  1. `aicoding lifecycle status --scope repo-context --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding lifecycle status --scope repo-context --json`。
- 怎么验证：`go test ./internal/repocontext/...`
- 一次查看：`bin/aicoding.exe capability describe --id repo-context --json`

<a id="capability-repo-health"></a>

### `repo-health` Repository Health

- 当前状态：`stable`（`product-workflow`）
- 是什么：聚合产品 doctor 与确定性 verify 检查。
- 架构图：[文档](architecture/01-system-architecture.md)
- 怎么用：
  1. `aicoding doctor --all --json`
  2. `aicoding verify --profile Smoke --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding doctor --all --json`。
- 怎么验证：`go test ./internal/repohealth/...`
- 一次查看：`bin/aicoding.exe capability describe --id repo-health --json`

<a id="capability-repo-init"></a>

### `repo-init` Repository Provisioning

- 当前状态：`stable`（`product-workflow`）
- 是什么：幂等初始化 Git 本地设置、Hook、状态根与文档骨架。
- 架构图：[文档](decisions/0005-repo-init.md)
- 怎么用：
  1. `aicoding provision --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding provision --json`。
- 怎么验证：`go test ./internal/repoinit/...`
- 一次查看：`bin/aicoding.exe capability describe --id repo-init --json`

<a id="capability-reuse-governance"></a>

### `reuse-governance` Reuse Governance

- 当前状态：`stable`（`domain-capability`）
- 是什么：验证可复用模块边界与既有复用证据。
- 架构图：[文档](architecture/GIT_REUSE_BOUNDARY.md)
- 怎么用：
  1. `aicoding governance reuse --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding governance reuse --json`。
- 怎么验证：`go test ./internal/reuse/...`
- 一次查看：`bin/aicoding.exe capability describe --id reuse-governance --json`

<a id="capability-tag-policy"></a>

### `tag-policy` Tag Policy

- 当前状态：`stable`（`domain-capability`）
- 是什么：只读审计 Git tag 命名空间与发布标签策略。
- 架构图：[文档](governance/TAGGING_POLICY.md)
- 怎么用：
  1. `aicoding tag audit --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding tag audit --json`。
- 怎么验证：`go test ./internal/tagpolicy/...`
- 一次查看：`bin/aicoding.exe capability describe --id tag-policy --json`

<a id="capability-test-engine"></a>

### `test-engine` Global Test Engine

- 当前状态：`stable`（`product-workflow`）
- 是什么：拥有 Smoke、Full、Release 测试注册、执行、超时与报告。
- 架构图：[文档](architecture/AICODING_CORE_ARCHITECTURE.md)
- 怎么用：
  1. `aicoding test --profile Smoke --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding test --profile Smoke --json`。
- 怎么验证：`go test ./internal/testengine/...`
- 一次查看：`bin/aicoding.exe capability describe --id test-engine --json`

<a id="capability-todolist"></a>

### `todolist` Todolist Projection

- 当前状态：`stable`（`domain-capability`）
- 是什么：只读投影 docs/todolist 的状态、标题与验证入口。
- 架构图：[文档](decisions/0004-todolist-primitive.md)
- 怎么用：
  1. `aicoding todolist --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding todolist --json`。
- 怎么验证：`go test ./internal/todolist/...`
- 一次查看：`bin/aicoding.exe capability describe --id todolist --json`

<a id="capability-validation-evidence"></a>

### `validation-evidence` Validation Evidence

- 当前状态：`stable`（`domain-capability`）
- 是什么：把测试结论绑定到 Git 内容身份，同一内容可审计复用。
- 架构图：[文档](decisions/0007-validation-evidence.md)
- 怎么用：
  1. `aicoding validation status --json`
  2. `aicoding validation list --json`
- 怎么进 Agent：`cli-entry`；命令已在 typed catalog，Agent 直接调用即可，无需单独 install。；调用 `aicoding validation status --json`。
- 怎么验证：`go test ./internal/validationevidence/...`
- 一次查看：`bin/aicoding.exe capability describe --id validation-evidence --json`
