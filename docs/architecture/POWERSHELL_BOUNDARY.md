# PowerShell Boundary

Status: Accepted and Frozen

本文档只描述当前 main 的 PowerShell 保留边界。默认控制面是 Go CLI；Taskfile 只做路由；PowerShell 不承载 Smoke、CI、Full、Release gate、DocSync、skill verify、lifecycle、export 或 fresh-clone 的默认编排。

## 默认入口

```powershell
bin\aicoding.exe doctor --all --json
bin\aicoding.exe verify --profile Smoke --json
bin\aicoding.exe test --profile Smoke --json
bin\aicoding.exe test --profile Full --json
bin\aicoding.exe test --profile Release --json
```

上述 profile 是彼此独立的正式入口，不表示发布时依次执行三档。Full 保留给开发迭代；
发布只执行 Release，因为当前 73-leaf Registry 的直接对照证明 Release 是 Full 的严格
超集，证据见 [TODO 0041](../todolist/done/0041-release-only-publication.md)。

Go CLI 同时拥有 lifecycle、product doctor/verify、test engine、release、hook、governance、
DocSync、skill verify、export、fresh-clone 和 C99 C/H style gate。runtime Skill profile/audit
脚本只能由显式 lifecycle adapter 或专项人工命令调用。

## 保留类别

| 类别 | 当前用途 |
|---|---|
| tag planning | 非破坏性 tag 审计和对齐计划 |
| release overlay compatibility | release overlay 专项验证 |
| PowerShell quality | AST、PSScriptAnalyzer、regex 和脚本安全检查 |
| Plan Mode helpers | Agent workflow helper，不属于默认验证控制面 |
| external skill workflows | 第三方 skill install/audit/status |
| safety/hardware/toolchain | DSS/XDS/flash 等需要独立安全边界的工具链路径 |

## 诊断

```powershell
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
```

`doctor pwsh-budget` 用于确认 PowerShell 调用仍限制在上述专项类别内。

2026-07-22 的 Phase 2 退役后快照为 21 个 `tools/specialty/**/*.ps1`：19 个顶层专项入口与
2 个嵌套 AEF Hook 薄壳；顶层计数中 `thinShells=1`、`deprecated=1`。可执行职责全部属于
上表六类，或处于 ADR/独立 Retirement Plan 已登记的 release 退役窗口。计数是只读观测，
不是归零 KPI；`doctor pwsh` 还从脚本头部 `# RETIRE-AFTER:` 读取逐候选
`retirementTrigger`，缺失显示 `unspecified`，同样不设门禁。默认 Taskfile/CI profile 仍只调用 Go CLI。详细逐文件裁决见
[TODO 0002](../todolist/done/0002-powershell-specialty-convergence.md)。

PWSH-001 的上述 report-only 契约不变。PWSH-002 另从 `config/pwsh-budget.json` 读取
已提交的顶层路径集合与原始 doctor 证据：当前集合必须与最后基线完全相同，后续基线只能
追加前一集合的严格子集。新增、删一换一、删除后未同步下调，或退休候选重新变为
`unspecified` 均由 `doctor pwsh-budget` 非零阻断并指出路径；不对 deprecated/thinShell
另设数值规则。

## 禁止事项

- 专项命令面停止增长：不新增专项脚本，不新增保留类别；新能力一律进入 Go 控制面。
- 不新增 PowerShell 默认门禁。
- 不通过 Taskfile 承载业务逻辑。
- 不把 Go 默认入口重新包装成 PowerShell。
- 不删除仍属于专项安全、Plan Mode、外部 skill、tag planning、overlay compatibility 或 PowerShell 质量的脚本，除非有单独计划和验证。
