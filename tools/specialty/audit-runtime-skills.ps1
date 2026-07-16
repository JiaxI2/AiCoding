param(
    [switch]$Json,
    [switch]$Strict,
    [switch]$AllowCodexRoot,
    [string]$ExpectedProfile,
    [ValidateSet('agents','codex')]
    [string]$StandaloneRoot = 'agents',
    [string]$Skill,
    [string]$SourceRepository,
    [switch]$IncludeAllPluginCaches
)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force

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

function Test-SamePath {
    param([string]$Left, [string]$Right)
    if ([string]::IsNullOrWhiteSpace($Left) -or [string]::IsNullOrWhiteSpace($Right)) { return $false }
    try {
        $leftPath = [System.IO.Path]::GetFullPath($Left).TrimEnd('\')
        $rightPath = [System.IO.Path]::GetFullPath($Right).TrimEnd('\')
        return $leftPath.Equals($rightPath, [System.StringComparison]::OrdinalIgnoreCase)
    } catch { return $false }
}

function Resolve-StandaloneSkillSourcePath {
    param($Config, [string]$SkillName)
    $sourcePaths = $Config.standaloneSkillRegistry.sourcePaths
    if ($sourcePaths) {
        $property = $sourcePaths.PSObject.Properties | Where-Object { $_.Name -eq $SkillName } | Select-Object -First 1
        if ($property) { return [string]$property.Value }
    }
    return $SkillName
}

function Find-CanonicalSkillPath {
    param([string]$Repository, [string]$SkillName)
    if (-not (Test-Path -LiteralPath $Repository)) { return $null }
    foreach ($skillFile in Get-ChildItem -LiteralPath $Repository -Recurse -Filter 'SKILL.md' -File -ErrorAction SilentlyContinue) {
        if ($skillFile.FullName -match 'plugins[\\/]AiCoding[\\/]skills') { continue }
        $name = Get-SkillNameFromFile $skillFile.FullName
        if ($name -eq $SkillName) { return Split-Path -Parent $skillFile.FullName }
    }
    return $null
}

function Get-LinkTarget {
    param([string]$PathValue)
    $item = Get-Item -LiteralPath $PathValue -Force -ErrorAction SilentlyContinue
    if (-not $item -or -not $item.LinkType) { return $null }
    return @($item.Target) | Select-Object -First 1
}

function Compare-PluginBuildInfo {
    param($Source, $Installed)
    $reasons = @()
    if (-not $Installed) { return @('installed plugin cache is missing BUILDINFO.json') }
    foreach ($field in @('pluginName','pluginVersion','sourceCommit','packManifestHash','pluginManifestHash','skillsDigest','hooksDigest')) {
        if ([string]$Source.$field -ne [string]$Installed.$field) { $reasons += "$field mismatch" }
    }
    if ([bool]$Installed.dirtySource) { $reasons += 'installed BUILDINFO.json reports dirtySource=true' }
    return $reasons
}

$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$agentsRoot = Resolve-CodexKitRuntimePath -RepoRoot $repo -PathValue $config.skillRuntime.canonicalUserRoot
$codexRoot = Resolve-CodexKitRuntimePath -RepoRoot $repo -PathValue $config.skillRuntime.codexUserRoot
if (-not $SourceRepository) { $SourceRepository = Resolve-CodexKitConfiguredPath -ConfigSection $config.skillRuntime -RepoRoot $repo }
$sourceRepository = [System.IO.Path]::GetFullPath($SourceRepository)
$pluginCache = Join-Path $env:USERPROFILE '.codex\plugins\cache'
$sourcePluginPath = Resolve-CodexKitRuntimePath -RepoRoot $repo -PathValue $config.agents.pluginPath
$pluginManifestPath = Join-Path $sourcePluginPath '.codex-plugin\plugin.json'
$sourceBuildInfoPath = Join-Path $sourcePluginPath 'BUILDINFO.json'
$pluginManifest = if (Test-Path -LiteralPath $pluginManifestPath) { Get-Content -LiteralPath $pluginManifestPath -Raw | ConvertFrom-Json } else { $null }
$sourcePluginBuildInfo = if (Test-Path -LiteralPath $sourceBuildInfoPath) { Get-Content -LiteralPath $sourceBuildInfoPath -Raw | ConvertFrom-Json } else { $null }
$installedPluginPath = if ($pluginManifest) { Join-Path $pluginCache (Join-Path 'aicoding-platform\aicoding' ([string]$pluginManifest.version)) } else { $null }
$installedBuildInfoPath = if ($installedPluginPath) { Join-Path $installedPluginPath 'BUILDINFO.json' } else { $null }
$installedPluginBuildInfo = if ($installedBuildInfoPath -and (Test-Path -LiteralPath $installedBuildInfoPath)) { Get-Content -LiteralPath $installedBuildInfoPath -Raw | ConvertFrom-Json } else { $null }
$pluginDriftReasons = if ($sourcePluginBuildInfo) { @(Compare-PluginBuildInfo -Source $sourcePluginBuildInfo -Installed $installedPluginBuildInfo) } else { @('source plugin BUILDINFO.json is missing') }
$pluginPackageDrift = ($pluginDriftReasons.Count -gt 0)

$entries = @()
$entries += Get-RootSkillEntries -Root $agentsRoot -SourceType 'agents-user-root'
$codexEntries = Get-RootSkillEntries -Root $codexRoot -SourceType 'codex-user-root'
$entries += $codexEntries
if ($IncludeAllPluginCaches) {
    $entries += Get-RecursiveSkillEntries -Root $pluginCache -SourceType 'codex-plugin-cache'
} elseif ($installedPluginPath) {
    $entries += Get-RecursiveSkillEntries -Root $installedPluginPath -SourceType 'aicoding-plugin-cache'
}

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
foreach ($skillRoot in @($agentsRoot, $codexRoot)) {
    if (Test-Path -LiteralPath $skillRoot) {
        foreach ($child in Get-ChildItem -LiteralPath $skillRoot -Force -Directory) {
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
}

$sourceRepositoryUnderSkillRoot = (Test-PathUnder $sourceRepository $agentsRoot) -or (Test-PathUnder $sourceRepository $codexRoot)
$codexRootSkills = @($codexEntries | Where-Object { $_.name }).Count
$profileKnown = $true
if ($ExpectedProfile) {
    $profileKnown = [bool]($config.profiles.PSObject.Properties.Name -contains $ExpectedProfile)
}

$missingExpectedSkills = @()
$unexpectedRegisteredSkills = @()
$mismatchedManagedLinks = @()
$missingSourceTargets = @()
$expectedByRoot = @{
    agents = @{}
    codex = @{}
}
if ($ExpectedProfile -and $profileKnown) {
    if ($ExpectedProfile -eq 'full') {
        foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = [System.IO.Path]::GetFullPath((Join-Path $sourceRepository $sourcePath))
            $expectedByRoot[$StandaloneRoot][$standalone] = $target
            if (-not (Test-Path -LiteralPath (Join-Path $target 'SKILL.md'))) {
                $missingSourceTargets += [pscustomobject]@{ name=$standalone; target=$target }
            }
        }
    }
    elseif ($ExpectedProfile -eq 'skill-development') {
        if ([string]::IsNullOrWhiteSpace($Skill)) {
            $missingExpectedSkills += [pscustomobject]@{ name=$null; root=$agentsRoot; reason='-Skill is required for skill-development audit' }
        } else {
            $target = Find-CanonicalSkillPath -Repository $sourceRepository -SkillName $Skill
            if ($target) { $expectedByRoot.agents[$Skill] = $target }
            else { $missingSourceTargets += [pscustomobject]@{ name=$Skill; target=$null } }
        }
    }

    foreach ($rootName in @('agents','codex')) {
        $rootPath = if ($rootName -eq 'agents') { $agentsRoot } else { $codexRoot }
        foreach ($expectedName in @($expectedByRoot[$rootName].Keys)) {
            $link = Join-Path $rootPath $expectedName
            if (-not (Test-Path -LiteralPath $link)) {
                $missingExpectedSkills += [pscustomobject]@{ name=$expectedName; root=$rootPath; target=$expectedByRoot[$rootName][$expectedName] }
                continue
            }
            $actualTarget = Get-LinkTarget -PathValue $link
            if (-not (Test-SamePath $actualTarget $expectedByRoot[$rootName][$expectedName])) {
                $mismatchedManagedLinks += [pscustomobject]@{ name=$expectedName; path=$link; expectedTarget=$expectedByRoot[$rootName][$expectedName]; actualTarget=$actualTarget }
            }
        }
        foreach ($registeredName in @($config.standaloneSkillRegistry.skills)) {
            $path = Join-Path $rootPath $registeredName
            if ((Test-Path -LiteralPath $path) -and -not $expectedByRoot[$rootName].ContainsKey($registeredName)) {
                $unexpectedRegisteredSkills += [pscustomobject]@{ name=$registeredName; path=$path; root=$rootName }
            }
        }
    }
}
$profileMatch = ($profileKnown -and ($missingExpectedSkills.Count -eq 0) -and ($unexpectedRegisteredSkills.Count -eq 0) -and ($mismatchedManagedLinks.Count -eq 0) -and ($missingSourceTargets.Count -eq 0))
$codexRootAllowed = $AllowCodexRoot -or ($ExpectedProfile -eq 'full' -and $StandaloneRoot -eq 'codex')
$ok = ($duplicateNames.Count -eq 0) -and ($brokenLinks.Count -eq 0) -and ($wholeRepositoryLinks.Count -eq 0) -and ($generatedSkillLinks.Count -eq 0) -and (-not $sourceRepositoryUnderSkillRoot) -and ($profileKnown) -and (-not $pluginPackageDrift)
if ($ExpectedProfile -and -not $profileMatch) { $ok = $false }
if (-not $codexRootAllowed -and $codexRootSkills -gt 0) { $ok = $false }

$result = [pscustomobject]@{
    ok = $ok
    expectedProfile = $ExpectedProfile
    profileKnown = $profileKnown
    profileMatch = $profileMatch
    standaloneRoot = $StandaloneRoot
    expectedSkill = $Skill
    activeSkills = @($entries | Where-Object { $_.name }).Count
    duplicateNames = $duplicateNames
    codexRoot = $codexRoot
    codexRootSkills = $codexRootSkills
    agentsRoot = $agentsRoot
    pluginCache = $pluginCache
    sourcePluginPath = $sourcePluginPath
    installedPluginPath = $installedPluginPath
    sourcePluginBuildInfo = $sourcePluginBuildInfo
    installedPluginBuildInfo = $installedPluginBuildInfo
    pluginPackageDrift = $pluginPackageDrift
    pluginDriftReasons = $pluginDriftReasons
    sourceRepository = $sourceRepository
    sourceRepositoryUnderSkillRoot = $sourceRepositoryUnderSkillRoot
    registeredStandaloneSkills = @($config.standaloneSkillRegistry.skills)
    registeredStandaloneSourcePaths = $config.standaloneSkillRegistry.sourcePaths
    brokenLinks = $brokenLinks
    wholeRepositoryLinks = $wholeRepositoryLinks
    generatedSkillLinks = $generatedSkillLinks
    missingExpectedSkills = $missingExpectedSkills
    unexpectedRegisteredSkills = $unexpectedRegisteredSkills
    mismatchedManagedLinks = $mismatchedManagedLinks
    missingSourceTargets = $missingSourceTargets
    entries = $entries
}

if ($Json) {
    $result | ConvertTo-Json -Depth 10
} else {
    $result | Format-List
}

if ($Strict -and -not $ok) { exit 1 }
