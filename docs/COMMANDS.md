# Commands

Taskfile 是人机短路由；Go CLI 是默认控制面；业务逻辑位于 Go 的 `internal/*` 包中。PowerShell/Python 只作为专项工具保留，不承载默认 Smoke、CI、Full 或 Release profile。

C/H 风格命令见 [C99 Standard C Skill](C99_STANDARD_C_SKILL.md)。

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
| C99 skill status | `bin\aicoding.exe skill c99-standard-c status --json` |
| C99 skill templates | `bin\aicoding.exe skill c99-standard-c templates --json` |
| C99 skill fmt/check | `bin\aicoding.exe skill c99-standard-c fmt|check --scope changed|staged|paths --json` |
| Lifecycle plan | `bin\aicoding.exe lifecycle plan --action install|update|uninstall --all --json` |
| Lifecycle apply | `bin\aicoding.exe lifecycle install|update|uninstall --all --json` |
| Rollback | `bin\aicoding.exe lifecycle rollback --last --json` |
| Export | `bin\aicoding.exe export --all --zip --json` |
| Fresh clone | `bin\aicoding.exe fresh-clone --profile Smoke|Full|Release --json` |
| Full aggregate | `bin\aicoding.exe full --json` |
| Release gate | `bin\aicoding.exe release gate --json` |
| Governance lint | `bin\aicoding.exe governance lint --json` |
| Hook verification | `bin\aicoding.exe verify hooks --json` |
| Repo text verification | `bin\aicoding.exe verify repo-text --json` |
| Release notes verification | `bin\aicoding.exe verify release-notes --json` |
| PowerShell inventory | `bin\aicoding.exe doctor pwsh --json` |
| PowerShell budget | `bin\aicoding.exe doctor pwsh-budget --json` |
| Tag audit | `bin\aicoding.exe tag audit --json` |

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
| `task fmt:c` | `bin/aicoding.exe skill c99-standard-c fmt --scope changed --json` |
| `task fmt-check:c` | `bin/aicoding.exe skill c99-standard-c check --scope changed --json` |
| `task fmt-check-staged:c` | `bin/aicoding.exe skill c99-standard-c check --scope staged --json` |

## CI

当前默认 CI workflow 是 `.github/workflows/aicoding-ci.yml`：

```powershell
go build -o bin/aicoding.exe ./cmd/aicoding
bin\aicoding.exe ci --profile Smoke --json
```

手动或定时 release job 运行：

```powershell
bin\aicoding.exe test release --json
```

## PowerShell 专项入口

当前源码仅保留专项脚本类别：tag planning / overlay compatibility、PowerShell 质量、安全、Plan Mode、外部 skill 和硬件/工具链专项流程。默认验证入口不调用这些脚本。

## JSON 和退出码

所有 Go CLI 默认入口输出 `report.Result` envelope。退出码：`0` 表示 `ok=true`，`1` 表示验证或执行失败，`2` 表示参数错误。
