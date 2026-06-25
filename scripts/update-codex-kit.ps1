param([switch]$DryRun)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$submodule = Resolve-KitPath $repo $config.agents.skillsSubmodule
if ($DryRun) {
    Write-Host "Would run: git -C $submodule fetch origin main"
    Write-Host "Would run: git -C $submodule pull --ff-only origin main"
    Write-Host "Would run: scripts/verify-codex-kit.ps1"
    Write-Host 'Would refresh/reinstall plugin through Codex plugin surfaces if available.'
    exit 0
}
& git -C $submodule fetch origin main
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& git -C $submodule checkout main
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& git -C $submodule pull --ff-only origin main
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot 'verify-codex-kit.ps1')