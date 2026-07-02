param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$changed = git -C $root diff --name-only HEAD
if (-not $changed) { $changed = git -C $root status --short | ForEach-Object { ($_ -replace "^\s*\S+\s+","") } }
function Read-Short([string]$Path, [int]$Max=1600) {
  if (-not (Test-Path -LiteralPath $Path)) { return "" }
  $t = Get-Content -Raw -LiteralPath $Path
  if ($t.Length -gt $Max) { return $t.Substring(0,$Max) + "`n...[truncated]..." }
  return $t
}
$data = @{
  ok=$true
  repoRoot=$root
  changedCount=@($changed).Count
  changedFiles=@($changed | Select-Object -First 30)
  current=Read-Short (Join-Path $root ".agent-memory/CURRENT.md") 1600
  decisions=Read-Short (Join-Path $root ".agent-memory/DECISIONS.md") 2500
  nextRecommendedCommands=@(
    "scripts/build-agent-context-pack.ps1 -Mode changed -MaxChars 12000 -Json",
    "scripts/token-audit.ps1 -Json",
    "scripts/invoke-agent-quality-gate.ps1 -Mode pre-commit -Json"
  )
}
Write-AgentDevKitJson -Json:$Json -Data $data
