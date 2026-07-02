param(
    [string]$RepoRoot = ".",
    [string]$Goal = "",
    [string]$Task = "",
    [string]$Next = "",
    [string]$Blockers = "None.",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$dir = Join-Path $root ".agent-memory"
New-AgentDevKitDirectory $dir
$path = Join-Path $dir "CURRENT.md"
$text = @"
# Current State

## Current Goal

$Goal

## Active Task

$Task

## Next Step

$Next

## Blockers

$Blockers
"@
$text | Set-Content -Encoding UTF8 -LiteralPath $path
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; path=$path; goal=$Goal; task=$Task; next=$Next }
