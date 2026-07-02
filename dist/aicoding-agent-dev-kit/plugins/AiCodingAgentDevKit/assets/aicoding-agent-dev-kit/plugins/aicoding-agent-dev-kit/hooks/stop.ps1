. "$PSScriptRoot/common.ps1"
$repo = Get-RepoRoot
$script = Join-Path $repo "scripts/show-context-manifest.ps1"
if (Test-Path -LiteralPath $script) {
    & pwsh -NoProfile -ExecutionPolicy Bypass -File $script -RepoRoot $repo -Json | Out-Null
}
exit 0
