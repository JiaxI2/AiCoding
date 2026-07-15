param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('runtime','skill-development','full')]
    [string]$Profile,
    [string]$Skill,
    [string]$SourceRepository,
    [ValidateSet('agents','codex')]
    [string]$StandaloneRoot = 'agents',
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

function Ensure-Junction {
    param([string]$Link, [string]$Target)
    if (-not (Test-Path -LiteralPath $Target)) { throw "Missing target: $Target" }
    if (Test-Path -LiteralPath $Link) {
        $item = Get-Item -LiteralPath $Link -Force
        $targets = @($item.Target)
        if ($item.LinkType -and ($targets -contains $Target)) { return 'exists' }
        throw "Refusing to overwrite existing path: $Link"
    }
    New-Item -ItemType Junction -Path $Link -Target $Target | Out-Null
    return 'created'
}

function Remove-ManagedJunction {
    param([string]$Link, [string]$Target)
    if (-not (Test-Path -LiteralPath $Link)) { return 'absent' }
    $item = Get-Item -LiteralPath $Link -Force
    $targets = @($item.Target)
    if (-not $item.LinkType -or -not ($targets -contains $Target)) {
        throw "Refusing to remove unmanaged path: $Link"
    }
    Remove-Item -LiteralPath $Link -Force
    return 'removed'
}

$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$agentsRoot = Expand-RuntimePath $config.skillRuntime.canonicalUserRoot
$codexRoot = Expand-RuntimePath $config.skillRuntime.codexUserRoot
$standaloneInstallRoot = Resolve-StandaloneRoot $config $StandaloneRoot
if (-not $SourceRepository) { $SourceRepository = Resolve-CodexKitConfiguredPath -ConfigSection $config.skillRuntime -RepoRoot $repo }
$actions = @()
$warnings = @()
$changes = @()

if ($Profile -eq 'skill-development' -and [string]::IsNullOrWhiteSpace($Skill)) {
    throw '-Skill is required for skill-development profile.'
}

$actions += 'Ensure canonical user skill root exists: ' + $agentsRoot
$actions += 'Standalone install root: ' + $standaloneInstallRoot
if ($Profile -eq 'runtime') {
    $actions += 'Use AiCoding Plugin as the only aicoding-* runtime source.'
    $actions += 'Do not link canonical embedded/ or platform/ sources.'
    $actions += 'Do not install standalone skills unless Profile full is selected.'
    $removeNames = if ($Skill) { @($Skill) } else { @($config.profiles.full.standaloneSkills) }
    foreach ($standalone in $removeNames) {
        if (@($config.standaloneSkillRegistry.skills) -notcontains $standalone) { throw "Standalone Skill is not registered: $standalone" }
        $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
        $target = Join-Path $SourceRepository $sourcePath
        $link = Join-Path $standaloneInstallRoot $standalone
        $actions += "Remove managed standalone skill link if present: $link -> $target"
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
        if (-not (Test-Path -LiteralPath $target)) { $warnings += "Standalone skill target missing: $target" }
    }
    $actions += 'Run audit-runtime-skills.ps1 -ExpectedProfile full.'
}
else {
    $target = Find-CanonicalSkillPath -Repository $SourceRepository -SkillName $Skill
    $link = Join-Path $agentsRoot $Skill
    $actions += 'Disable AiCoding Plugin before exposing same-name canonical source.'
    $actions += "Ensure development skill link: $link -> $target"
    if (-not $target) { $warnings += "Canonical skill not found in source repository: $Skill" }
    $actions += 'Run audit-runtime-skills.ps1 -ExpectedProfile skill-development -Strict.'
}

if (-not $DryRun) {
    New-Item -ItemType Directory -Force -Path $agentsRoot | Out-Null
    New-Item -ItemType Directory -Force -Path $standaloneInstallRoot | Out-Null
    if ($Profile -eq 'runtime') {
        $removeNames = if ($Skill) { @($Skill) } else { @($config.profiles.full.standaloneSkills) }
        foreach ($standalone in $removeNames) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = Join-Path $SourceRepository $sourcePath
            $link = Join-Path $standaloneInstallRoot $standalone
            $result = Remove-ManagedJunction -Link $link -Target $target
            $changes += [pscustomobject]@{ name=$standalone; sourcePath=$sourcePath; link=$link; target=$target; result=$result }
        }
    }
    elseif ($Profile -eq 'full') {
        foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
            $sourcePath = Resolve-StandaloneSkillSourcePath -Config $config -SkillName $standalone
            $target = Join-Path $SourceRepository $sourcePath
            $link = Join-Path $standaloneInstallRoot $standalone
            $result = Ensure-Junction -Link $link -Target $target
            $changes += [pscustomobject]@{ name=$standalone; sourcePath=$sourcePath; link=$link; target=$target; result=$result }
        }
    }
    elseif ($Profile -eq 'skill-development') {
        $target = Find-CanonicalSkillPath -Repository $SourceRepository -SkillName $Skill
        if (-not $target) { throw "Canonical skill not found: $Skill" }
        $link = Join-Path $agentsRoot $Skill
        $result = Ensure-Junction -Link $link -Target $target
        $changes += [pscustomobject]@{ name=$Skill; link=$link; target=$target; result=$result }
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
    actions = $actions
    warnings = $warnings
    changes = $changes
}
if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
