param([string]$RepoRoot = ".", [switch]$Json)
& "$PSScriptRoot\invoke-agent-quality-gate.ps1" -RepoRoot $RepoRoot -Mode all -Json:$Json
