param(
  [string]$RepoRoot = '.',
  [switch]$Staged
)

$ErrorActionPreference = 'Stop'
$repo = Resolve-Path -LiteralPath $RepoRoot
Push-Location $repo.Path
try {
  if ($Staged) {
    $files = git diff --cached --name-only --diff-filter=ACMRT
  } elseif ($env:GITHUB_EVENT_NAME -eq 'pull_request' -and $env:GITHUB_BASE_REF) {
    git fetch --no-tags --depth=1 origin $env:GITHUB_BASE_REF | Out-Null
    $base = git merge-base HEAD "origin/$env:GITHUB_BASE_REF"
    $files = git diff --name-only --diff-filter=ACMRT $base HEAD
  } elseif ($env:GITHUB_EVENT_BEFORE -and $env:GITHUB_EVENT_BEFORE -notmatch '^0+$') {
    $files = git diff --name-only --diff-filter=ACMRT $env:GITHUB_EVENT_BEFORE HEAD
  } elseif ($env:GITHUB_ACTIONS) {
    git rev-parse --verify HEAD^ | Out-Null 2>$null
    if ($LASTEXITCODE -eq 0) { $files = git diff --name-only --diff-filter=ACMRT HEAD^ HEAD } else { $files = git diff --name-only --diff-filter=ACMRT HEAD }
  } else {
    $files = git diff --name-only --diff-filter=ACMRT HEAD
  }
  @($files | Where-Object { $_ -and $_.Trim() -ne '' } | ForEach-Object { $_ -replace '\\','/' })
} finally {
  Pop-Location
}
