param([string]$RepoRoot='.',[switch]$FromImplementationPlan,[switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root=Resolve-AgentDevKitRepoRoot $RepoRoot; $dir=Join-Path $root '.agent-dev-kit/progress'; New-AgentDevKitDirectory $dir; $boardPath=Join-Path $dir 'progress-board.json'
$features=@()
if($FromImplementationPlan -and (Test-Path -LiteralPath (Join-Path $root 'spec/IMPLEMENTATION_PLAN.md'))){ $plan=Get-Content -Raw -LiteralPath (Join-Path $root 'spec/IMPLEMENTATION_PLAN.md'); $matches=[regex]::Matches($plan,'(?m)^### Task\s+(\d+):\s+(.+)$'); foreach($m in $matches){ $id='F-{0:000}' -f [int]$m.Groups[1].Value; $features += [ordered]@{featureId=$id;title=$m.Groups[2].Value.Trim();mvp=$true;status='todo';owner='agent';linkedSpec='spec/IMPLEMENTATION_PLAN.md';linkedTests=@();currentStep='';nextStep='Write failing test';evidence=@()} } }
if($features.Count -eq 0){$features += [ordered]@{featureId='F-001';title='Example small feature';mvp=$true;status='todo';owner='agent';linkedSpec='spec/IMPLEMENTATION_PLAN.md';linkedTests=@();currentStep='Not started';nextStep='Write failing test';evidence=@()}}
$board=[ordered]@{schema='aicoding-agent-dev-kit.progress-board.v1';version='0.11.1';updatedAt=(Get-Date).ToString('o');activeFeature='';features=$features}; $board|ConvertTo-Json -Depth 10|Set-Content -Encoding UTF8 -LiteralPath $boardPath
& "$PSScriptRoot\show-progress-board.ps1" -RepoRoot $root | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $dir 'PROGRESS_BOARD.md')
Write-AgentDevKitJson -Json:$Json -Data @{ok=$true;board=$boardPath;count=$features.Count}
