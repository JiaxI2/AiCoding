[CmdletBinding(SupportsShouldProcess=$true)]
param(
  [string]$RepoRoot = "",
  [Parameter(Mandatory=$true)][string]$Title,
  [Parameter(Mandatory=$true)][string]$SelectedOption,
  [string]$Rationale = "",
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Result([bool]$Ok, [string]$Code, [string]$Message, $Data = @{}) {
  $obj = [ordered]@{ schema_version="1.0"; ok=$Ok; code=$Code; message=$Message; data=$Data }
  if ($Json) { $obj | ConvertTo-Json -Depth 30 } else { Write-Host ("[{0}] {1}" -f $Code, $Message); $Data | ConvertTo-Json -Depth 20 }
  if (-not $Ok) { exit 1 }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $specDir = Join-Path $RepoRoot "spec"
  $memoryDir = Join-Path $RepoRoot ".agent-memory"
  if (-not (Test-Path -LiteralPath $specDir)) { New-Item -ItemType Directory -Path $specDir -Force | Out-Null }
  if (-not (Test-Path -LiteralPath $memoryDir)) { New-Item -ItemType Directory -Path $memoryDir -Force | Out-Null }

  $now = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss zzz")
  $selectedPath = Join-Path $specDir "SELECTED_SOLUTION.md"
  $needsPath = Join-Path $specDir "NEEDS_USER_DECISION.md"
  $memoryPath = Join-Path $memoryDir "DECISIONS.md"

  $selectedText = @"
# 已选择方案（Selected Solution）：$Title

Decision Status: Selected

Selected option: $SelectedOption

## 选择理由

$Rationale

## 记录时间

$now
"@
  Set-Content -LiteralPath $selectedPath -Value $selectedText -Encoding UTF8

  $decisionEntry = @"

## $Title

Decision Status: Selected

Selected option: $SelectedOption

Rationale: $Rationale

Recorded: $now
"@
  if (Test-Path -LiteralPath $memoryPath -PathType Leaf) {
    Add-Content -LiteralPath $memoryPath -Value $decisionEntry -Encoding UTF8
  } else {
    Set-Content -LiteralPath $memoryPath -Value ("# Agent Decisions`r`n" + $decisionEntry) -Encoding UTF8
  }

  $removedNeedsDecision = $false
  if (Test-Path -LiteralPath $needsPath -PathType Leaf) {
    if ($PSCmdlet.ShouldProcess($needsPath, "移除 Plan Mode 待用户决策阻塞文件")) {
      Remove-Item -LiteralPath $needsPath -Force
      $removedNeedsDecision = $true
    }
  }

  Out-Result $true "OK" "用户技术路线选择已记录，Plan Mode 阻塞标记已处理。" ([ordered]@{
    repoRoot=$RepoRoot
    selectedSolution=$selectedPath
    decisionMemory=$memoryPath
    removedNeedsDecision=$removedNeedsDecision
  })
}
catch {
  Out-Result $false "INTERNAL_ERROR" ("确认用户决策时发生内部错误：{0}" -f $_.Exception.Message)
}
