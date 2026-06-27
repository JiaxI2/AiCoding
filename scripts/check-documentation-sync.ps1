param(
  [ValidateSet('pre-commit','all')]
  [string]$Mode = 'pre-commit',
  [switch]$Staged,
  [string]$PolicyPath = 'config/docs-sync.policy.json'
)

$ErrorActionPreference = 'Stop'

function Get-GitFiles {
  param([switch]$Staged)

  if ($Staged) {
    return git diff --cached --name-only --diff-filter=ACMRT
  }

  if ($env:GITHUB_EVENT_NAME -eq 'pull_request' -and $env:GITHUB_BASE_REF) {
    git fetch --no-tags --depth=1 origin $env:GITHUB_BASE_REF | Out-Null
    $base = git merge-base HEAD "origin/$env:GITHUB_BASE_REF"
    return git diff --name-only --diff-filter=ACMRT $base HEAD
  }

  if ($env:GITHUB_EVENT_BEFORE -and $env:GITHUB_EVENT_BEFORE -notmatch '^0+$') {
    return git diff --name-only --diff-filter=ACMRT $env:GITHUB_EVENT_BEFORE HEAD
  }

  if ($env:GITHUB_ACTIONS) {
    git rev-parse --verify HEAD^ | Out-Null 2>$null
    if ($LASTEXITCODE -eq 0) {
      return git diff --name-only --diff-filter=ACMRT HEAD^ HEAD
    }
  }

  return git diff --name-only --diff-filter=ACMRT HEAD
}

function Test-DocPath {
  param([string]$Path)
  return ($Path -match '(^|/)(README|README_CN|CHANGELOG|AGENTS)\.md$' -or $Path -match '(^|/)docs/.*\.md$' -or $Path -match '(^|/)config/.*\.md$')
}

function Test-CodeLikePath {
  param([string]$Path)
  return (
    $Path -match '\.(c|h|cpp|hpp|py|ps1|sh|bat|cmake|json|yaml|yml|toml)$' -or
    $Path -match '^(src|scripts|config|\.githooks|\.github|\.agents|CodingKit|skills|codex-skills)/'
  )
}

if (-not (Test-Path -LiteralPath $PolicyPath)) {
  throw "Documentation sync policy not found: $PolicyPath"
}

$files = @(Get-GitFiles -Staged:$Staged | Where-Object { $_ -and $_.Trim() -ne '' })
if ($files.Count -eq 0) {
  Write-Host '[docsync] No changed files to check.'
  exit 0
}

$docChanged = @($files | Where-Object { Test-DocPath $_ })
$codeLikeChanged = @($files | Where-Object { Test-CodeLikePath $_ -and -not (Test-DocPath $_) })

if ($codeLikeChanged.Count -eq 0) {
  Write-Host '[docsync] No code/config/platform changes requiring documentation review.'
  exit 0
}

if ($docChanged.Count -gt 0) {
  Write-Host '[docsync] Documentation update detected:'
  $docChanged | ForEach-Object { Write-Host "  - $_" }
  exit 0
}

Write-Error @"
[docsync] Code/platform changes are staged but no documentation update is staged.

Changed files requiring docs review:
$($codeLikeChanged | ForEach-Object { "  - $_" } | Out-String)

Fix one of the following:
  1. Update and stage the relevant docs, for example SDD, BDD, README, AGENTS, CHANGELOG, docs/**, or config docs.
  2. Add a short no-doc-change justification to an appropriate docs file or commit plan using marker: DOCSYNC-NO-DOC-CHANGE.

This gate is independent of Superpowers.
"@
exit 1
