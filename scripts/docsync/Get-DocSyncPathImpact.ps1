param(
  [Parameter(Mandatory = $true)]
  [string[]]$Files
)

function Test-DocPath {
  param([string]$Path)
  return ($Path -match '(^|/)(README|README_CN|README_EN|CHANGELOG|AGENTS)\.md$' -or $Path -match '(^|/)docs/.*\.md$' -or $Path -match '(^|/)config/.*\.md$')
}

function Test-CodeLikePath {
  param([string]$Path)
  return (
    $Path -match '\.(c|h|cpp|hpp|py|ps1|sh|bat|cmake|json|yaml|yml|toml)$' -or
    $Path -match '^(src|scripts|config|\.githooks|\.github|\.agents|CodingKit|skills|codex-skills)/'
  )
}

$normalized = @($Files | Where-Object { $_ } | ForEach-Object { $_ -replace '\\','/' })
$docChanged = @($normalized | Where-Object { Test-DocPath $_ })
$codeLikeChanged = @($normalized | Where-Object { (Test-CodeLikePath $_) -and -not (Test-DocPath $_) })

[pscustomobject]@{
  changedFiles = $normalized
  docChanged = $docChanged
  codeLikeChanged = $codeLikeChanged
  hasDocChanged = ($docChanged.Count -gt 0)
  hasCodeLikeChanged = ($codeLikeChanged.Count -gt 0)
}
