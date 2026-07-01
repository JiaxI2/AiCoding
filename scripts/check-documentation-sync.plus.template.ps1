param(
  [ValidateSet('pre-commit','all','ci','release')]
  [string]$Mode = 'pre-commit',

  [switch]$Staged,

  [string]$PolicyPath = 'config/docs-sync.policy.json',

  [string]$SemanticPolicyPath = 'config/docs-sync.semantic.json',

  [ValidateSet('text','json','markdown')]
  [string]$Format = 'text',

  [string]$ReportPath
)

$ErrorActionPreference = 'Stop'
$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptRoot '..')
$invoke = Join-Path $scriptRoot 'docsync/Invoke-DocSyncPlus.ps1'

if (-not (Test-Path -LiteralPath $invoke)) {
  throw "DocSync Plus entry module not found: $invoke"
}

& $invoke -RepoRoot $repoRoot.Path -Mode $Mode -Staged:$Staged -PolicyPath $PolicyPath -SemanticPolicyPath $SemanticPolicyPath -Format $Format -ReportPath $ReportPath
exit $LASTEXITCODE
