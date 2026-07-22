# RETIRE-AFTER: after AEF consumers call aicoding plan verify directly
# Compatibility wrapper only.
[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [string]$Event = "manual",
  [ValidateSet("warn","enforce")][string]$Mode = "warn",
  [switch]$Json
)

$ErrorActionPreference = "Stop"
if (-not $RepoRoot) {
  $RepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..\..\..")).Path
}
$cli = Join-Path $RepoRoot "bin/aicoding.exe"
if (-not (Test-Path -LiteralPath $cli -PathType Leaf)) {
  throw "缺少 AiCoding CLI；请先运行 bootstrap。"
}
$output = & $cli plan verify --repo-root $RepoRoot --json 2>&1
$exitCode = $LASTEXITCODE
if ($Json) { $output } else {
  try { Write-Host ((($output | Out-String) | ConvertFrom-Json).message) } catch { $output }
}
if ($exitCode -ne 0 -and $Mode -eq "enforce") { exit 1 }
exit 0
