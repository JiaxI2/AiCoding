param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$detectJson = & "$PSScriptRoot\detect-existing-hooks.ps1" -RepoRoot $root -Json
$detect = $detectJson | ConvertFrom-Json
Write-AgentDevKitJson -Json:$Json -Data @{
    ok=[bool]$detect.hasAgentDevKitBridge
    hasExistingHook=[bool]$detect.hasExistingHook
    hasAgentDevKitBridge=[bool]$detect.hasAgentDevKitBridge
    existingPreCommitHooks=$detect.existingPreCommitHooks
    coreHooksPath=$detect.coreHooksPath
}
if (-not $detect.hasAgentDevKitBridge) { exit 1 }
