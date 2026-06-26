param(
    [string]$SourceRepository,
    [ValidateSet('runtime','full','skill-development')]
    [string]$Profile = 'full',
    [string]$BackupPath,
    [switch]$DryRun,
    [switch]$Json,
    [switch]$SkipPluginRefresh,
    [switch]$KeepLegacyRoot
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
if (-not $SourceRepository) { $SourceRepository = Expand-RuntimePath $config.skillRuntime.defaultSourceRepository }
$legacyRoot = Expand-RuntimePath $config.skillRuntime.legacyUserRoot
$agentsRoot = Expand-RuntimePath $config.skillRuntime.canonicalUserRoot
if (-not $BackupPath) { $BackupPath = "$legacyRoot.legacy-backup" }

$legacySkillCount = 0
if (Test-Path -LiteralPath $legacyRoot) {
    $legacySkillCount = @((Get-ChildItem -LiteralPath $legacyRoot -Recurse -Filter 'SKILL.md' -File -ErrorAction SilentlyContinue)).Count
}
$sourceStatus = if (Test-Path -LiteralPath $SourceRepository) { @(& git -C $SourceRepository status --short 2>$null) } else { @('missing') }
$actions = @(
    "Inventory legacy root: $legacyRoot",
    "Verify source repository: $SourceRepository",
    "Ensure user skill root exists: $agentsRoot",
    "Apply profile through scripts/set-codex-skill-profile.ps1 -Profile $Profile -DryRun",
    'Run scripts/audit-runtime-skills.ps1 -Json',
    'Refresh AiCoding Plugin through supported Codex plugin surface when available',
    'Ask before renaming or changing legacy root'
)
if ($KeepLegacyRoot) { $actions += 'Keep legacy root in place and report remaining duplicate exposure.' }
else { $actions += "Proposed legacy backup path: $BackupPath" }
if ($SkipPluginRefresh) { $actions += 'Skip plugin refresh by explicit option.' }

if (-not $DryRun) {
    throw 'Real migration is intentionally not automated in the MVP. Re-run with -DryRun, review the plan, then perform approved steps explicitly.'
}

$result = [pscustomobject]@{
    dryRun = [bool]$DryRun
    profile = $Profile
    sourceRepository = $SourceRepository
    sourceRepositoryStatus = $sourceStatus
    legacyRoot = $legacyRoot
    legacySkillCount = $legacySkillCount
    agentsRoot = $agentsRoot
    backupPath = $BackupPath
    keepLegacyRoot = [bool]$KeepLegacyRoot
    skipPluginRefresh = [bool]$SkipPluginRefresh
    actions = $actions
}
if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
