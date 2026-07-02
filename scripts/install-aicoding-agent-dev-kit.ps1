param(
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$SkipPipInstall,
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 30 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 10 }
  if (-not $ok) { exit 1 }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path

  $pluginPath = Join-Path $RepoRoot "dist\aicoding-agent-dev-kit\plugins\AiCodingAgentDevKit"
  $assetRoot = Join-Path $pluginPath "assets\aicoding-agent-dev-kit"
  $marketplacePath = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  $stateDir = Join-Path $RepoRoot ".aicoding-agent-dev-kit"
  $statePath = Join-Path $stateDir "install-state.json"

  if (-not (Test-Path -LiteralPath (Join-Path $pluginPath ".codex-plugin\plugin.json"))) {
    Out-Result $false "PACKAGE_INVALID" "Agent Dev Kit plugin manifest not found" @{ pluginPath=$pluginPath }
  }
  if (-not (Test-Path -LiteralPath (Join-Path $assetRoot "pyproject.toml"))) {
    Out-Result $false "ASSET_INVALID" "Agent Dev Kit asset package not found" @{ assetRoot=$assetRoot }
  }

  $plan = [ordered]@{
    dryRun = [bool]$DryRun
    repoRoot = $RepoRoot
    pluginPath = $pluginPath
    assetRoot = $assetRoot
    marketplacePath = $marketplacePath
    statePath = $statePath
    skipPipInstall = [bool]$SkipPipInstall
    actions = @("update-marketplace", "write-install-state")
  }
  if (-not $SkipPipInstall) { $plan.actions = @("update-marketplace", "pip-install", "write-install-state") }
  if ($DryRun) { Out-Result $true "OK" "AiCoding Agent Dev Kit install dry-run completed" $plan; exit 0 }

  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $marketplacePath) | Out-Null
  if (Test-Path -LiteralPath $marketplacePath) {
    $marketplace = Get-Content -LiteralPath $marketplacePath -Raw | ConvertFrom-Json
  } else {
    $marketplace = [pscustomobject]@{ name="aicoding-platform"; interface=[pscustomobject]@{displayName="AiCoding Platform"}; plugins=@() }
  }

  $entry = [pscustomobject]@{
    name="aicoding-agent-dev-kit"
    source=[pscustomobject]@{ source="local"; path="./dist/aicoding-agent-dev-kit/plugins/AiCodingAgentDevKit" }
    policy=[pscustomobject]@{ installation="AVAILABLE"; authentication="ON_INSTALL" }
    category="Developer Tools"
  }
  $plugins = @()
  if ($marketplace.plugins) {
    foreach ($p in $marketplace.plugins) { if ($p.name -ne "aicoding-agent-dev-kit") { $plugins += $p } }
  }
  $plugins += $entry
  $marketplace | Add-Member -NotePropertyName plugins -NotePropertyValue $plugins -Force
  $marketplace | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $marketplacePath -Encoding UTF8

  $pipInstalled = $false
  if (-not $SkipPipInstall) {
    python -m pip install --user --force-reinstall $assetRoot | Out-Host
    if ($LASTEXITCODE -ne 0) { Out-Result $false "PIP_INSTALL_FAILED" "pip install failed" @{ assetRoot=$assetRoot } }
    $pipInstalled = $true
  }

  New-Item -ItemType Directory -Force -Path $stateDir | Out-Null
  $state = [ordered]@{
    installed=$true
    name="aicoding-agent-dev-kit"
    version="0.11.1"
    repoRoot=$RepoRoot
    pluginPath=$pluginPath
    assetRoot=$assetRoot
    marketplacePath=$marketplacePath
    pipInstalled=$pipInstalled
    installedAt=(Get-Date).ToString("o")
  }
  $state | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $statePath -Encoding UTF8
  Out-Result $true "OK" "AiCoding Agent Dev Kit installed" $state
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
