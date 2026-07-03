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
  $script = Join-Path $RepoRoot "scripts/verify-agent-dev-kit-plan-mode.ps1"
  if (-not (Test-Path -LiteralPath $script -PathType Leaf)) {
    Out-Hook $false "MISSING_VERIFY" "scripts/verify-agent-dev-kit-plan-mode.ps1 not found" @{ repoRoot=$RepoRoot }
  }

  $capture = & pwsh -NoProfile -ExecutionPolicy Bypass -File $script -RepoRoot $RepoRoot -Json 2>&1
  $ok = ($LASTEXITCODE -eq 0)
  $parsed = $null
  try { $parsed = ($capture | Out-String).Trim() | ConvertFrom-Json } catch { $parsed = @{ raw=($capture | Out-String) } }

  Out-Hook $ok ($(if ($ok) { "OK" } else { "PLAN_MODE_BLOCKED" })) "Plan mode gate completed" @{ verify=$parsed }
}
catch {
  Out-Hook $false "INTERNAL_ERROR" $_.Exception.Message
}
