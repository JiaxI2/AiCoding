param([string]$RepoRoot = ".", [switch]$Json)
$global:LASTEXITCODE = 0
& "$PSScriptRoot\validate-spec-pack.ps1" -RepoRoot $RepoRoot -Json | Out-Null
if ((-not $?) -or $LASTEXITCODE -ne 0) { exit 1 }
$global:LASTEXITCODE = 0
& "$PSScriptRoot\validate-memory.ps1" -RepoRoot $RepoRoot -Json | Out-Null
if ((-not $?) -or $LASTEXITCODE -ne 0) { exit 1 }
$global:LASTEXITCODE = 0
& "$PSScriptRoot\validate-implementation-plan.ps1" -RepoRoot $RepoRoot -Json:$Json
if ((-not $?) -or $LASTEXITCODE -ne 0) { exit 1 }