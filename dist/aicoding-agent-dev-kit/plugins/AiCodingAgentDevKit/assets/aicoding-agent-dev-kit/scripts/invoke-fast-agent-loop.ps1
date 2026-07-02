param(
    [string]$RepoRoot = ".",
    [int]$MaxChars = 0,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
& "$PSScriptRoot\cache-file-index.ps1" -RepoRoot $root -Json | Out-Null
& "$PSScriptRoot\load-agent-context.ps1" -RepoRoot $root -Auto -MaxChars $MaxChars -Json | Out-Null
& "$PSScriptRoot\token-audit.ps1" -RepoRoot $root -Json | Out-Null
Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  action="fast-agent-loop"
  contextPack=".agent-dev-kit/context/context-pack.md"
  manifest=".agent-dev-kit/context/context-manifest.json"
}
