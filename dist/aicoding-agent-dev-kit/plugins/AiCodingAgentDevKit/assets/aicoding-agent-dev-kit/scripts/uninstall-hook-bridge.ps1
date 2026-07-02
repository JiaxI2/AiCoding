param(
    [string]$RepoRoot = ".",
    [string]$HookFile = "",
    [switch]$DryRun,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$detectJson = & "$PSScriptRoot\detect-existing-hooks.ps1" -RepoRoot $root -Json
$detect = $detectJson | ConvertFrom-Json

$targets = @()
if ($HookFile) {
    $targets += $HookFile
} else {
    foreach ($p in $detect.existingPreCommitHooks) { $targets += [string]$p }
}

$changed = @()
foreach ($t in $targets) {
    if (-not (Test-Path -LiteralPath $t)) { continue }
    $txt = Get-Content -Raw -LiteralPath $t
    $new = [regex]::Replace($txt, "(?ms)\n?# BEGIN AICODING_AGENT_DEV_KIT_BRIDGE.*?# END AICODING_AGENT_DEV_KIT_BRIDGE\n?", "`n")
    if ($new -ne $txt) {
        if (-not $DryRun) { $new | Set-Content -Encoding UTF8 -LiteralPath $t }
        $changed += $t
    }
}

Write-AgentDevKitJson -Json:$Json -Data @{
    ok=$true
    dryRun=[bool]$DryRun
    action="uninstall-hook-bridge"
    changed=$changed
}
