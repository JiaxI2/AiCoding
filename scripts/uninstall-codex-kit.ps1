param([switch]$DryRun, [switch]$Json)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$codex = Test-CodexPluginCli
$result = [ordered]@{
    dryRun=[bool]$DryRun
    plugin='aicoding'
    marketplace=(Resolve-KitPath $repo $config.agents.marketplacePath)
    codexPluginCli=$codex
    note='Uninstall through Codex /plugins or supported Codex plugin CLI. This script never deletes plugin cache directories directly.'
}
if($Json){ $result | ConvertTo-Json -Depth 6 } else { $result | Format-List }