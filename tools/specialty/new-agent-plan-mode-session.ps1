[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [Parameter(Mandatory=$true)][string]$Feature,
  [string]$Description = "",
  [string[]]$Scope = @("**"),
  [switch]$NeedsDecision,
  [switch]$DryRun,
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data }
  if ($Json) { $obj | ConvertTo-Json -Depth 30 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 20 }
  if (-not $ok) { exit 1 }
}

function Safe-Slug([string]$Text) {
  $slug = ($Text.ToLowerInvariant() -replace '[^a-z0-9]+','-').Trim('-')
  if ([string]::IsNullOrWhiteSpace($slug)) { return "plan-mode-session" }
  if ($slug.Length -gt 64) { return $slug.Substring(0,64).Trim('-') }
  return $slug
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $id = Safe-Slug $Feature
  $planDir = Join-Path $RepoRoot "docs/spec/$id"
  $planFile = Join-Path $planDir "PLAN.md"
  if (Test-Path -LiteralPath $planFile) {
    Out-Result $false "PLAN_EXISTS" "同名 Plan Mode 会话已存在，拒绝覆盖。" @{ id=$id; path=$planFile }
  }
  $status = if ($NeedsDecision) { "needs-decision" } else { "draft" }
  $decision = if ($NeedsDecision) { "docs/spec/$id/DECISION.md" } else { "" }
  $scopeLines = @($Scope | ForEach-Object { "  - " + ($_ | ConvertTo-Json -Compress) }) -join "`n"

  $planText = @"
---
id: $id
status: $status
scope:
$scopeLines
approvedTree: ""
decision: $decision
gates:
  - profile: full
---

# 计划模式会话（Plan Mode Session）：$Feature

## 需求

$Description

## 约束与范围

- 只在 frontmatter `scope` 范围内实现。
- 批准前 `approvedTree` 保持为空；不得人工填写。
- 完成前运行 frontmatter 声明的验证门禁。

## 实现计划

1. 澄清需求与边界。
2. 如有多方案，记录选项并等待用户决策。
3. 将已选方案拆成可验证任务。
4. 实现后运行门禁并记录交接结果。

## 回滚

实现前补充准确的回滚命令或文件恢复路径。
"@

  $tasksText = @"
# 任务（Tasks）：$Feature

- [ ] 完成需求与范围确认。
- [ ] 如需要，记录用户决策。
- [ ] 实施最小变更。
- [ ] 运行声明的验证门禁。
- [ ] 总结验证与回滚。
"@

  $optionsText = @"
# 选项（Options）：$Feature

Decision Status: Pending User Selection

## Option A

- 适用性：
- 影响：
- 验证：
- 回滚：

## Option B

- 适用性：
- 影响：
- 验证：
- 回滚：
"@

  $planned = @(
    @{ path=$planFile; content=$planText },
    @{ path=(Join-Path $planDir "TASKS.md"); content=$tasksText }
  )
  if ($NeedsDecision) {
    $planned += @{ path=(Join-Path $planDir "OPTIONS.md"); content=$optionsText }
  }
  if (-not $DryRun) {
    New-Item -ItemType Directory -Force -Path $planDir | Out-Null
    foreach ($item in $planned) {
      Set-Content -LiteralPath $item.path -Value $item.content -Encoding UTF8
    }
  }
  Out-Result $true "OK" $(if ($DryRun) { "Plan Mode 会话 dry-run 完成，未写入文件。" } else { "Plan Mode 会话已创建。" }) ([ordered]@{
    id=$id
    status=$status
    directory=$planDir
    files=@($planned | ForEach-Object { $_.path })
    dryRun=[bool]$DryRun
  })
}
catch {
  Out-Result $false "INTERNAL_ERROR" ("创建 Plan Mode 会话时发生内部错误：{0}" -f $_.Exception.Message)
}
