param([string]$RepoRoot = "", [switch]$Json, [switch]$UninstallPip)
$ErrorActionPreference = "Stop"
function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 20 } else { Write-Host "[$code] $message" }
  if (-not $ok) { exit 1 }
}
try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path $RepoRoot).Path
  $removed = @()
  $pluginPath = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  if (Test-Path $pluginPath) { Remove-Item -Recurse -Force $pluginPath; $removed += $pluginPath }
  $marketplacePath = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  if (Test-Path $marketplacePath) {
    $marketplace = Get-Content $marketplacePath -Raw | ConvertFrom-Json
    $plugins = @()
    if ($marketplace.plugins) { foreach ($p in $marketplace.plugins) { if ($p.name -ne "aicoding-ai-debug-repair-kit") { $plugins += $p } } }
    $marketplace | Add-Member -NotePropertyName plugins -NotePropertyValue $plugins -Force
    $marketplace | ConvertTo-Json -Depth 30 | Set-Content -Encoding UTF8 $marketplacePath
  }
  $statePath = Join-Path $RepoRoot ".ai-debug-repair\install-state.json"
  if (Test-Path $statePath) { Remove-Item -Force $statePath; $removed += $statePath }
  $pipUninstalled = $false
  if ($UninstallPip) { python -m pip uninstall -y ai-debug-repair-kit | Out-Host; $pipUninstalled = $true }
  Out-Result $true "OK" "AI Debug Repair Kit uninstalled" @{ removed=$removed; pipUninstalled=$pipUninstalled }
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
