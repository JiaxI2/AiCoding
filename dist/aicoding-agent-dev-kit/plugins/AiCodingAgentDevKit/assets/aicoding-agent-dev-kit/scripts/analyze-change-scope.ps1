param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot

$changed = git -C $root diff --name-only HEAD
if (-not $changed) {
  $changed = git -C $root status --short | ForEach-Object { ($_ -replace "^\s*\S+\s+","") }
}
$files = @($changed | Where-Object { $_ -and ($_ -notmatch "^\s*$") } | Sort-Object -Unique)

$stage = "L1"
$reason = "default changed-file task context"
foreach ($f in $files) {
  if ($f -match "^(\.github/|\.githooks/|scripts/|config/)") {
    $stage = "L2"
    $reason = "workflow/hook/script/config change"
  }
  if ($f -match "^(spec/|docs/adr/|specs/bdd/|specs/tdd/|docs/traceability/)") {
    $stage = "L3"
    $reason = "spec/ADR/BDD/TDD/traceability change"
    break
  }
}
if ($files.Count -eq 0) {
  $stage = "L0"
  $reason = "no changed files"
}

Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  stage=$stage
  reason=$reason
  changedCount=$files.Count
  changedFiles=$files
}
