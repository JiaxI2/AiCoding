param([string]$RepoRoot=".", [Parameter(Mandatory=$true)][string]$Path, [switch]$Force, [switch]$Json)
. "$PSScriptRoot\..\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$args = @("worktree","remove",$Path)
if ($Force) { $args += "--force" }
git -C $root @args
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; action="remove"; path=$Path; force=[bool]$Force }
