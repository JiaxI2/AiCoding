[CmdletBinding(SupportsShouldProcess)]
param(
  [string]$PackageRoot = "",
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

function Same-ResolvedPath([string]$Left, [string]$Right) {
  if (-not (Test-Path -LiteralPath $Left) -or -not (Test-Path -LiteralPath $Right)) { return $false }
  $l = (Resolve-Path -LiteralPath $Left).Path.TrimEnd('\')
  $r = (Resolve-Path -LiteralPath $Right).Path.TrimEnd('\')
  return [string]::Equals($l, $r, [System.StringComparison]::OrdinalIgnoreCase)
}

try {
  if (-not $PackageRoot) { $PackageRoot = Split-Path -Parent $PSScriptRoot }
  $PackageRoot = (Resolve-Path -LiteralPath $PackageRoot).Path
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path

  $sourcePlugin = Join-Path $PackageRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  if (-not (Test-Path -LiteralPath $sourcePlugin)) {
    $sourcePlugin = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  }
  if (-not (Test-Path -LiteralPath $sourcePlugin)) { Out-Result $false "PACKAGE_INVALID" "Plugin source not found" @{ sourcePlugin=$sourcePlugin } }

  $targetPlugin = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  $marketplacePath = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  $assetRoot = Join-Path $targetPlugin "assets\ai-debug-repair-kit"
  $repoScripts = Join-Path $RepoRoot "scripts"
  $stateDir = Join-Path $RepoRoot ".ai-debug-repair"
  $statePath = Join-Path $stateDir "install-state.json"
  $samePluginPath = Same-ResolvedPath $sourcePlugin $targetPlugin

  $plannedScripts = @("uninstall-ai-debug-repair-kit.ps1", "status-ai-debug-repair-kit.ps1", "verify-ai-debug-repair-kit.ps1", "test-ai-debug-repair-kit.ps1")
  $plan = [ordered]@{
    dryRun = [bool]$DryRun
    packageRoot = $PackageRoot
    repoRoot = $RepoRoot
    sourcePlugin = $sourcePlugin
    targetPlugin = $targetPlugin
    sourceEqualsTarget = $samePluginPath
    marketplacePath = $marketplacePath
    assetRoot = $assetRoot
    scripts = $plannedScripts
    statePath = $statePath
    skipPipInstall = [bool]$SkipPipInstall
    actions = @()
  }

  if (-not $samePluginPath) { $plan.actions += "copy-plugin" } else { $plan.actions += "use-existing-plugin" }
  $plan.actions += "update-marketplace"
  if (-not $SkipPipInstall) { $plan.actions += "pip-install" }
  $plan.actions += "copy-lifecycle-scripts"
  $plan.actions += "write-install-state"

  if ($DryRun) {
    Out-Result $true "OK" "AI Debug Repair Kit install dry-run completed" $plan
    exit 0
  }

  if (-not $samePluginPath) {
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $targetPlugin) | Out-Null
    if (Test-Path -LiteralPath $targetPlugin) {
      if ($PSCmdlet.ShouldProcess($targetPlugin, "Remove existing plugin directory")) {
        Remove-Item -Recurse -Force -LiteralPath $targetPlugin
      }
    }
    Copy-Item -Recurse -Force -LiteralPath $sourcePlugin -Destination $targetPlugin
  }

  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $marketplacePath) | Out-Null
  if (Test-Path -LiteralPath $marketplacePath) {
    $marketplace = Get-Content -LiteralPath $marketplacePath -Raw | ConvertFrom-Json
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
  $marketplace | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $marketplacePath -Encoding UTF8

  $pipInstalled = $false
  if (-not $SkipPipInstall) {
    python -m pip install --user --force-reinstall $assetRoot | Out-Host
    if ($LASTEXITCODE -ne 0) { Out-Result $false "PIP_INSTALL_FAILED" "pip install failed" @{ assetRoot=$assetRoot } }
    $pipInstalled = $true
  }

  New-Item -ItemType Directory -Force -Path $repoScripts | Out-Null
  foreach ($script in $plannedScripts) {
    $scriptSource = Join-Path $PackageRoot "scripts\$script"
    $scriptTarget = Join-Path $repoScripts $script
    if (Test-Path -LiteralPath $scriptSource) {
      if (-not (Same-ResolvedPath $scriptSource $scriptTarget)) {
        Copy-Item -Force -LiteralPath $scriptSource -Destination $scriptTarget
      }
    } elseif (-not (Test-Path -LiteralPath $scriptTarget)) {
      Out-Result $false "SCRIPT_MISSING" "Lifecycle script source not found" @{ script=$script; source=$scriptSource }
    }
  }

  New-Item -ItemType Directory -Force -Path $stateDir | Out-Null
  $state = [ordered]@{ installed=$true; packageRoot=$PackageRoot; repoRoot=$RepoRoot; pluginPath=$targetPlugin; marketplacePath=$marketplacePath; pipInstalled=$pipInstalled; installedAt=(Get-Date).ToString("o") }
  $state | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $statePath -Encoding UTF8
  Out-Result $true "OK" "AI Debug Repair Kit installed" $state
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }