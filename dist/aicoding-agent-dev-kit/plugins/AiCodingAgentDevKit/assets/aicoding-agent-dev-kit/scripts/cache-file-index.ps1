param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$cacheDir = Join-Path $root ".agent-dev-kit/cache"
New-AgentDevKitDirectory $cacheDir
$files = Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue |
  Where-Object { $_.FullName -notmatch "\\.git\\" -and $_.FullName -notmatch "\\node_modules\\" -and $_.FullName -notmatch "\\.next\\" } |
  Select-Object @{n="path";e={$_.FullName.Substring($root.Length + 1).Replace("\","/")}}, Length, LastWriteTimeUtc
$index = [ordered]@{
  schema = "aicoding-agent-dev-kit.repo-index.v1"
  generatedAt = (Get-Date).ToString("o")
  repoRoot = $root
  fileCount = @($files).Count
  files = $files
}
$path = Join-Path $cacheDir "repo-index.json"
$index | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 -LiteralPath $path
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; indexPath=$path; fileCount=@($files).Count }
