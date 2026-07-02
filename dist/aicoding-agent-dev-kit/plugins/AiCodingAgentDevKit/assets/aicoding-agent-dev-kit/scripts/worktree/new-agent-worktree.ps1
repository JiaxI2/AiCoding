param([string]$RepoRoot=".", [Parameter(Mandatory=$true)][string]$Name, [string]$Base="main", [switch]$Json)
. "$PSScriptRoot\..\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$parent = Split-Path -Parent $root
$path = Join-Path $parent ("agent-wt-" + $Name)
git -C $root worktree add $path -b $Name $Base | Out-Null
New-AgentDevKitDirectory (Join-Path $root ".agent-memory/worktrees")
"Worktree: $Name`nPath: $path`nBase: $Base" | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $root ".agent-memory/worktrees/$Name.md")
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; name=$Name; path=$path; base=$Base }
