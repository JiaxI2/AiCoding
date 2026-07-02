param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$path = Join-Path $root ".agent-memory/CURRENT.md"
$content = ""
if (Test-Path -LiteralPath $path) { $content = Get-Content -Raw -LiteralPath $path }
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; path=$path; content=$content }
