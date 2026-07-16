param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('runtime','skill-development','full')]
    [string]$Profile,
    [string]$Skill,
    [string]$SourceRepository,
    [ValidateSet('agents','codex')]
    [string]$StandaloneRoot = 'agents',
    [switch]$MigrateUnmanaged,
    [switch]$DryRun,
    [switch]$Json
)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force

function Expand-RuntimePath {
    param([string]$PathValue)
    return Resolve-CodexKitRuntimePath -RepoRoot $repo -PathValue $PathValue
}

function Resolve-StandaloneRoot {
    param($Config, [string]$RootName)
    if ($RootName -eq 'codex') { return Expand-RuntimePath $Config.skillRuntime.codexUserRoot }
    return Expand-RuntimePath $Config.skillRuntime.canonicalUserRoot
}

function Resolve-OtherStandaloneRoot {
    param($Config, [string]$RootName)
    if ($RootName -eq 'codex') { return Expand-RuntimePath $Config.skillRuntime.canonicalUserRoot }
    return Expand-RuntimePath $Config.skillRuntime.codexUserRoot
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
        $text = Get-Content -LiteralPath $skillFile.FullName -Raw
        $pattern = '(?m)^name:\s*[''\"]?' + [regex]::Escape($SkillName) + '[''\"]?\s*$'
        if ($text -match $pattern) {
            return Split-Path -Parent $skillFile.FullName
        }
    }
    return $null
}

