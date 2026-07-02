param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot

$coreHooksPath = ""
try { $coreHooksPath = (git -C $root config --get core.hooksPath 2>$null) } catch {}

$hookDirs = @()
if ($coreHooksPath) {
    $hookDirs += (Join-Path $root $coreHooksPath)
}
$hookDirs += (Join-Path $root ".githooks")
$hookDirs += (Join-Path $root ".git/hooks")

$preCommitCandidates = @()
foreach ($d in $hookDirs | Select-Object -Unique) {
    $preCommitCandidates += (Join-Path $d "pre-commit")
    $preCommitCandidates += (Join-Path $d "pre-commit.ps1")
}

$existing = @()
foreach ($p in $preCommitCandidates | Select-Object -Unique) {
    if (Test-Path -LiteralPath $p) { $existing += $p }
}

$hasBridge = $false
foreach ($p in $existing) {
    $txt = Get-Content -Raw -LiteralPath $p -ErrorAction SilentlyContinue
    if ($txt -match "BEGIN AICODING_AGENT_DEV_KIT_BRIDGE") { $hasBridge = $true }
}

Write-AgentDevKitJson -Json:$Json -Data @{
    ok=$true
    repoRoot=$root
    coreHooksPath=$coreHooksPath
    existingPreCommitHooks=$existing
    hasExistingHook=($existing.Count -gt 0)
    hasAgentDevKitBridge=$hasBridge
    recommendation= if ($existing.Count -gt 0) { "merge bridge into existing hook; do not overwrite" } else { "create hook only if explicitly requested" }
}
