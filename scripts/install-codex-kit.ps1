param([switch]$DryRun, [switch]$Json)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$marketplace = Resolve-KitPath $repo $config.agents.marketplacePath
$plugin = Resolve-KitPath $repo $config.agents.pluginPath
$codex = Test-CodexPluginCli
$actions = @(
    "Verify submodule and plugin package",
    "Configure repository Git hooks: git config core.hooksPath .githooks",
    "Register marketplace when Codex plugin CLI is available: $marketplace",
    "Install plugin name: aicoding",
    "Review plugin hooks in Codex with /hooks"
)
$result = [ordered]@{ dryRun=[bool]$DryRun; marketplace=$marketplace; plugin=$plugin; codexPluginCli=$codex; actions=$actions }
if (-not (Test-Path -LiteralPath $plugin)) { throw "Missing plugin package: $plugin" }
if (-not (Test-Path -LiteralPath $marketplace)) { throw "Missing marketplace: $marketplace" }
if ($DryRun) { if($Json){ $result | ConvertTo-Json -Depth 6 } else { $result }; exit 0 }
& git -C $repo config core.hooksPath .githooks
if ($codex.available) {
    $result.note = 'Codex plugin CLI is available; register/install through the supported Codex plugin surface for this CLI version. If command syntax differs, use /plugins UI with the marketplace path.'
} else {
    $result.note = 'Codex plugin CLI is not available in this environment. Use Codex /plugins UI to add the marketplace, then install aicoding. Do not edit plugin cache manually.'
}
if($Json){ $result | ConvertTo-Json -Depth 6 } else { $result | Format-List }