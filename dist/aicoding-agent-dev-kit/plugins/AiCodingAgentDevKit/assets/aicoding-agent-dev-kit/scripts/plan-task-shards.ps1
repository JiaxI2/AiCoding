param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$plan = Join-Path $root "spec/IMPLEMENTATION_PLAN.md"
$shardDir = Join-Path $root ".agent-dev-kit/shards"
New-AgentDevKitDirectory $shardDir
$created = @()
if (Test-Path -LiteralPath $plan) {
  $text = Get-Content -Raw -LiteralPath $plan
  $matches = [regex]::Matches($text, "(?m)^### Task\s+\d+:\s+(.+)$")
  $i = 0
  foreach ($m in $matches) {
    $i += 1
    $name = ($m.Groups[1].Value -replace "[^\w\-]+","-").Trim("-").ToLowerInvariant()
    if (-not $name) { $name = "task-$i" }
    $path = Join-Path $shardDir ("{0:00}-{1}.md" -f $i,$name)
    "# Task Shard $i`n`nSource: spec/IMPLEMENTATION_PLAN.md`n`nTask: $($m.Groups[1].Value)`n`nRecommended loop: Red -> Green -> Refactor -> Gate -> Commit`n" | Set-Content -Encoding UTF8 -LiteralPath $path
    $created += $path
  }
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; shardCount=$created.Count; shards=$created }
