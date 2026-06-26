param(
    [string]$BackupPath,
    [switch]$DryRun,
    [switch]$Json
)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force

function Expand-RuntimePath {
    param([string]$PathValue)
    if ([string]::IsNullOrWhiteSpace($PathValue)) { return $null }
    return [Environment]::ExpandEnvironmentVariables($PathValue.Replace('%USERPROFILE%', $env:USERPROFILE))
}

$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$legacyRoot = Expand-RuntimePath $config.skillRuntime.legacyUserRoot
if (-not $BackupPath) { $BackupPath = "$legacyRoot.legacy-backup" }
$actions = @(
    "Inspect backup path: $BackupPath",
    "Inspect legacy root: $legacyRoot",
    'Remove only managed links recorded in install-state when that schema is present',
    'Restore legacy root only after confirming target path is absent or approved',
    'Run scripts/audit-runtime-skills.ps1 -Json after restore',
    'Restart Codex and verify skill discovery manually'
)

if (-not $DryRun) {
    throw 'Real restore is intentionally not automated in the MVP. Re-run with -DryRun and execute approved recovery steps manually.'
}

$result = [pscustomobject]@{
    dryRun = [bool]$DryRun
    backupPath = $BackupPath
    legacyRoot = $legacyRoot
    backupExists = Test-Path -LiteralPath $BackupPath
    legacyRootExists = Test-Path -LiteralPath $legacyRoot
    actions = $actions
}
if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
