param([string]$RepoRoot='.',[switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root=Resolve-AgentDevKitRepoRoot $RepoRoot; $boardPath=Join-Path $root '.agent-dev-kit/progress/progress-board.json'; $specPath=Join-Path $root 'spec/MVP_PROGRESS.md'
if(-not(Test-Path -LiteralPath $boardPath)){Write-AgentDevKitJson -Json:$Json -Data @{ok=$false;error='progress board not found';board=$boardPath}; exit 1}
$board=Get-Content -Raw -LiteralPath $boardPath|ConvertFrom-Json; $lines=@('# MVP Progress','',"Updated: $($board.updatedAt)",'','| Feature ID | Title | Status | Current Step | Next Step | Evidence |','|---|---|---|---|---|---|'); foreach($f in $board.features){$ev=(@($f.evidence)-join '; ');$lines += "| $($f.featureId) | $($f.title) | $($f.status) | $($f.currentStep) | $($f.nextStep) | $ev |"}; $lines -join "`n" | Set-Content -Encoding UTF8 -LiteralPath $specPath; Write-AgentDevKitJson -Json:$Json -Data @{ok=$true;spec=$specPath;count=$board.features.Count}
