[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [string]$Event = "manual",
  [ValidateSet("warn","enforce")][string]$Mode = "warn",
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Hook($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; event=$Event; mode=$Mode; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 50 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 20 }
  if (-not $ok -and $Mode -eq "enforce") { exit 1 }
  exit 0
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..\..\..")).Path }
  $script = Join-Path $RepoRoot "tools/specialty/verify-agent-dev-kit-plan-mode.ps1"
  if (-not (Test-Path -LiteralPath $script -PathType Leaf)) {
    Out-Hook $false "MISSING_VERIFY" "缺少 Plan Mode 验证脚本：tools/specialty/verify-agent-dev-kit-plan-mode.ps1。" @{ repoRoot=$RepoRoot }
  }

  $capture = & pwsh -NoProfile -ExecutionPolicy Bypass -File $script -RepoRoot $RepoRoot -Json 2>&1
  $ok = ($LASTEXITCODE -eq 0)
  $parsed = $null
  try { $parsed = ($capture | Out-String).Trim() | ConvertFrom-Json } catch { $parsed = @{ raw=($capture | Out-String) } }

  $nextSteps = @"
检测到架构敏感变更，但没有找到已接受的用户决策记录，或仍存在待用户选择标记。

请先生成 Plan Mode 会话：
pwsh tools/specialty/new-agent-plan-mode-session.ps1 -Feature "<功能名>" -Description "<需求描述>" -NeedsDecision -Json

然后让用户从 docs/decisions/plan-mode-overlay/PRD_OPTIONS.md 中选择技术路线。

用户选择后执行：
pwsh tools/specialty/confirm-agent-decision.ps1 -Title "<标题>" -SelectedOption "<用户选择的方案>" -Rationale "<选择理由>" -Json
"@
  $message = if ($ok) { "Plan Mode 门禁验证通过。" } else { "Plan Mode 门禁未通过：用户尚未完成技术路线选择或计划产物不完整。" }
  $data = if ($ok) { @{ verify=$parsed } } else { @{ verify=$parsed; nextSteps=$nextSteps } }
  $code = if ($ok) { "OK" } else { "PLAN_MODE_BLOCKED" }
  Out-Hook $ok $code $message $data
}
catch {
  Out-Hook $false "INTERNAL_ERROR" ("Plan Mode 门禁执行时发生内部错误：{0}" -f $_.Exception.Message)
}
