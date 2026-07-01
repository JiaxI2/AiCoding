param(
  [string]$RepoRoot = '.',
  [switch]$DryRun,
  [switch]$Json
)

$ErrorActionPreference = 'Stop'
$repo = Resolve-Path -LiteralPath $RepoRoot
$requiredDirs = @('scripts/docsync','config','docs')
$actions = New-Object System.Collections.Generic.List[object]

foreach ($dir in $requiredDirs) {
  $path = Join-Path $repo.Path $dir
  $exists = Test-Path -LiteralPath $path
  $actionName = if ($exists) { 'exists' } else { 'create-dir' }
  $actions.Add([pscustomobject]@{ action = $actionName; path = $dir }) | Out-Null
  if (-not $DryRun -and -not $exists) { New-Item -ItemType Directory -Path $path -Force | Out-Null }
}

$checks = @(
  'scripts/check-documentation-sync.ps1',
  'config/docs-sync.policy.json',
  'config/docs-sync.semantic.json',
  'scripts/docsync/Invoke-DocSyncPlus.ps1',
  'scripts/status-docsync-plus.ps1',
  'scripts/verify-docsync-plus.ps1',
  'scripts/test-docsync-plus.ps1'
)

$missing = @()
foreach ($item in $checks) {
  if (-not (Test-Path -LiteralPath (Join-Path $repo.Path $item))) { $missing += $item }
}

$result = [ordered]@{}
$result['ok'] = ($missing.Count -eq 0)
$result['repoRoot'] = $repo.Path
$result['dryRun'] = ([bool]$DryRun)
$result['missing'] = @($missing)
$result['actions'] = @($actions.ToArray())
$result['note'] = 'This installer validates local DocSync Plus layout. Use the package root installer to copy kit files into AiCoding first.'

if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
if (-not $result['ok']) { exit 1 }
