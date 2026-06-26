param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('runtime','skill-development','full')]
    [string]$Profile,
    [string]$Skill,
    [string]$SourceRepository,
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

$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$agentsRoot = Expand-RuntimePath $config.skillRuntime.canonicalUserRoot
if (-not $SourceRepository) { $SourceRepository = Expand-RuntimePath $config.skillRuntime.defaultSourceRepository }
$actions = @()
$warnings = @()

if ($Profile -eq 'skill-development' -and [string]::IsNullOrWhiteSpace($Skill)) {
    throw '-Skill is required for skill-development profile.'
}

$actions += 'Ensure user skill root exists: ' + $agentsRoot
if ($Profile -eq 'runtime') {
    $actions += 'Use AiCoding Plugin as the only aicoding-* runtime source.'
    $actions += 'Do not link canonical embedded/ or platform/ sources.'
    $actions += 'Run audit-runtime-skills.ps1 -ExpectedProfile runtime.'
}
elseif ($Profile -eq 'full') {
    $actions += 'Use AiCoding Plugin as the only aicoding-* runtime source.'
    foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
        $target = Join-Path $SourceRepository $standalone
        $link = Join-Path $agentsRoot $standalone
        $actions += "Ensure standalone skill link: $link -> $target"
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
    if ($Profile -eq 'full') {
        foreach ($standalone in @($config.profiles.full.standaloneSkills)) {
            $target = Join-Path $SourceRepository $standalone
            $link = Join-Path $agentsRoot $standalone
            if (-not (Test-Path -LiteralPath $target)) { throw "Missing target: $target" }
            if (Test-Path -LiteralPath $link) { throw "Refusing to overwrite existing path: $link" }
            New-Item -ItemType Junction -Path $link -Target $target | Out-Null
        }
    }
    elseif ($Profile -eq 'skill-development') {
        $target = Find-CanonicalSkillPath -Repository $SourceRepository -SkillName $Skill
        if (-not $target) { throw "Canonical skill not found: $Skill" }
        $link = Join-Path $agentsRoot $Skill
        if (Test-Path -LiteralPath $link) { throw "Refusing to overwrite existing path: $link" }
        New-Item -ItemType Junction -Path $link -Target $target | Out-Null
    }
}

$result = [pscustomobject]@{
    profile = $Profile
    dryRun = [bool]$DryRun
    sourceRepository = $SourceRepository
    agentsRoot = $agentsRoot
    actions = $actions
    warnings = $warnings
}
if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }

