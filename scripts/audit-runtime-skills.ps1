param(
    [switch]$Json,
    [switch]$Strict,
    [switch]$AllowLegacyRoot,
    [string]$ExpectedProfile
)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force

function Expand-RuntimePath {
    param([string]$PathValue)
    if ([string]::IsNullOrWhiteSpace($PathValue)) { return $null }
    $expanded = $PathValue.Replace('%USERPROFILE%', $env:USERPROFILE)
    return [Environment]::ExpandEnvironmentVariables($expanded)
}

function Get-SkillNameFromFile {
    param([string]$SkillFile)
    $text = Get-Content -LiteralPath $SkillFile -Raw -ErrorAction Stop
    $match = [regex]::Match($text, '(?ms)^---\s*(.*?)\s*---')
    if (-not $match.Success) { return $null }
    $nameMatch = [regex]::Match($match.Groups[1].Value, '(?m)^name:\s*[''\"]?([^''\"\r\n]+)[''\"]?\s*$')
    if (-not $nameMatch.Success) { return $null }
    return $nameMatch.Groups[1].Value.Trim()
}

function Get-RootSkillEntries {
    param([string]$Root, [string]$SourceType)
    $entries = @()
    if (-not (Test-Path -LiteralPath $Root)) { return $entries }
    foreach ($child in Get-ChildItem -LiteralPath $Root -Force -Directory) {
        $skillFile = Join-Path $child.FullName 'SKILL.md'
        if (Test-Path -LiteralPath $skillFile) {
            $item = Get-Item -LiteralPath $child.FullName -Force
            $target = $null
            if ($item.LinkType) { $target = @($item.Target) -join ';' }
            $entries += [pscustomobject]@{
                name = Get-SkillNameFromFile $skillFile
                path = $skillFile
                root = $Root
                sourceType = $SourceType
                linkType = $item.LinkType
                target = $target
            }
        }
    }
    return $entries
}

function Get-RecursiveSkillEntries {
    param([string]$Root, [string]$SourceType)
    $entries = @()
    if (-not (Test-Path -LiteralPath $Root)) { return $entries }
    foreach ($skillFile in Get-ChildItem -LiteralPath $Root -Force -Recurse -Filter 'SKILL.md' -File -ErrorAction SilentlyContinue) {
        $entries += [pscustomobject]@{
            name = Get-SkillNameFromFile $skillFile.FullName
            path = $skillFile.FullName
            root = $Root
            sourceType = $SourceType
            linkType = $null
            target = $null
        }
    }
    return $entries
}

function Test-PathUnder {
    param([string]$PathValue, [string]$RootValue)
    if ([string]::IsNullOrWhiteSpace($PathValue) -or [string]::IsNullOrWhiteSpace($RootValue)) { return $false }
    try {
        $p = [System.IO.Path]::GetFullPath($PathValue).TrimEnd('\') + '\'
        $r = [System.IO.Path]::GetFullPath($RootValue).TrimEnd('\') + '\'
        return $p.StartsWith($r, [System.StringComparison]::OrdinalIgnoreCase)
    } catch { return $false }
}

$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$agentsRoot = Expand-RuntimePath $config.skillRuntime.canonicalUserRoot
$legacyRoot = Expand-RuntimePath $config.skillRuntime.legacyUserRoot
$sourceRepository = Expand-RuntimePath $config.skillRuntime.defaultSourceRepository
$pluginCache = Join-Path $env:USERPROFILE '.codex\plugins\cache'

$entries = @()
$entries += Get-RootSkillEntries -Root $agentsRoot -SourceType 'agents-user-root'
$legacyEntries = Get-RecursiveSkillEntries -Root $legacyRoot -SourceType 'legacy-codex-root'
$entries += $legacyEntries
$entries += Get-RecursiveSkillEntries -Root $pluginCache -SourceType 'codex-plugin-cache'

$duplicateNames = @()
foreach ($group in ($entries | Where-Object { $_.name } | Group-Object -Property name | Where-Object { $_.Count -gt 1 })) {
    $duplicateNames += [pscustomobject]@{
        name = $group.Name
        sources = @($group.Group | ForEach-Object { $_.path })
    }
}

$brokenLinks = @()
$wholeRepositoryLinks = @()
$generatedSkillLinks = @()
if (Test-Path -LiteralPath $agentsRoot) {
    foreach ($child in Get-ChildItem -LiteralPath $agentsRoot -Force -Directory) {
        $item = Get-Item -LiteralPath $child.FullName -Force
        $targetText = if ($item.LinkType) { @($item.Target) -join ';' } else { $null }
        if ($item.LinkType -and $targetText) {
            foreach ($target in @($item.Target)) {
                if (-not (Test-Path -LiteralPath $target)) {
                    $brokenLinks += [pscustomobject]@{ path = $child.FullName; target = $target }
                }
                if ((Test-Path -LiteralPath (Join-Path $target '.git')) -and (Test-Path -LiteralPath (Join-Path $target 'config\aicoding-plugin-pack.json'))) {
                    $wholeRepositoryLinks += [pscustomobject]@{ path = $child.FullName; target = $target }
                }
                if ($target -match 'plugins[\\/]AiCoding[\\/]skills') {
                    $generatedSkillLinks += [pscustomobject]@{ path = $child.FullName; target = $target }
                }
            }
        }
    }
}

$sourceRepositoryUnderSkillRoot = (Test-PathUnder $sourceRepository $agentsRoot) -or (Test-PathUnder $sourceRepository $legacyRoot)
$legacyRootSkills = @($legacyEntries | Where-Object { $_.name }).Count
$profileKnown = $true
if ($ExpectedProfile) {
    $profileKnown = [bool]($config.profiles.PSObject.Properties.Name -contains $ExpectedProfile)
}

$ok = ($duplicateNames.Count -eq 0) -and ($brokenLinks.Count -eq 0) -and ($wholeRepositoryLinks.Count -eq 0) -and ($generatedSkillLinks.Count -eq 0) -and (-not $sourceRepositoryUnderSkillRoot) -and ($profileKnown)
if (-not $AllowLegacyRoot -and $legacyRootSkills -gt 0) { $ok = $false }

$result = [pscustomobject]@{
    ok = $ok
    expectedProfile = $ExpectedProfile
    profileKnown = $profileKnown
    activeSkills = @($entries | Where-Object { $_.name }).Count
    duplicateNames = $duplicateNames
    legacyRoot = $legacyRoot
    legacyRootSkills = $legacyRootSkills
    agentsRoot = $agentsRoot
    pluginCache = $pluginCache
    sourceRepository = $sourceRepository
    sourceRepositoryUnderSkillRoot = $sourceRepositoryUnderSkillRoot
    brokenLinks = $brokenLinks
    wholeRepositoryLinks = $wholeRepositoryLinks
    generatedSkillLinks = $generatedSkillLinks
    entries = $entries
}

if ($Json) {
    $result | ConvertTo-Json -Depth 10
} else {
    $result | Format-List
}

if ($Strict -and -not $ok) { exit 1 }
