param([string]$RepoRoot = ".", [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
Write-AgentDevKitJson -Json:$Json -Data @{ ok = $true; validator = "validate-traceability"; mode = "mvp-structural"; repoRoot = $root }
