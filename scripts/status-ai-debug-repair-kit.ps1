param([string]$RepoRoot = "", [switch]$Json)
function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 20 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 10 }
  if (-not $ok) { exit 1 }
}
try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path $RepoRoot).Path
  $pluginPath = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  $manifest = Join-Path $pluginPath ".codex-plugin\plugin.json"
  $marketplace = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  $state = Join-Path $RepoRoot ".ai-debug-repair\install-state.json"
  $airepair = $null
  try { $airepair = (Get-Command airepair -ErrorAction Stop).Source } catch {}
  $data = [ordered]@{ repoRoot=$RepoRoot; pluginExists=(Test-Path $pluginPath); manifestExists=(Test-Path $manifest); marketplaceExists=(Test-Path $marketplace); stateExists=(Test-Path $state); airepair=$airepair }
  $ok = $data.pluginExists -and $data.manifestExists -and $data.marketplaceExists
  Out-Result $ok ($(if ($ok) { "OK" } else { "PARTIAL" })) "AI Debug Repair Kit status" $data
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
