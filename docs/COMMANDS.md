# Commands

Taskfile 是人机短路由；Go CLI 是默认控制面；业务逻辑位于 Go 的 `internal/*` 包中。PowerShell/Python 只作为专项工具保留，不承载默认 Smoke、CI、Full 或 Release profile。

C/H 风格命令见 [C99 Standard C Skill](guides/C99_STANDARD_C_SKILL.md)。

## 默认入口

| 场景 | 命令 | 控制面 |
|---|---|---|
| Bootstrap | `go run ./cmd/aicoding bootstrap --json` | Go |
| 本地 Smoke | `task smoke` | Go |
| CI Smoke | `bin\aicoding.exe ci --profile Smoke --json` | Go |
| Full profile | `task full` | Go |
| Release profile | `task release` | Go |
| 最近测试报告 | `bin\aicoding.exe test latest` | Go |

## Go CLI

| 目的 | 命令 |
|---|---|
| Bootstrap | `bin\aicoding.exe bootstrap --json` |
| Smoke 聚合 | `bin\aicoding.exe smoke --json` |
| CI 聚合 | `bin\aicoding.exe ci --profile Smoke --json` |
| 官方 Full 测试 | `bin\aicoding.exe test full --json` |
| 官方 Release 测试 | `bin\aicoding.exe test release --json` |
| 最近测试报告 | `bin\aicoding.exe test latest` |
| DocSync staged/all/ci/release | `bin\aicoding.exe docsync staged|all|ci|release --json` |
| Skill verify | `bin\aicoding.exe skill verify --all --profile Smoke|Full|Release --json` |
| MCP inventory | `bin\aicoding.exe mcp list --json` |
| MCP status/doctor | `bin\aicoding.exe mcp status|doctor <COMPONENT> --json` |
| MCP verify | `bin\aicoding.exe mcp verify <COMPONENT>\|--all --profile Smoke\|Full\|Release --configured --json` |
| MCP lifecycle | `bin\aicoding.exe mcp install|update|uninstall <COMPONENT>\|--all --dry-run --json` |
| C99 skill status | `bin\aicoding.exe skill c99-standard-c status --json` |
| C99 skill templates | `bin\aicoding.exe skill c99-standard-c templates --json` |
| C99 skill 快速/完整验证 | `bin\aicoding.exe skill c99-standard-c verify --profile fast\|full --timings --json` |
| C99 skill fmt/check | `bin\aicoding.exe skill c99-standard-c fmt|check --scope changed|staged|paths --json` |
| Lifecycle plan | `bin\aicoding.exe lifecycle plan --action install|update|uninstall --all --json` |
| Lifecycle apply | `bin\aicoding.exe lifecycle install|update|uninstall --all --json` |
| Rollback | `bin\aicoding.exe lifecycle rollback --last --json` |
| Export | `bin\aicoding.exe export --all --zip --json` |
| Fresh clone | `bin\aicoding.exe fresh-clone --profile Smoke|Full|Release --json` |
| Full aggregate | `bin\aicoding.exe full --json` |
| Release gate | `bin\aicoding.exe release gate --json` |
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

## Codex Skill 运行时同步

插件更新先比较 released package 与 installed cache 的 `BUILDINFO.json`；发生漂移时通过 Codex 官方 plugin CLI 重装同一 Marketplace plugin，刷新成功后才写 install state：

```powershell
bin\aicoding.exe lifecycle plan --action update --kit aicoding-platform --json
bin\aicoding.exe lifecycle update --kit aicoding-platform --json
```

Standalone full profile 默认统一到官方 user root，并在显式迁移时备份同名 unmanaged path：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File tools/specialty/set-codex-skill-profile.ps1 -Profile full -StandaloneRoot agents -DryRun -Json
powershell -NoProfile -ExecutionPolicy Bypass -File tools/specialty/set-codex-skill-profile.ps1 -Profile full -StandaloneRoot agents -MigrateUnmanaged -Json
powershell -NoProfile -ExecutionPolicy Bypass -File tools/specialty/audit-runtime-skills.ps1 -ExpectedProfile full -StandaloneRoot agents -Strict -Json
```

## MCP 组件控制面

MCP registry、component manifest、Codex 配置、生命周期和兼容性回归统一由 Go CLI 管理：

```powershell
bin\aicoding.exe mcp list --json
bin\aicoding.exe mcp status visio-mcp --json
bin\aicoding.exe mcp doctor visio-mcp --json
bin\aicoding.exe mcp verify visio-mcp --profile Smoke --json
bin\aicoding.exe mcp verify --all --profile Smoke --json
```

`--configured` 显式包含 Codex 当前配置的 stdio/Streamable HTTP MCP 只读 initialize/discovery；`--all` 会自动包含该兼容性 probe。生命周期操作先使用 `--dry-run`，再执行 install、update 或 uninstall。

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
| `task smoke` | `bin/aicoding.exe smoke --json` |
| `task ci` | `bin/aicoding.exe ci --profile Smoke --json` |
| `task full` | `bin/aicoding.exe test full --json` |
| `task release` | `bin/aicoding.exe test release --json` |
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
.\bin\aicoding.exe ci --profile Smoke --json
```

手动或定时 release job 运行：

```text
bin\aicoding.exe test release --json
```

## 运行模型

默认控制面统一由 Go CLI 承担：`Full` 和 `Release` 分别通过 `test full` 与 `test release` 运行，全局测试报告落在临时的 `test-results/`；DocSync、Lifecycle、Skill verify 和 fresh-clone 均由上表中的 Go CLI 路由执行。PowerShell 仅保留专项慢路径，边界见 [PowerShell Boundary](architecture/POWERSHELL_BOUNDARY.md)。

## PowerShell 专项入口

当前源码仅保留专项脚本类别：tag planning / overlay compatibility、PowerShell 质量、安全、Plan Mode、外部 skill 和硬件/工具链专项流程。默认验证入口不调用这些脚本。

## JSON 和退出码

所有 Go CLI 默认入口输出 `report.Result` envelope。退出码：`0` 表示 `ok=true`，`1` 表示验证或执行失败，`2` 表示参数错误。
