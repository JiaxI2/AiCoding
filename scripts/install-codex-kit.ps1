param([switch]$DryRun, [switch]$Json)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$marketplace = Resolve-KitPath $repo $config.agents.marketplacePath
$plugin = Resolve-KitPath $repo $config.agents.pluginPath
$marketplaceRoot = $repo
$marketplacePluginLink = Join-Path $repo 'plugins\AiCoding'
$codex = Test-CodexPluginCli

function Ensure-Junction {
    param([string]$Link, [string]$Target)
    if (-not (Test-Path -LiteralPath $Target)) { throw "Missing target: $Target" }
    if (Test-Path -LiteralPath $Link) {
        $item = Get-Item -LiteralPath $Link -Force
        $targetPath = (Resolve-Path -LiteralPath $Target).Path
        $targets = @($item.Target | ForEach-Object { if ($_ -and (Test-Path -LiteralPath $_)) { (Resolve-Path -LiteralPath $_).Path } else { $_ } })
        if ($item.LinkType -and ($targets -contains $targetPath)) { return 'exists' }
        throw "Refusing to overwrite existing local marketplace path: $Link"
    }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Link) | Out-Null
    New-Item -ItemType Junction -Path $Link -Target $Target | Out-Null
    return 'created'
}

$actions = @(
    "Verify submodule and plugin package",
    "Configure repository Git hooks: git config core.hooksPath .githooks",
    "Ensure local marketplace plugin link: $marketplacePluginLink -> $plugin",
    "Register marketplace root when Codex plugin CLI is available: $marketplaceRoot",
    "Install plugin id: aicoding@aicoding-platform",
    "Review plugin hooks in Codex with /hooks"
)
$result = [ordered]@{
    dryRun=[bool]$DryRun
    marketplace=$marketplace
    marketplaceRoot=$marketplaceRoot
    marketplacePluginLink=$marketplacePluginLink
    plugin=$plugin
    codexPluginCli=$codex
    actions=$actions
}
if (-not (Test-Path -LiteralPath $plugin)) { throw "Missing plugin package: $plugin" }
if (-not (Test-Path -LiteralPath $marketplace)) { throw "Missing marketplace: $marketplace" }
if ($DryRun) { if($Json){ $result | ConvertTo-Json -Depth 6 } else { $result }; exit 0 }
& git -C $repo config core.hooksPath .githooks
$result.marketplacePluginLinkResult = Ensure-Junction -Link $marketplacePluginLink -Target $plugin
if ($codex.available) {
    $codexPath = $codex.path
    $result.marketplaceAdd = @(& $codexPath plugin marketplace add $marketplaceRoot --json 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "Codex marketplace registration failed: $($result.marketplaceAdd -join "`n")" }
    $result.pluginAdd = @(& $codexPath plugin add 'aicoding@aicoding-platform' --json 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "Codex plugin installation failed: $($result.pluginAdd -join "`n")" }
    $result.note = 'Codex plugin CLI completed marketplace registration and aicoding plugin installation. Review plugin hooks in Codex with /hooks.'
} else {
    $result.note = 'Codex plugin CLI is not available in this environment. Use Codex /plugins UI to add the marketplace, then install aicoding. Do not edit plugin cache manually.'
}
if($Json){ $result | ConvertTo-Json -Depth 6 } else { $result | Format-List }
