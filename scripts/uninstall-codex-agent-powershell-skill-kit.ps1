[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$RepoRoot = '.',
    [switch]$KeepState,
    [switch]$KeepPackage,
    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath $RepoRoot).Path
$runtimeMirror = Join-Path $repo '.agents\skills\codex-agent-powershell-skill-kit'
if (Test-Path -LiteralPath $runtimeMirror) {
    $marker = Join-Path $runtimeMirror '.runtime-mirror.json'
    if (-not (Test-Path -LiteralPath $marker)) {
        throw "Refusing to remove .agents/skills/codex-agent-powershell-skill-kit because runtime mirror marker is missing: $marker"
    }
}

$paths = @(
    '.agents/skills/codex-agent-powershell-skill-kit',
    'scripts/install-codex-agent-powershell-skill-kit.ps1',
    'scripts/status-codex-agent-powershell-skill-kit.ps1',
    'scripts/uninstall-codex-agent-powershell-skill-kit.ps1',
    'scripts/verify-codex-agent-powershell-skill-kit.ps1',
    'scripts/test-codex-agent-powershell-skill-kit.ps1'
)
if (-not $KeepPackage) {
    $paths += @(
        'config/codex-agent-powershell-skill-kit.json',
        'dist/codex-agent-powershell-skill-kit',
        'docs/codex-agent-powershell-skill-kit'
    )
}
foreach ($rel in $paths) {
    $full = Join-Path $repo $rel
    if (Test-Path -LiteralPath $full) {
        if ($DryRun) { Write-Host "DRYRUN remove $rel" }
        else { if ($PSCmdlet.ShouldProcess($full, 'Remove kit file or directory')) { Remove-Item -LiteralPath $full -Recurse -Force } }
    }
}

$marketplace = Join-Path $repo '.agents\plugins\marketplace.json'
if (Test-Path -LiteralPath $marketplace) {
    if ($DryRun) { Write-Host 'DRYRUN update marketplace' }
    else {
        Copy-Item -LiteralPath $marketplace -Destination "$marketplace.bak" -Force
        $data = Get-Content -LiteralPath $marketplace -Raw | ConvertFrom-Json
        if ($data.PSObject.Properties.Name -contains 'plugins') {
            $data.plugins = @($data.plugins | Where-Object { $_.name -ne 'codex-agent-powershell-skill-kit' })
            $data | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $marketplace -Encoding utf8
        }
    }
}

if (-not $KeepState) {
    $state = Join-Path $repo '.codex-agent-powershell-skill-kit'
    if (Test-Path -LiteralPath $state) {
        if ($DryRun) { Write-Host 'DRYRUN remove install state' } else { if ($PSCmdlet.ShouldProcess($state, 'Remove install state')) { Remove-Item -LiteralPath $state -Recurse -Force } }
    }
}

Write-Host "Uninstalled codex-agent-powershell-skill-kit from $repo"
