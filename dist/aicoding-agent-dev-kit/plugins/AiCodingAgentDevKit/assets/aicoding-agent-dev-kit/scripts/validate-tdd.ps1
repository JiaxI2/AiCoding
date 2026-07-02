param([string]$RepoRoot = ".", [switch]$Json)
& "$PSScriptRoot\validate-implementation-plan.ps1" -RepoRoot $RepoRoot -Json:$Json
