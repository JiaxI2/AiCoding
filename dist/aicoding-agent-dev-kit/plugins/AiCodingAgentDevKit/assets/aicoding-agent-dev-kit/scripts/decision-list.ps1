param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$path = Join-Path $root ".agent-memory/DECISIONS.md"
$items = @()
if (Test-Path -LiteralPath $path) {
  $text = Get-Content -Raw -LiteralPath $path
  $matches = [regex]::Matches($text, "(?m)^##\s+(D-\d{4}):\s+(.+)$")
  foreach ($m in $matches) {
    $items += [ordered]@{ id=$m.Groups[1].Value; title=$m.Groups[2].Value }
  }
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; path=$path; count=$items.Count; decisions=$items }
