param(
    [string]$RepoRoot = ".",
    [string]$Summary = "",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  deprecated=$true
  message="v0.11.1 removed session-summary. Use current-set.ps1 for current state and decision-add.ps1 for important decisions."
}
