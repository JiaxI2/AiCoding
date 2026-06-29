param(
  [string]$PackageRoot = "",
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$SkipPipInstall
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 30 } else { Write-Host "[$code] $message" }
  if (-not $ok) { exit 1 }
}

try {
  if (-not $PackageRoot) { $PackageRoot = Split-Path -Parent $PSScriptRoot }
  $PackageRoot = (Resolve-Path $PackageRoot).Path
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path $RepoRoot).Path

  $sourcePlugin = Join-Path $PackageRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  if (-not (Test-Path $sourcePlugin)) {
    $sourcePlugin = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  }
  if (-not (Test-Path $sourcePlugin)) { Out-Result $false "PACKAGE_INVALID" "Plugin source not found" @{ sourcePlugin=$sourcePlugin } }

  $targetPlugin = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  if ((Resolve-Path $sourcePlugin).Path -ne (Resolve-Path (Split-Path -Parent $targetPlugin) -ErrorAction SilentlyContinue).Path) {
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $targetPlugin) | Out-Null
    if (Test-Path $targetPlugin) { Remove-Item -Recurse -Force $targetPlugin }
    Copy-Item -Recurse -Force $sourcePlugin $targetPlugin
  }

  $marketplacePath = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $marketplacePath) | Out-Null
  if (Test-Path $marketplacePath) {
    $marketplace = Get-Content $marketplacePath -Raw | ConvertFrom-Json
  } else {
    $marketplace = [pscustomobject]@{ name="aicoding-local-marketplace"; interface=[pscustomobject]@{displayName="AiCoding Local Marketplace"}; plugins=@() }
  }

  $newPlugin = [pscustomobject]@{
    name="aicoding-ai-debug-repair-kit"
    source=[pscustomobject]@{ source="local"; path="./dist/ai-debug-repair-kit/plugins/AiCodingAIDebugRepairKit" }
    policy=[pscustomobject]@{ installation="AVAILABLE"; authentication="ON_INSTALL" }
    category="Developer Tools"
  }
  $plugins = @()
  if ($marketplace.plugins) {
    foreach ($p in $marketplace.plugins) { if ($p.name -ne "aicoding-ai-debug-repair-kit") { $plugins += $p } }
  }
  $plugins += $newPlugin
  $marketplace | Add-Member -NotePropertyName plugins -NotePropertyValue $plugins -Force
  $marketplace | ConvertTo-Json -Depth 30 | Set-Content -Encoding UTF8 $marketplacePath

  $assetRoot = Join-Path $targetPlugin "assets\ai-debug-repair-kit"
  $pipInstalled = $false
  if (-not $SkipPipInstall) {
    python -m pip install --user --force-reinstall $assetRoot | Out-Host
    if ($LASTEXITCODE -ne 0) { Out-Result $false "PIP_INSTALL_FAILED" "pip install failed" @{ assetRoot=$assetRoot } }
    $pipInstalled = $true
  }

  $repoScripts = Join-Path $RepoRoot "scripts"
  New-Item -ItemType Directory -Force -Path $repoScripts | Out-Null
  foreach ($script in @("uninstall-ai-debug-repair-kit.ps1","status-ai-debug-repair-kit.ps1","verify-ai-debug-repair-kit.ps1")) {
    Copy-Item -Force (Join-Path $PackageRoot "scripts\$script") (Join-Path $repoScripts $script)
  }

  $stateDir = Join-Path $RepoRoot ".ai-debug-repair"
  New-Item -ItemType Directory -Force -Path $stateDir | Out-Null
  $state = [ordered]@{ installed=$true; packageRoot=$PackageRoot; repoRoot=$RepoRoot; pluginPath=$targetPlugin; marketplacePath=$marketplacePath; pipInstalled=$pipInstalled; installedAt=(Get-Date).ToString("o") }
  $state | ConvertTo-Json -Depth 20 | Set-Content -Encoding UTF8 (Join-Path $stateDir "install-state.json")
  Out-Result $true "OK" "AI Debug Repair Kit installed" $state
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
