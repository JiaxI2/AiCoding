param([string]$RepoRoot='.',[switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root=Resolve-AgentDevKitRepoRoot $RepoRoot; $path=Join-Path $root '.agent-dev-kit/progress/progress-board.json'
if(-not(Test-Path -LiteralPath $path)){Write-AgentDevKitJson -Json:$Json -Data @{ok=$false;error='progress board not found';path=$path}; exit 1}
if($Json){Get-Content -Raw -LiteralPath $path; exit 0}
$board=Get-Content -Raw -LiteralPath $path|ConvertFrom-Json; Write-Output '# Progress Board'; Write-Output ''; Write-Output "Updated: $($board.updatedAt)"; Write-Output ''; Write-Output '| Feature ID | Title | Status | Current Step | Next Step |'; Write-Output '|---|---|---|---|---|'; foreach($f in $board.features){Write-Output "| $($f.featureId) | $($f.title) | $($f.status) | $($f.currentStep) | $($f.nextStep) |"}
