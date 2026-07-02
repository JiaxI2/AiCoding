param([string]$RepoRoot=".", [Parameter(Mandatory=$true)][string]$Name, [switch]$Json)
. "$PSScriptRoot\..\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
git -C $root merge $Name
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; action="merge"; name=$Name }
