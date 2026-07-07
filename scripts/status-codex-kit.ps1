# Deprecated: this fast-path check is superseded by bin\aicoding.exe status --all --json.
# Kept as a temporary fallback for v0.1.x.
# Do not call from Taskfile smoke or Git hooks.

param([switch]$Json)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$submodule = Resolve-KitPath $repo $config.agents.skillsSubmodule
$plugin = Resolve-KitPath $repo $config.agents.pluginPath
$marketplace = Resolve-KitPath $repo $config.agents.marketplacePath
$buildInfoPath = Join-Path $plugin 'BUILDINFO.json'
$buildInfo = if (Test-Path -LiteralPath $buildInfoPath) { Get-Content -Raw -LiteralPath $buildInfoPath | ConvertFrom-Json } else { $null }
$assetStatus = @{}
foreach ($prop in $config.assets.PSObject.Properties) {
    $assetStatus[$prop.Name] = Test-Path -LiteralPath (Resolve-KitPath $repo $prop.Value)
}
$result = [pscustomobject]@{
    repository = $repo
    branch = (& git -C $repo branch --show-current).Trim()
    workingTree = @(& git -C $repo status --porcelain)
    submodule = Get-SubmoduleStatus $submodule
    pluginPath = $plugin
    pluginExists = Test-Path -LiteralPath $plugin
    marketplacePath = $marketplace
    marketplaceExists = Test-Path -LiteralPath $marketplace
    buildInfo = $buildInfo
    codexPluginCli = Test-CodexPluginCli
    assets = $assetStatus
    rules = $config.rules
    skillRuntime = $config.skillRuntime
    profiles = $config.profiles
}
if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
