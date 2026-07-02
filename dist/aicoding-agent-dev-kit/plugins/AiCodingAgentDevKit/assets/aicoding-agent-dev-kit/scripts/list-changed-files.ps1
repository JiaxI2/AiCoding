param(
    [string]$RepoRoot = ".",
    [switch]$Staged,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
if ($Staged) {
  $changed = git -C $root diff --cached --name-only
} else {
  $changed = git -C $root diff --name-only HEAD
  if (-not $changed) { $changed = git -C $root status --short | ForEach-Object { ($_ -replace "^\s*\S+\s+","") } }
}
$items = @($changed | Where-Object { $_ -and ($_ -notmatch "^\s*$") } | Sort-Object -Unique)
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; staged=[bool]$Staged; count=$items.Count; files=$items }
