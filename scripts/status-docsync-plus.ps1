param(
  [string]$RepoRoot = '.',
  [switch]$Json
)

$ErrorActionPreference = 'Stop'
$repo = Resolve-Path -LiteralPath $RepoRoot
function Test-RelPath([string]$rel) { Test-Path -LiteralPath (Join-Path $repo.Path $rel) }

$modules = @(
  'Invoke-DocSyncPlus.ps1',
  'Get-DocSyncChangedFiles.ps1',
  'Get-DocSyncPathImpact.ps1',
  'Get-PowerShellSemanticDiff.ps1',
  'Get-JsonPolicyDiff.ps1',
  'Get-MarkdownCommandIndex.ps1',
  'Test-DocSyncNoDocMarker.ps1',
  'Measure-DocDriftScore.ps1',
  'New-DocSyncReport.ps1'
)
$moduleStatus = [ordered]@{}
foreach ($m in $modules) { $moduleStatus[$m] = Test-RelPath ("scripts/docsync/$m") }

$hookPath = Join-Path $repo.Path '.githooks/pre-commit'
$workflowPath = Join-Path $repo.Path '.github/workflows/docs-sync.yml'
$hookIntegrated = $false
$ciIntegrated = $false
if (Test-Path -LiteralPath $hookPath) { $hookText = Get-Content -Raw -LiteralPath $hookPath; $hookIntegrated = ($hookText -match 'check-documentation-sync\.ps1') -or ($hookText -match 'bin/aicoding(\.exe)? hook pre-commit') -or ($hookText -match 'go run ./cmd/aicoding hook pre-commit') }
if (Test-Path -LiteralPath $workflowPath) { $ciIntegrated = ((Get-Content -Raw -LiteralPath $workflowPath) -match 'check-documentation-sync\.ps1') }

$result = [ordered]@{
  ok = ((Test-RelPath 'scripts/check-documentation-sync.ps1') -and (Test-RelPath 'config/docs-sync.policy.json') -and (Test-RelPath 'config/docs-sync.semantic.json') -and $hookIntegrated -and $ciIntegrated)
  repoRoot = $repo.Path
  entrypoint = 'scripts/check-documentation-sync.ps1'
  policy = 'config/docs-sync.policy.json'
  semanticPolicy = 'config/docs-sync.semantic.json'
  modules = $moduleStatus
  hookIntegrated = $hookIntegrated
  ciIntegrated = $ciIntegrated
}

if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
if (-not $result['ok']) { exit 1 }
