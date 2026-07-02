. "$PSScriptRoot/common.ps1"
$repo = Get-RepoRoot
$script = Join-Path $repo "scripts/load-agent-context.ps1"
if (Test-Path -LiteralPath $script) {
    & pwsh -NoProfile -ExecutionPolicy Bypass -File $script -RepoRoot $repo -Auto -Json | Out-Null
}
exit 0
