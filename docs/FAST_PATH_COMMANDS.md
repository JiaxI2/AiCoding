# Fast Path Commands

Fast Path 是 Go CLI 默认控制面。当前 main 的重复开发检查、CI Smoke、Full、Release gate、DocSync、skill verify、lifecycle、export 和 fresh-clone 都由 `bin/aicoding.exe` 承担。

## Bootstrap

```powershell
go run ./cmd/aicoding bootstrap --json
bin\aicoding.exe bootstrap --json
```

`bootstrap` 解析仓库根目录、检查基础工具并构建 `bin/aicoding.exe`。

## Smoke And CI

```powershell
bin\aicoding.exe smoke --json
bin\aicoding.exe ci --profile Smoke --json
```

`smoke` 是本地快速聚合。`ci` 在 Smoke 聚合基础上包含 `go test ./...`，用于 `.github/workflows/aicoding-ci.yml`。

聚合检查由 `internal/runner` 的并发 Plan 执行。新增或移除检查点时，只需要在对应 Plan 注册或移除任务 ID，不需要重写 worker 调度。

## Full And Release

```powershell
bin\aicoding.exe full --json
bin\aicoding.exe release gate --json
```

Full 和 Release 是 Go-native aggregate gates。Release gate 额外覆盖 export 和 fresh-clone Release 路径。

## Lifecycle, Export, Fresh Clone

```powershell
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe lifecycle install --all --json
bin\aicoding.exe lifecycle rollback --last --json
bin\aicoding.exe export --all --zip --json
bin\aicoding.exe fresh-clone --profile Smoke --json
```

Lifecycle plan、skill verify、kit smoke 和 export manifest hash 阶段使用 Go 并发计划；实际写状态和写 ZIP 的路径保持串行，避免副作用交叉。

## Doctor

```powershell
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe doctor perf --json
```

`doctor pwsh-budget` 用来确认 PowerShell 只留在专项边界内。

## PowerShell Boundary

PowerShell 保留为显式专项工具：tag planning、release overlay compatibility、PowerShell 质量、安全、Plan Mode、外部 skill 和硬件/工具链诊断。默认 Smoke/CI/Full/Release 不通过 PowerShell 编排。