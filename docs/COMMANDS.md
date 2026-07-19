# Commands

Taskfile 是人机短路由；Go CLI 是默认控制面；业务逻辑位于 Go 的 `internal/*` 包中。PowerShell/Python 只作为专项工具保留，不承载默认 Smoke、CI、Full 或 Release profile。

C/H 风格命令见 [C99 Standard C Skill](guides/C99_STANDARD_C_SKILL.md)。

## 默认入口

| 场景 | 命令 | 控制面 |
|---|---|---|
| Bootstrap | `go run ./cmd/aicoding bootstrap --json` | Go |
| 生命周期 | `bin\aicoding.exe lifecycle ... --json` | Go |
| 产品诊断 | `bin\aicoding.exe doctor --all --json` | Go |
| 产品验证 | `bin\aicoding.exe verify --profile Smoke\|Full\|Release --json` | Go |
| 产品测试 | `bin\aicoding.exe test --profile Smoke\|Full\|Release --json` | Go |
| 发布闭环 | `bin\aicoding.exe release verify\|gate --json` | Go |
| 最近测试报告 | `bin\aicoding.exe test latest` | Go |

## 正式产品命令

| 目的 | 命令 |
|---|---|
| Bootstrap | `bin\aicoding.exe bootstrap --json` |
| Kit 生命周期 plan/apply | `bin\aicoding.exe lifecycle plan --action install\|update\|uninstall --scope kit --all --json` / `lifecycle install\|update\|uninstall --scope kit --all --json` |
| 全域生命周期 plan | `bin\aicoding.exe lifecycle plan --action install\|update --scope all --runtime-profile runtime\|full\|skill-development --json` |
| 全域生命周期状态/诊断 | `bin\aicoding.exe lifecycle status\|doctor --scope all --json` |
| 产品诊断 | `bin\aicoding.exe doctor --all --timeout-sec 180 --json` |
| 产品验证 | `bin\aicoding.exe verify --profile Smoke\|Full\|Release --timeout-sec 180 --json` |
| 官方测试 | `bin\aicoding.exe test --profile Smoke\|Full\|Release --json` |
| 最近测试报告 | `bin\aicoding.exe test latest` |
| Release 结构验证 | `bin\aicoding.exe release verify --json` |
| Release 测试门禁 | `bin\aicoding.exe release gate --json` |

`doctor` 只做环境和状态诊断；`verify` 只组合确定性静态/结构验证；`test` 独占
Smoke/Full/Release 测试 Registry、timeout、runner、report 和 exit code；`release`
不创建 Tag 或 Release，只执行结构验证或复用 Release test profile。
测试 profile 对 rollback 只执行 `lifecycle rollback --scope kit --help` 的只读契约检查，不会应用
本地 rollback snapshot。

## 命令契约固化

顶层 command ID、名称/alias、是否要求 subcommand、handler 和 `aicoding --help` form
由 `internal/cli` 的 typed command catalog 统一描述。新增或删除顶层命令必须先更新该
catalog，并通过 catalog 完整性与 CLI contract 测试；不能再在 router、help 和
namespace 判断中分别维护字符串列表。

到期的兼容命令已从 catalog、router 和 help 删除；旧写法返回 usage error（退出码 2），
不会再静默转发。`lifecycle` 现在要求显式 `--scope kit|mcp|runtime-skill|repo-context|all`。
`--scope repo-context` 作用于整个仓库，不接受 `--kit`、`--component` 或 `--all`；它扫描仓库
事实（目录、语言/工具链、依赖边）生成受管的小粒度上下文文件到 `.aicoding/repo-context/`，
并用 facts digest 与生成物 digest 对账新鲜度。详见
[ADR 0003](decisions/0003-repo-context-domain.md) 与
[07 演进路线](architecture/07-roadmap.md) §3。
`kit list --json` 与 `mcp list --json` 的外层报告包含
`inputDigest`；MCP inventory 同时保留 `registryDigest` 并增加 `catalogDigest`。前者只标识
规范化 registry，后者标识 registry 与全部 referenced manifests 的内容树。
正式 `lifecycle ... --json` 在 `data` 中返回静态 adapter `catalogDigest`、本次
`planDigest`，并在每个 adapter result 中返回 `inputDigest`。Agent/Skill 应使用这些字段
追踪“对什么事实执行了什么意图”，不解析人类文本或直接调用 specialty 脚本。
`aicoding version` 从构建注入值或 `config/codex-kit.json` manifest 元数据读取版本，
不再把实现代际标签硬编码到 Go 文件。
Fast Path 的稳定 cache identity 为 `.aicoding/cache/fast-path`；旧的 versioned cache
是可删除的临时数据，不再由当前 `cache status|clean` 管理。

## 领域与专项命令

