param(
    [string]$RepoRoot = ".",
    [ValidateSet("pre-commit","ci","all","release")]
    [string]$Mode = "all",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot

$steps = @(
    "validate-spec-pack.ps1",
    "validate-implementation-plan.ps1",
    "validate-tdd.ps1",
    "validate-memory.ps1",
    "validate-adr.ps1",
    "validate-bdd.ps1",
    "validate-traceability.ps1",
    "validate-solution-doc-sync.ps1"
)

if ($Mode -ne "pre-commit") {
    $steps += "validate-desensitization.ps1"
}

$failed = @()
foreach ($s in $steps) {
    $global:LASTEXITCODE = 0
    & "$PSScriptRoot\$s" -RepoRoot $root -Json | Out-Null
    if ((-not $?) -or $LASTEXITCODE -ne 0) { $failed += $s }
}

# Optional DocSync bridge. Orchestrate, do not replace.
$docSync = Join-Path $root "scripts/check-documentation-sync.ps1"
$docSyncRan = $false
if (Test-Path -LiteralPath $docSync) {
    $docSyncRan = $true
    $global:LASTEXITCODE = 0
    if ($Mode -eq "pre-commit") {
        & $docSync -Mode pre-commit 2>$null | Out-Null
    } else {
        & $docSync -Mode all 2>$null | Out-Null
    }
    if ((-not $?) -or $LASTEXITCODE -ne 0) { $failed += "check-documentation-sync.ps1" }
}

# Optional Git governance bridge.
$gitGov = Join-Path $root "bin/aicoding.exe"
$gitGovRan = $false
if (Test-Path -LiteralPath $gitGov) {
    $gitGovRan = $true
    $global:LASTEXITCODE = 0
    & $gitGov governance lint --json 2>$null | Out-Null
    if ((-not $?) -or $LASTEXITCODE -ne 0) { $failed += "aicoding governance lint" }
}

Write-AgentDevKitJson -Json:$Json -Data @{
  ok = ($failed.Count -eq 0)
  mode = $Mode
  failed = $failed
  optimized = $true
  docSyncBridge = $docSyncRan
  gitGovernanceBridge = $gitGovRan
}
if ($failed.Count -gt 0) { exit 1 }