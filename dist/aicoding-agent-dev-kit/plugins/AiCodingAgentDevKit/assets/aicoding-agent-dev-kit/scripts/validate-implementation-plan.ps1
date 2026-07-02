param([string]$RepoRoot = ".", [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$planPath = Join-Path $root "spec/IMPLEMENTATION_PLAN.md"
$missing = @()
$phrases = @(
  "Write failing test",
  "Run test and confirm failure",
  "Implement minimal code",
  "Run test and confirm pass",
  "Refactor",
  "Run test again"
)
if (-not (Test-Path -LiteralPath $planPath)) {
  $missing += "spec/IMPLEMENTATION_PLAN.md"
} else {
  $text = Get-Content -Raw -LiteralPath $planPath
  foreach ($p in $phrases) {
    if ($text -notmatch [regex]::Escape($p)) { $missing += $p }
  }
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok = ($missing.Count -eq 0); validator = "validate-implementation-plan"; missing = $missing }
if ($missing.Count -gt 0) { exit 1 }
