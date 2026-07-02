param([string]$RepoRoot = ".", [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$required = @(".agent-memory/CURRENT.md",".agent-memory/DECISIONS.md")
$missing = @()
foreach ($r in $required) {
  if (-not (Test-Path -LiteralPath (Join-Path $root $r))) { $missing += $r }
}
$tooLarge = @()
foreach ($r in $required) {
  $p = Join-Path $root $r
  if (Test-Path -LiteralPath $p) {
    $len = (Get-Content -Raw -LiteralPath $p).Length
    if ($r -like "*CURRENT.md" -and $len -gt 4000) { $tooLarge += "$r exceeds 4000 chars" }
    if ($r -like "*DECISIONS.md" -and $len -gt 40000) { $tooLarge += "$r exceeds 40000 chars; promote old decisions to ADR/docs" }
  }
}
$errors = @($missing + $tooLarge)
Write-AgentDevKitJson -Json:$Json -Data @{ ok = ($errors.Count -eq 0); validator = "validate-memory"; errors = $errors }
if ($errors.Count -gt 0) { exit 1 }
