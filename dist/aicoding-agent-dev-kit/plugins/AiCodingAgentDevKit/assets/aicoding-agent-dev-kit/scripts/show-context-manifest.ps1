param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$path = Join-Path $root ".agent-dev-kit/context/context-manifest.json"
if (-not (Test-Path -LiteralPath $path)) {
  Write-AgentDevKitJson -Json:$Json -Data @{ ok=$false; error="context manifest not found"; path=$path }
  exit 1
}
if ($Json) {
  Get-Content -Raw -LiteralPath $path
} else {
  $m = Get-Content -Raw -LiteralPath $path | ConvertFrom-Json
  Write-Host "Stage: $($m.stage)"
  Write-Host "Reason: $($m.reason)"
  Write-Host "Chars: $($m.chars)"
  Write-Host "Rough tokens: $($m.roughTokens)"
  Write-Host "Included: $($m.includedFiles.Count)"
  Write-Host "Skipped: $($m.skippedFiles.Count)"
  Write-Host "Truncated: $($m.truncatedFiles.Count)"
}
