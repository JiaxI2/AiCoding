[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$RepoRoot = '.',
    [switch]$InstallMissingTools,
    [switch]$DryRun,
    [switch]$NoRuntimeMirror
)

$ErrorActionPreference = 'Stop'

function Resolve-RepoRoot {
    param([string]$InputPath)
    $resolved = Resolve-Path -LiteralPath $InputPath -ErrorAction Stop
    Push-Location $resolved.Path
    try {
        $gitRoot = git rev-parse --show-toplevel 2>$null
        if ($LASTEXITCODE -eq 0 -and $gitRoot) { return (Resolve-Path -LiteralPath $gitRoot).Path }
        return $resolved.Path
    } finally { Pop-Location }
}

function Test-SamePath {
    param([string]$Left, [string]$Right)
    try {
        $leftPath = (Resolve-Path -LiteralPath $Left -ErrorAction Stop).Path
        $rightResolved = Resolve-Path -LiteralPath $Right -ErrorAction SilentlyContinue
        if ($null -eq $rightResolved) { return $false }
        return [string]::Equals($leftPath, $rightResolved.Path, [System.StringComparison]::OrdinalIgnoreCase)
    } catch {
        return $false
    }
}

function Copy-DirectorySafe {
    param([string]$Source, [string]$Destination)
    if (-not (Test-Path -LiteralPath $Source)) { throw "Source directory not found: $Source" }
    if (Test-SamePath -Left $Source -Right $Destination) { Write-Host "Skip copy; source and destination are the same: $Source"; return }
    if ($DryRun) { Write-Host "DRYRUN copy $Source -> $Destination"; return }
    if (Test-Path -LiteralPath $Destination) {
        if ($PSCmdlet.ShouldProcess($Destination, 'Remove existing directory before copy')) {
            Remove-Item -LiteralPath $Destination -Recurse -Force
        }
    }
    New-Item -ItemType Directory -Path (Split-Path -Parent $Destination) -Force | Out-Null
    Copy-Item -LiteralPath $Source -Destination $Destination -Recurse -Force
}

function Write-JsonFile {
    param([string]$Path, [object]$Object)
    if ($DryRun) { Write-Host "DRYRUN write $Path"; return }
    New-Item -ItemType Directory -Path (Split-Path -Parent $Path) -Force | Out-Null
    $Object | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $Path -Encoding utf8
}

function Merge-MarketplaceEntry {
    param([string]$MarketplacePath, [object]$Entry)
    if ($DryRun) { Write-Host "DRYRUN merge marketplace $MarketplacePath"; return }
    New-Item -ItemType Directory -Path (Split-Path -Parent $MarketplacePath) -Force | Out-Null
    if (Test-Path -LiteralPath $MarketplacePath) {
        Copy-Item -LiteralPath $MarketplacePath -Destination "$MarketplacePath.bak" -Force
        $raw = Get-Content -LiteralPath $MarketplacePath -Raw
        if ([string]::IsNullOrWhiteSpace($raw)) { $data = [pscustomobject]@{ plugins = @() } }
        else { $data = $raw | ConvertFrom-Json }
    } else {
        $data = [pscustomobject]@{ plugins = @() }
    }

    if (-not ($data.PSObject.Properties.Name -contains 'plugins')) {
        $data | Add-Member -NotePropertyName plugins -NotePropertyValue @()
    }
    $plugins = @($data.plugins | Where-Object { $_.name -ne $Entry.name })
    $plugins += $Entry
    $data.plugins = $plugins
    $data | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $MarketplacePath -Encoding utf8
}

