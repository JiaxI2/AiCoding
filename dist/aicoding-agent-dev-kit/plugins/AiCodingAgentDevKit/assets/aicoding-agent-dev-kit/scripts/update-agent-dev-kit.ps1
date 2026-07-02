param([string]$RepoRoot = ".", [switch]$DryRun, [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
Write-AgentDevKitJson -Json:$Json -Data @{ ok = $true; action = "update"; dryRun = [bool]$DryRun; note = "Use install to refresh managed assets in v0.5 MVP." }
