[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [string]$Id = "",
  [Parameter(Mandatory=$true)][string]$Title,
  [Parameter(Mandatory=$true)][string]$SelectedOption,
  [string]$Rationale = "",
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data }
  if ($Json) { $obj | ConvertTo-Json -Depth 30 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 20 }
  if (-not $ok) { exit 1 }
}

function Safe-Slug([string]$Text) {
  return (($Text.ToLowerInvariant() -replace '[^a-z0-9]+','-').Trim('-'))
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  if (-not $Id) { $Id = Safe-Slug $Title }
  if (-not $Id) { Out-Result $false "INVALID_ID" "无法从标题生成 plan id；请显式传入 -Id。" }
  $planDir = Join-Path $RepoRoot "docs/spec/$Id"
  $planFile = Join-Path $planDir "PLAN.md"
  if (-not (Test-Path -LiteralPath $planFile -PathType Leaf)) {
    Out-Result $false "PLAN_NOT_FOUND" "未找到对应 PLAN.md，拒绝写入全局单槽决策。" @{ id=$Id; path=$planFile }
  }
  $decisionFile = Join-Path $planDir "DECISION.md"
  $memoryDir = Join-Path $RepoRoot ".aicoding/memory"
  $memoryFile = Join-Path $memoryDir "DECISIONS.md"
  $now = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss zzz")
  $decisionText = @"
# 已选择方案（Decision）：$Title

Decision Status: Selected

Selected option: $SelectedOption

## 选择理由

$Rationale

## 记录时间

$now
"@
  Set-Content -LiteralPath $decisionFile -Value $decisionText -Encoding UTF8
  New-Item -ItemType Directory -Force -Path $memoryDir | Out-Null
  $entry = "`n## $Title`n`nDecision Status: Selected`n`nSelected option: $SelectedOption`n`nRationale: $Rationale`n`nRecorded: $now`n"
  if (Test-Path -LiteralPath $memoryFile -PathType Leaf) {
    Add-Content -LiteralPath $memoryFile -Value $entry -Encoding UTF8
  } else {
    Set-Content -LiteralPath $memoryFile -Value ("# Agent Decisions`n" + $entry) -Encoding UTF8
  }
  Out-Result $true "OK" "用户技术路线选择已记录；批准与 Tree 绑定由 Plan CLI 单独完成。" @{
    id=$Id
    decision=$decisionFile
    decisionMemory=$memoryFile
  }
}
catch {
  Out-Result $false "INTERNAL_ERROR" ("确认用户决策时发生内部错误：{0}" -f $_.Exception.Message)
}
