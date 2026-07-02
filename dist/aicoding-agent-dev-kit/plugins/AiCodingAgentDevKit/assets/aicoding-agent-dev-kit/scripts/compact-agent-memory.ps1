param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  deprecated=$true
  message="v0.11.1 uses lightweight decision memory. Use current-set.ps1 and decision-add.ps1 instead of compacting long memory."
}