function Test-PathUnder {
    param([string]$PathValue, [string]$RootValue)
    try {
        $path = [System.IO.Path]::GetFullPath($PathValue).TrimEnd('\') + '\'
        $root = [System.IO.Path]::GetFullPath($RootValue).TrimEnd('\') + '\'
        return $path.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)
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

function Backup-ExistingPath {
    param([string]$PathValue, [string]$ManagedRoot, [string]$RootName)
    if (-not $MigrateUnmanaged) { throw "Refusing to replace unmanaged path without -MigrateUnmanaged: $PathValue" }
    if (-not (Test-PathUnder -PathValue $PathValue -RootValue $ManagedRoot)) {
        throw "Backup source escaped managed Skill root: $PathValue"
    }
    if (-not (Test-PathUnder -PathValue $backupRoot -RootValue $backupBase)) {
        throw "Backup target escaped approved backup root: $backupRoot"
    }
    $destinationRoot = Join-Path $backupRoot $RootName
    New-Item -ItemType Directory -Force -Path $destinationRoot | Out-Null
    $destination = Join-Path $destinationRoot (Split-Path -Leaf $PathValue)
    if (Test-Path -LiteralPath $destination) { throw "Backup destination already exists: $destination" }
    Move-Item -LiteralPath $PathValue -Destination $destination
    return $destination
}

function Assert-ExistingPathManageable {
    param([string]$PathValue, [string]$Target)
    if (-not (Test-Path -LiteralPath $PathValue)) { return }
    $item = Get-Item -LiteralPath $PathValue -Force
    $actualTarget = if ($item.LinkType) { @($item.Target) | Select-Object -First 1 } else { $null }
    if ($item.LinkType -and (Test-SamePath -Left $actualTarget -Right $Target)) { return }
    if (-not $MigrateUnmanaged) { throw "Existing registered path requires -MigrateUnmanaged: $PathValue" }
}

function Ensure-Junction {
    param([string]$Link, [string]$Target, [string]$ManagedRoot, [string]$RootName)
    if (-not (Test-Path -LiteralPath $Target)) { throw "Missing target: $Target" }
    if (Test-Path -LiteralPath $Link) {
        $item = Get-Item -LiteralPath $Link -Force
        $actualTarget = if ($item.LinkType) { @($item.Target) | Select-Object -First 1 } else { $null }
        if ($item.LinkType -and (Test-SamePath -Left $actualTarget -Right $Target)) {
            return [pscustomobject]@{ result='exists'; backedUpTo=$null }
        }
        $backup = Backup-ExistingPath -PathValue $Link -ManagedRoot $ManagedRoot -RootName $RootName
    } else {
        $backup = $null
    }
    New-Item -ItemType Junction -Path $Link -Target $Target | Out-Null
    return [pscustomobject]@{ result='created'; backedUpTo=$backup }
}

function Remove-RegisteredPath {
    param([string]$Link, [string]$Target, [string]$ManagedRoot, [string]$RootName)
    if (-not (Test-Path -LiteralPath $Link)) { return [pscustomobject]@{ result='absent'; backedUpTo=$null } }
    $item = Get-Item -LiteralPath $Link -Force
    $actualTarget = if ($item.LinkType) { @($item.Target) | Select-Object -First 1 } else { $null }
    if ($item.LinkType -and (Test-SamePath -Left $actualTarget -Right $Target)) {
        [System.IO.Directory]::Delete($Link)
        return [pscustomobject]@{ result='removed'; backedUpTo=$null }
    }
    $backup = Backup-ExistingPath -PathValue $Link -ManagedRoot $ManagedRoot -RootName $RootName
    return [pscustomobject]@{ result='backed-up'; backedUpTo=$backup }
}

$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$agentsRoot = Expand-RuntimePath $config.skillRuntime.canonicalUserRoot
$codexRoot = Expand-RuntimePath $config.skillRuntime.codexUserRoot
$standaloneInstallRoot = Resolve-StandaloneRoot $config $StandaloneRoot
$otherStandaloneRoot = Resolve-OtherStandaloneRoot $config $StandaloneRoot
$otherStandaloneRootName = if ($StandaloneRoot -eq 'agents') { 'codex' } else { 'agents' }
if (-not $SourceRepository) { $SourceRepository = Resolve-CodexKitConfiguredPath -ConfigSection $config.skillRuntime -RepoRoot $repo }
$SourceRepository = [System.IO.Path]::GetFullPath($SourceRepository)
$backupBase = Join-Path $env:USERPROFILE '.codex\backups\aicoding-skill-profiles'
$backupRoot = Join-Path $backupBase ([DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss-fff'))
$actions = @()
$warnings = @()
$changes = @()
$rollbackFile = $null

if ($Profile -eq 'skill-development' -and [string]::IsNullOrWhiteSpace($Skill)) {
    throw '-Skill is required for skill-development profile.'
}

$actions += 'Ensure canonical user skill root exists: ' + $agentsRoot
$actions += 'Standalone install root: ' + $standaloneInstallRoot
$actions += 'Other supported standalone root: ' + $otherStandaloneRoot
if ($MigrateUnmanaged) { $actions += 'Back up unmanaged or mismatched registered paths under: ' + $backupRoot }
if ($Profile -eq 'runtime') {
    $actions += 'Use AiCoding Plugin as the only aicoding-* runtime source.'
    $actions += 'Do not link canonical embedded/ or platform/ sources.'
    $actions += 'Do not install standalone skills unless Profile full is selected.'
    $removeNames = if ($Skill) { @($Skill) } else { @($config.profiles.full.standaloneSkills) }
    foreach ($standalone in $removeNames) {
        if (@($config.standaloneSkillRegistry.skills) -notcontains $standalone) { throw "Standalone Skill is not registered: $standalone" }
        $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
        $target = Join-Path $SourceRepository $sourcePath
        foreach ($rootPath in @($agentsRoot, $codexRoot)) {
            $link = Join-Path $rootPath $standalone
            $actions += "Remove registered standalone skill path if present: $link -> $target"
        }
    }
    $actions += 'Run audit-runtime-skills.ps1 -ExpectedProfile runtime.'
}
elseif ($Profile -eq 'full') {
    $actions += 'Use AiCoding Plugin as the only aicoding-* runtime source.'
    foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
        $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
        $target = Join-Path $SourceRepository $sourcePath
        $link = Join-Path $standaloneInstallRoot $standalone
        $actions += "Ensure standalone skill link: $link -> $target (source: $sourcePath)"
        $actions += "Remove duplicate registered path from other root: $(Join-Path $otherStandaloneRoot $standalone)"
        if (-not (Test-Path -LiteralPath $target)) { $warnings += "Standalone skill target missing: $target" }
    }
    $actions += 'Run audit-runtime-skills.ps1 -ExpectedProfile full.'
}
else {
    $target = Find-CanonicalSkillPath -Repository $SourceRepository -SkillName $Skill
    $link = Join-Path $agentsRoot $Skill
    $actions += 'Disable AiCoding Plugin before exposing same-name canonical source.'
    $actions += "Ensure development skill link: $link -> $target"
    $actions += "Remove same-name development path from Codex compatibility root: $(Join-Path $codexRoot $Skill)"
    if (-not $target) { $warnings += "Canonical skill not found in source repository: $Skill" }
    $actions += 'Run audit-runtime-skills.ps1 -ExpectedProfile skill-development -Strict.'
}

if (-not $DryRun) {
    if ($Profile -eq 'runtime') {
        $removeNames = if ($Skill) { @($Skill) } else { @($config.profiles.full.standaloneSkills) }
        foreach ($standalone in $removeNames) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = Join-Path $SourceRepository $sourcePath
            Assert-ExistingPathManageable -PathValue (Join-Path $agentsRoot $standalone) -Target $target
            Assert-ExistingPathManageable -PathValue (Join-Path $codexRoot $standalone) -Target $target
        }
    }
    elseif ($Profile -eq 'full') {
        foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = Join-Path $SourceRepository $sourcePath
            if (-not (Test-Path -LiteralPath (Join-Path $target 'SKILL.md'))) { throw "Missing standalone Skill target: $target" }
            Assert-ExistingPathManageable -PathValue (Join-Path $standaloneInstallRoot $standalone) -Target $target
            Assert-ExistingPathManageable -PathValue (Join-Path $otherStandaloneRoot $standalone) -Target $target
        }
    }
    elseif ($Profile -eq 'skill-development') {
        $developmentTarget = Find-CanonicalSkillPath -Repository $SourceRepository -SkillName $Skill
        if (-not $developmentTarget) { throw "Canonical skill not found: $Skill" }
        Assert-ExistingPathManageable -PathValue (Join-Path $agentsRoot $Skill) -Target $developmentTarget
        Assert-ExistingPathManageable -PathValue (Join-Path $codexRoot $Skill) -Target $developmentTarget
    }

    New-Item -ItemType Directory -Force -Path $agentsRoot | Out-Null
    New-Item -ItemType Directory -Force -Path $standaloneInstallRoot | Out-Null
    if ($Profile -eq 'runtime') {
        $removeNames = if ($Skill) { @($Skill) } else { @($config.profiles.full.standaloneSkills) }
        foreach ($standalone in $removeNames) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = Join-Path $SourceRepository $sourcePath
            foreach ($rootInfo in @(
                [pscustomobject]@{ name='agents'; path=$agentsRoot },
                [pscustomobject]@{ name='codex'; path=$codexRoot }
            )) {
                $link = Join-Path $rootInfo.path $standalone
                $operation = Remove-RegisteredPath -Link $link -Target $target -ManagedRoot $rootInfo.path -RootName $rootInfo.name
                $changes += [pscustomobject]@{ name=$standalone; sourcePath=$sourcePath; link=$link; target=$target; result=$operation.result; backedUpTo=$operation.backedUpTo }
            }
        }
    }
    elseif ($Profile -eq 'full') {
        foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = Join-Path $SourceRepository $sourcePath
            $link = Join-Path $standaloneInstallRoot $standalone
            $operation = Ensure-Junction -Link $link -Target $target -ManagedRoot $standaloneInstallRoot -RootName $StandaloneRoot
            $changes += [pscustomobject]@{ name=$standalone; sourcePath=$sourcePath; link=$link; target=$target; result=$operation.result; backedUpTo=$operation.backedUpTo }
            $otherLink = Join-Path $otherStandaloneRoot $standalone
            $otherOperation = Remove-RegisteredPath -Link $otherLink -Target $target -ManagedRoot $otherStandaloneRoot -RootName $otherStandaloneRootName
            $changes += [pscustomobject]@{ name=$standalone; sourcePath=$sourcePath; link=$otherLink; target=$target; result=$otherOperation.result; backedUpTo=$otherOperation.backedUpTo }
        }
    }
    elseif ($Profile -eq 'skill-development') {
        $target = Find-CanonicalSkillPath -Repository $SourceRepository -SkillName $Skill
        if (-not $target) { throw "Canonical skill not found: $Skill" }
        $link = Join-Path $agentsRoot $Skill
        $operation = Ensure-Junction -Link $link -Target $target -ManagedRoot $agentsRoot -RootName 'agents'
        $changes += [pscustomobject]@{ name=$Skill; link=$link; target=$target; result=$operation.result; backedUpTo=$operation.backedUpTo }
        $codexLink = Join-Path $codexRoot $Skill
        $codexOperation = Remove-RegisteredPath -Link $codexLink -Target $target -ManagedRoot $codexRoot -RootName 'codex'
        $changes += [pscustomobject]@{ name=$Skill; link=$codexLink; target=$target; result=$codexOperation.result; backedUpTo=$codexOperation.backedUpTo }
    }

    $backedUp = @($changes | Where-Object { $_.backedUpTo })
    if ($backedUp.Count -gt 0) {
        New-Item -ItemType Directory -Force -Path $backupRoot | Out-Null
        $rollbackFile = Join-Path $backupRoot 'rollback.json'
        $rollback = [pscustomobject]@{
            schemaVersion = 1
            createdAtUtc = [DateTime]::UtcNow.ToString('o')
            profile = $Profile
            sourceRepository = $SourceRepository
            entries = $backedUp
        }
        $rollbackText = $rollback | ConvertTo-Json -Depth 8
        [System.IO.File]::WriteAllText($rollbackFile, $rollbackText, (New-Object System.Text.UTF8Encoding $false))
    }
}

$result = [pscustomobject]@{
    profile = $Profile
    dryRun = [bool]$DryRun
    sourceRepository = $SourceRepository
    agentsRoot = $agentsRoot
    codexRoot = $codexRoot
    standaloneRoot = $StandaloneRoot
    standaloneInstallRoot = $standaloneInstallRoot
    otherStandaloneRoot = $otherStandaloneRoot
    migrateUnmanaged = [bool]$MigrateUnmanaged
    backupRoot = if (@($changes | Where-Object { $_.backedUpTo }).Count -gt 0) { $backupRoot } else { $null }
    rollbackFile = $rollbackFile
    actions = $actions
    warnings = $warnings
    changes = $changes
}
if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