function Write-RuntimeMirrorNotice {
    param([string]$SkillDest, [string]$RepoRoot)
    if ($DryRun) { return }
    $notice = @'
# Runtime Mirror Notice

This directory is a generated repo-scoped runtime mirror for `codex-agent-powershell-skill-kit`.

AiCoding does not own this directory as canonical skill source.

Authoritative package payload:

```text
dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/skills/codex-agent-powershell-skill-kit/
```

Regenerate this mirror by running:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-codex-agent-powershell-skill-kit.ps1
```

Do not manually edit this mirror. Update the canonical kit/package, rebuild, and reinstall.
'@
    Set-Content -LiteralPath (Join-Path $SkillDest 'RUNTIME_MIRROR_NOTICE.md') -Value $notice -Encoding utf8
    $meta = [pscustomobject]@{
        name = 'codex-agent-powershell-skill-kit'
        version = '1.3.0'
        role = 'generated-runtime-mirror'
        canonicalOwnedByAiCoding = $false
        authoritativePackagePath = 'dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/skills/codex-agent-powershell-skill-kit'
        generatedAt = (Get-Date).ToString('o')
        repoRoot = $RepoRoot
    }
    $meta | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath (Join-Path $SkillDest '.runtime-mirror.json') -Encoding utf8
}

$repo = Resolve-RepoRoot -InputPath $RepoRoot
$overlayRoot = Split-Path -Parent $PSScriptRoot

$configSource = Join-Path $overlayRoot 'config\codex-agent-powershell-skill-kit.json'
$configDest = Join-Path $repo 'config\codex-agent-powershell-skill-kit.json'
if (-not $DryRun) {
    if (-not (Test-SamePath -Left $configSource -Right $configDest)) {
        New-Item -ItemType Directory -Path (Split-Path -Parent $configDest) -Force | Out-Null
        Copy-Item -LiteralPath $configSource -Destination $configDest -Force
    }
} else { Write-Host "DRYRUN copy config" }

$docsSource = Join-Path $overlayRoot 'docs\codex-agent-powershell-skill-kit'
$docsDest = Join-Path $repo 'docs\codex-agent-powershell-skill-kit'
Copy-DirectorySafe -Source $docsSource -Destination $docsDest

$distSource = Join-Path $overlayRoot 'dist\codex-agent-powershell-skill-kit'
$distDest = Join-Path $repo 'dist\codex-agent-powershell-skill-kit'
Copy-DirectorySafe -Source $distSource -Destination $distDest

$scripts = @(
    'install-codex-agent-powershell-skill-kit.ps1',
    'status-codex-agent-powershell-skill-kit.ps1',
    'uninstall-codex-agent-powershell-skill-kit.ps1',
    'verify-codex-agent-powershell-skill-kit.ps1',
    'test-codex-agent-powershell-skill-kit.ps1'
)
foreach ($script in $scripts) {
    $src = Join-Path $overlayRoot "scripts\$script"
    $dst = Join-Path $repo "scripts\$script"
    if (-not $DryRun) {
        if (-not (Test-SamePath -Left $src -Right $dst)) {
            New-Item -ItemType Directory -Path (Split-Path -Parent $dst) -Force | Out-Null
            Copy-Item -LiteralPath $src -Destination $dst -Force
        }
    } else { Write-Host "DRYRUN copy $script" }
}

$entryPath = Join-Path $overlayRoot '.agents\plugins\codex-agent-powershell-skill-kit.marketplace.json'
if (Test-Path -LiteralPath $entryPath) {
    $entry = Get-Content -LiteralPath $entryPath -Raw | ConvertFrom-Json
} else {
    $config = Get-Content -LiteralPath $configDest -Raw | ConvertFrom-Json
    $entry = [pscustomobject]@{
        name = $config.name
        displayName = 'Codex Agent PowerShell Skill Kit'
        version = $config.version
        description = 'PS7-first PowerShell AST, safe rewrite, and PSScriptAnalyzer gate for Codex/AiCoding agents.'
        path = $config.distPluginPath
        skillPath = $config.skillPath
        entrypoint = '.codex-plugin/plugin.json'
        tags = @('powershell', 'codex', 'aicoding', 'agent-guard', 'psscriptanalyzer', 'ast')
        skillPathRole = $config.sourceOwnership.repoScopedSkillRole
        packagedSkillPath = $config.packagedSkillPath
    }
}
Merge-MarketplaceEntry -MarketplacePath (Join-Path $repo '.agents\plugins\marketplace.json') -Entry $entry

if (-not $NoRuntimeMirror) {
    if ($DryRun) {
        $packagedSkillSource = Join-Path $overlayRoot 'dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\skills\codex-agent-powershell-skill-kit'
    } else {
        $packagedSkillSource = Join-Path $repo 'dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\skills\codex-agent-powershell-skill-kit'
    }
    $skillDest = Join-Path $repo '.agents\skills\codex-agent-powershell-skill-kit'
    Copy-DirectorySafe -Source $packagedSkillSource -Destination $skillDest
    Write-RuntimeMirrorNotice -SkillDest $skillDest -RepoRoot $repo
}

$state = [pscustomobject]@{
    name = 'codex-agent-powershell-skill-kit'
    version = '1.3.0'
    installedAt = (Get-Date).ToString('o')
    repoRoot = $repo
    runtime = 'pwsh'
    installMissingTools = [bool]$InstallMissingTools
    sourceOwnership = [pscustomobject]@{
        aicodingOwnsCanonicalSource = $false
        repoScopedSkillRole = 'generated-runtime-mirror'
        materialized = -not [bool]$NoRuntimeMirror
        authoritativePackagePath = 'dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/skills/codex-agent-powershell-skill-kit'
    }
    files = @(
        'config/codex-agent-powershell-skill-kit.json',
        'dist/codex-agent-powershell-skill-kit',
        'docs/codex-agent-powershell-skill-kit',
        'scripts/*codex-agent-powershell-skill-kit.ps1',
        '.agents/plugins/marketplace.json',
        '.agents/skills/codex-agent-powershell-skill-kit (generated runtime mirror)'
    )
}
Write-JsonFile -Path (Join-Path $repo '.codex-agent-powershell-skill-kit\install-state.json') -Object $state

if ($InstallMissingTools -and -not $DryRun) {
    $pssa = Get-Module -ListAvailable -Name PSScriptAnalyzer | Select-Object -First 1
    if (-not $pssa) { Install-Module -Name PSScriptAnalyzer -Scope CurrentUser -Force -AllowClobber }
}

Write-Host "Installed codex-agent-powershell-skill-kit v1.3.0 into $repo"
Write-Host "Repo-scoped skill is a generated runtime mirror, not AiCoding-owned canonical source."
