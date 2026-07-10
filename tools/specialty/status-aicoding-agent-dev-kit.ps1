param([string]$RepoRoot = "", [switch]$Json)

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 20 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 10 }
  if (-not $ok) { exit 1 }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $pluginPath = Join-Path $RepoRoot "dist\aicoding-agent-dev-kit\plugins\AiCodingAgentDevKit"
  $assetRoot = Join-Path $pluginPath "assets\aicoding-agent-dev-kit"
  $manifest = Join-Path $pluginPath ".codex-plugin\plugin.json"
  $marketplace = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  $state = Join-Path $RepoRoot ".aicoding-agent-dev-kit\install-state.json"
  $cmd = Get-Command aicoding-agent-kit -ErrorAction SilentlyContinue
$cliPath = if ($cmd) { $cmd.Source } else { $null }
if (-not $cliPath) {
  try {
    $scriptDir = python -c "import sysconfig; print(sysconfig.get_path('scripts', scheme='nt_user') or '')" 2>$null | Select-Object -First 1
    if ($scriptDir) {
      $candidate = Join-Path ([string]$scriptDir) "aicoding-agent-kit.exe"
      if (Test-Path -LiteralPath $candidate) { $cliPath = $candidate }
    }
  } catch {}
}
  $marketplaceHasEntry = $false
  if (Test-Path -LiteralPath $marketplace) {
    $data = Get-Content -LiteralPath $marketplace -Raw | ConvertFrom-Json
    $marketplaceHasEntry = (@($data.plugins | Where-Object { $_.name -eq "aicoding-agent-dev-kit" }).Count -gt 0)
  }
  $result = [ordered]@{
    repoRoot=$RepoRoot
    pluginExists=(Test-Path -LiteralPath $pluginPath)
    manifestExists=(Test-Path -LiteralPath $manifest)
    assetRootExists=(Test-Path -LiteralPath $assetRoot)
    marketplaceExists=(Test-Path -LiteralPath $marketplace)
    marketplaceHasEntry=$marketplaceHasEntry
    stateExists=(Test-Path -LiteralPath $state)
    cli=$cliPath
  }
  $ok = $result.pluginExists -and $result.manifestExists -and $result.assetRootExists -and $result.marketplaceExists -and $result.marketplaceHasEntry
  Out-Result $ok ($(if ($ok) { "OK" } else { "PARTIAL" })) "AiCoding Agent Dev Kit status" $result
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
