# PowerShell Regex Optimization Skill Kit

## 摘要

本 Kit 是 `codex-agent-powershell-skill-kit` 的 member skill，同时接入 Go Fast Path。目标是让 Agent 在生成 PowerShell 正则替换代码时默认避开三类问题：

1. `$1`、`${Name}` 捕获组 replacement 被双引号提前展开。
2. `Get-Content | ForEach-Object { -replace } | Set-Content` 逐行替换造成吞吐瓶颈。
3. 动态回调替换未显式约束 PowerShell 7+。

## 快路径

```powershell
bin\aicoding.exe powershell regex-lint --staged --json
bin\aicoding.exe powershell regex-lint --path scripts --json
```

`hook pre-commit` 应自动调用 staged regex lint。

## 慢路径

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\skills\codex-agent-powershell-skill-kit\tools\Invoke-PowerShellRegexOptimizationGate.ps1 -Path .\scripts -Recurse -Format Json
```

## 推荐原子函数

```powershell
Invoke-SafeRegexReplace -InputText $text -Pattern '([a-z]+)(\d+)' -ReplaceToken '$1-$2'
Update-FileContentBulk -FilePath .\file.ps1 -Pattern 'source' -ReplaceToken 'target'
Update-CodeDynamically -SourceCode $source -Pattern '(?:^|_)(\w)' -Callback { param($m) $m.Groups[1].Value.ToUpperInvariant() }
```

## 验收

```powershell
go test ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin\aicoding.exe powershell regex-lint --path dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\skills --json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\test-codex-agent-powershell-skill-kit.ps1 -InstallMissingTools -Json
```
