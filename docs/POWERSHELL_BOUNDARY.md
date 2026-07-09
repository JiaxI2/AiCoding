# PowerShell Boundary

本文档只描述当前 main 的 PowerShell 保留边界。默认控制面是 Go CLI；Taskfile 只做路由；PowerShell 不承载 Smoke、CI、Full、Release gate、DocSync、skill verify、lifecycle、export 或 fresh-clone 的默认编排。

## 默认入口

```powershell
bin\aicoding.exe smoke --json
bin\aicoding.exe ci --profile Smoke --json
bin\aicoding.exe full --json
bin\aicoding.exe release gate --json
```

Go CLI 同时拥有 hook、governance、repohealth、DocSync、skill verify、lifecycle、export、fresh-clone 和 C99 C/H style gate。

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

## 禁止事项

- 不新增 PowerShell 默认门禁。
- 不通过 Taskfile 承载业务逻辑。
- 不把 Go 默认入口重新包装成 PowerShell。
- 不删除仍属于专项安全、Plan Mode、外部 skill、tag planning、overlay compatibility 或 PowerShell 质量的脚本，除非有单独计划和验证。