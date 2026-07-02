param([string]$RepoRoot=".", [switch]$Json)
. "$PSScriptRoot\..\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$list = git -C $root worktree list --porcelain
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; worktrees=$list }