| 目的 | 命令 |
|---|---|
| DocSync staged/all/ci/release | `bin\aicoding.exe docsync staged|all|ci|release --json` |
| Skill verify | `bin\aicoding.exe skill verify --all --profile Smoke|Full|Release --json` |
| MCP inventory | `bin\aicoding.exe mcp list --json` |
| MCP status/doctor | `bin\aicoding.exe mcp status|doctor <COMPONENT> --json` |
| MCP verify | `bin\aicoding.exe mcp verify <COMPONENT>\|--all --profile Smoke\|Full\|Release --configured --json` |
| C99 skill status | `bin\aicoding.exe skill c99-standard-c status --json` |
| C99 skill templates | `bin\aicoding.exe skill c99-standard-c templates --json` |
| C99 skill 快速/完整验证 | `bin\aicoding.exe skill c99-standard-c verify --profile fast\|full --timings --json` |
| C99 skill fmt/check | `bin\aicoding.exe skill c99-standard-c fmt|check --scope changed|staged|paths --json` |
| Rollback | `bin\aicoding.exe lifecycle rollback --scope kit --last --json` |
| Repo-context 生成/保鲜 | `bin\aicoding.exe lifecycle install\|update\|uninstall\|status\|doctor\|verify --scope repo-context --json` |
| Export | `bin\aicoding.exe export --all --zip --json` |
| Fresh clone | `bin\aicoding.exe fresh-clone --profile Smoke|Full|Release --json` |
| Governance lint | `bin\aicoding.exe governance lint --json` |
| Dependency direction / stable identity | `bin\aicoding.exe governance dependencies --json` |
| Repository layout gate | `bin\aicoding.exe governance layout --json` |
| Reuse governance evidence | `bin\aicoding.exe governance reuse --json` |
| Hook verification | `bin\aicoding.exe verify hooks --json` |
| Repo text verification | `bin\aicoding.exe verify repo-text --json` |
| Release notes verification | `bin\aicoding.exe verify release-notes --json` |
| PowerShell inventory | `bin\aicoding.exe doctor pwsh --json` |
| PowerShell budget | `bin\aicoding.exe doctor pwsh-budget --json` |
| Tag audit | `bin\aicoding.exe tag audit --json` |
| 解析 Codex Token JSONL | `bin\aicoding.exe codex usage parse --file <FILE> --json` |
| 运行 Codex 并采集 Token | `bin\aicoding.exe codex usage run -- codex exec --json "<PROMPT>"` |

## 已移除的兼容入口

以下入口不再路由；调用会返回 usage error。迁移时使用右侧正式入口：

| 兼容入口 | 正式入口 |
|---|---|
| `smoke` | `test --profile Smoke` |
| `ci --profile <PROFILE>` | `test --profile <PROFILE>` |
| `full` | `test --profile Full` |
| `test full\|release` | `test --profile Full\|Release` |
| `kit lifecycle ...` | `lifecycle ... --scope kit` |
| `mcp install\|update\|uninstall ...` | `lifecycle install\|update\|uninstall --scope mcp ...` |
| `status --all` | `doctor --all` |

## Codex Skill 运行时同步

插件更新先比较 released package 与 installed cache 的 `BUILDINFO.json`；发生漂移时通过 Codex 官方 plugin CLI 重装同一 Marketplace plugin，刷新成功后才写 install state：

```powershell
bin\aicoding.exe lifecycle plan --action update --scope kit --kit aicoding-platform --json
bin\aicoding.exe lifecycle update --scope kit --kit aicoding-platform --json
```

Standalone full profile 通过正式 lifecycle adapter 统一到官方 user root，并在显式迁移时
备份同名 unmanaged path：

```powershell
bin\aicoding.exe lifecycle plan --action update --scope runtime-skill --runtime-profile full --source-repository F:\Study\AI\Codex-Skills --json
bin\aicoding.exe lifecycle update --scope runtime-skill --runtime-profile full --source-repository F:\Study\AI\Codex-Skills --migrate-unmanaged --json
bin\aicoding.exe verify --profile Smoke --runtime-profile full --source-repository F:\Study\AI\Codex-Skills --json
```

底层 PowerShell profile/audit 脚本保留为 lifecycle adapter 的显式 specialty 实现，不再作为
常规文档主入口。

## MCP 组件控制面

MCP registry、component manifest、Codex 配置、生命周期和兼容性回归统一由 Go CLI 管理：

```powershell
bin\aicoding.exe mcp list --json
bin\aicoding.exe mcp status visio-mcp --json
bin\aicoding.exe mcp doctor visio-mcp --json
bin\aicoding.exe mcp verify visio-mcp --profile Smoke --json
bin\aicoding.exe mcp verify --all --profile Smoke --json
```

Agent/Skill 执行写操作时使用正式 lifecycle plan/apply：

```powershell
bin\aicoding.exe lifecycle plan --scope mcp --action update --component visio-mcp --json
bin\aicoding.exe lifecycle update --scope mcp --component visio-mcp --json
```

