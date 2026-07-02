param(
    [string]$RepoRoot = ".",
    [switch]$Purge,
    [switch]$Force,
    [switch]$DryRun,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$statePath = Join-Path $root ".agent-dev-kit/install-state.json"
$removed = New-Object System.Collections.Generic.List[string]
if (Test-Path -LiteralPath $statePath) {
    $state = Get-Content -Raw -LiteralPath $statePath | ConvertFrom-Json
    foreach ($f in $state.ownedFiles) {
        $target = Join-Path $root $f
        if (Test-Path -LiteralPath $target) {
            if (-not $DryRun) { Remove-Item -LiteralPath $target -Recurse -Force }
            $removed.Add($f)
        }
    }
}
if ($Purge -and $Force) {
    foreach ($p in @("spec",".agent-memory","docs/adr","specs/bdd","specs/tdd","docs/traceability")) {
        $target = Join-Path $root $p
        if (Test-Path -LiteralPath $target) {
            if (-not $DryRun) { Remove-Item -LiteralPath $target -Recurse -Force }
            $removed.Add($p)
        }
    }
}
if (-not $DryRun) {
    if (Test-Path -LiteralPath $statePath) { Remove-Item -LiteralPath $statePath -Force }
    $hooksPath = git -C $root config --get core.hooksPath 2>$null
    if ($hooksPath -eq ".githooks") { git -C $root config --unset core.hooksPath | Out-Null }
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok = $true; action = "uninstall"; purge = [bool]$Purge; force = [bool]$Force; removed = $removed.ToArray() }