`--configured` 显式包含 Codex 当前配置的 stdio/Streamable HTTP MCP 只读 initialize/discovery。
MCP 生命周期正式入口使用 `lifecycle --scope mcp`；旧 `mcp install|update|uninstall`
已移除。

Smoke 与 Full 使用 mock/benchmark 路径，不要求 Microsoft Visio；Release 会显式运行可见 Visio COM smoke 和 VSDX/PNG/SVG/PDF 导出。MCP capability 不注册工作流 prompt，画图步骤、有限 repair 和最终视觉确认由上层 `visio-diagram` Skill 负责。

详细边界见 [MCP Control Plane](architecture/MCP_CONTROL_PLANE.md)，操作说明见 [MCP Components](operations/MCP_COMPONENTS.md)。

## Codex Token 报告

`codex usage parse` 支持 Codex App Server 的 `thread/tokenUsage/updated` 通知和
`codex exec --json` 的 `turn.completed.usage` 事件；`--file -` 表示从 stdin 读取。
App Server 报告使用 `tokenUsage.total` 作为会话累计值，并使用
`tokenUsage.last.totalTokens` 计算当前上下文占用，避免把累计会话用量误当作上下文大小。

结构化结果继续使用统一 `report.Result` 外壳，标准报告的
`data.details.token_usage` 保存归一化 Token 数据。`codex usage run` 将子进程 JSONL
事件流保留到 stderr，并在 stdout 输出最终 AiCoding 报告。

## Taskfile 路由

| Task | 命令 |
|---|---|
| `task setup` | `go run ./cmd/aicoding bootstrap --json` |
| `task doctor` | `bin/aicoding.exe doctor --all --json` |
| `task verify` | `bin/aicoding.exe verify --profile Smoke --json` |
| `task smoke` | `bin/aicoding.exe test --profile Smoke --json` |
| `task ci` | `bin/aicoding.exe test --profile Smoke --json` |
| `task full` | `bin/aicoding.exe test --profile Full --json` |
| `task release` | `bin/aicoding.exe test --profile Release --json` |
| `task test:latest` | `bin/aicoding.exe test latest` |
| `task style:c:status` | `bin/aicoding.exe skill c99-standard-c status --json` |
| `task style:c:templates` | `bin/aicoding.exe skill c99-standard-c templates --json` |
| `task style:c:verify` | `bin/aicoding.exe skill c99-standard-c verify --profile fast --timings --json` |
| `task fmt:c` | `bin/aicoding.exe skill c99-standard-c fmt --scope changed --json` |
| `task fmt-check:c` | `bin/aicoding.exe skill c99-standard-c check --scope changed --json` |
| `task fmt-check-staged:c` | `bin/aicoding.exe skill c99-standard-c check --scope staged --json` |

外部候选可追加 `--target path/to/verify-target.json`；项目差异配置可用可重复的
`--overlay path/to/project-overlay.json`。完整 target/schema 说明见
[C99 Standard C Skill](guides/C99_STANDARD_C_SKILL.md)。

## CI

当前默认 CI workflow 是 `.github/workflows/aicoding-ci.yml`：

```text
go build -o bin/aicoding.exe ./cmd/aicoding
.\bin\aicoding.exe test --profile Smoke --json
```

手动或定时 release job 运行：

```text
bin\aicoding.exe test --profile Release --json
```

CI 不额外调用 `doctor` 或 `verify` 聚合器；唯一 test Registry 已直接登记相应 leaf gate，
避免同一检查在一个 job 中重复执行。

## 运行模型

默认控制面统一由 Go CLI 承担。Smoke、Full 和 Release 只通过 `test --profile` 进入唯一
test engine，全局测试报告落在临时的 `test-results/`；Doctor、Verify、DocSync、
Lifecycle、Skill verify 和 fresh-clone 均由上表中的 Go CLI 路由执行。PowerShell 仅保留
专项慢路径，边界见 [PowerShell Boundary](architecture/POWERSHELL_BOUNDARY.md)。

## PowerShell 专项入口

当前源码仅保留专项脚本类别：tag planning / overlay compatibility、PowerShell 质量、安全、Plan Mode、外部 skill 和硬件/工具链专项流程。默认验证入口不调用这些脚本。

## JSON 和退出码

所有 Go CLI 默认入口输出 `report.Result` envelope。退出码：`0` 表示 `ok=true`，`1`
表示验证或执行失败，`2` 表示参数错误；`errorKind` 使用 `usage`、`execution` 或
`validation`。共享 Schema 见
[CLI、验证与测试报告 Schema](operations/testing/REPORT_SCHEMA.md)。
JSON 报告契约的冻结面、兼容扩展与解冻规则见
[契约冻结与获取/激活边界](architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md)。
