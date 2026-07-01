param(
  [string]$RepoRoot = '.',
  [switch]$Json,
  [switch]$Strict
)

$ErrorActionPreference = 'Stop'
$repo = Resolve-Path -LiteralPath $RepoRoot
$errors = New-Object System.Collections.Generic.List[string]
$warnings = New-Object System.Collections.Generic.List[string]

function Add-Error([string]$m) { $errors.Add($m) | Out-Null }
function Add-Warning([string]$m) { $warnings.Add($m) | Out-Null }
function Test-RelPath([string]$rel) { Test-Path -LiteralPath (Join-Path $repo.Path $rel) }

$required = @(
  'scripts/check-documentation-sync.ps1',
  'scripts/docsync/Invoke-DocSyncPlus.ps1',
  'scripts/docsync/Get-DocSyncChangedFiles.ps1',
  'scripts/docsync/Get-DocSyncPathImpact.ps1',
  'scripts/docsync/Get-PowerShellSemanticDiff.ps1',
  'scripts/docsync/Get-JsonPolicyDiff.ps1',
  'scripts/docsync/Get-MarkdownCommandIndex.ps1',
  'scripts/docsync/Test-DocSyncNoDocMarker.ps1',
  'scripts/docsync/Measure-DocDriftScore.ps1',
  'scripts/docsync/New-DocSyncReport.ps1',
  'scripts/install-docsync-plus.ps1',
  'scripts/status-docsync-plus.ps1',
  'scripts/verify-docsync-plus.ps1',
  'scripts/test-docsync-plus.ps1',
  'config/docs-sync.policy.json',
  'config/docs-sync.semantic.json',
  'docs/DOC_SYNC_PLUS_SPEC.md',
  'docs/DOC_SYNC_PLUS_VALIDATION_PLAN.md'
)
foreach ($rel in $required) { if (-not (Test-RelPath $rel)) { Add-Error "Missing required file: $rel" } }

foreach ($jsonRel in @('config/docs-sync.policy.json','config/docs-sync.semantic.json')) {
  $path = Join-Path $repo.Path $jsonRel
  if (Test-Path -LiteralPath $path) {
    try { Get-Content -Raw -LiteralPath $path | ConvertFrom-Json | Out-Null } catch { Add-Error "Invalid JSON: $jsonRel :: $($_.Exception.Message)" }
  }
}

$psFiles = Get-ChildItem -LiteralPath (Join-Path $repo.Path 'scripts') -Filter '*.ps1' -Recurse -ErrorAction SilentlyContinue
foreach ($ps in $psFiles) {
  try {
    $tokens = $null; $parseErrors = $null
    [System.Management.Automation.PSParser]::Tokenize((Get-Content -Raw -LiteralPath $ps.FullName), [ref]$parseErrors) | Out-Null
    if ($parseErrors -and $parseErrors.Count -gt 0) { Add-Error "PowerShell parse error: $($ps.FullName)" }
  } catch { Add-Error "PowerShell parse failed: $($ps.FullName) :: $($_.Exception.Message)" }
}

$hook = Join-Path $repo.Path '.githooks/pre-commit'
if (Test-Path -LiteralPath $hook) {
  if ((Get-Content -Raw -LiteralPath $hook) -notmatch 'check-documentation-sync\.ps1') { Add-Error 'pre-commit hook does not call check-documentation-sync.ps1' }
} else { Add-Warning 'Missing .githooks/pre-commit' }

$workflow = Join-Path $repo.Path '.github/workflows/docs-sync.yml'
if (Test-Path -LiteralPath $workflow) {
  if ((Get-Content -Raw -LiteralPath $workflow) -notmatch 'check-documentation-sync\.ps1') { Add-Error 'docs-sync workflow does not call check-documentation-sync.ps1' }
} else { Add-Warning 'Missing .github/workflows/docs-sync.yml' }

if ($Strict) {
  foreach ($doc in @('README.md','CHANGELOG.md','docs/MAINTENANCE_METHOD.md')) {
    if (-not (Test-RelPath $doc)) { Add-Error "Missing strict doc: $doc" }
    elseif ((Get-Content -Raw -LiteralPath (Join-Path $repo.Path $doc)) -notmatch 'DocSync Plus|docsync') { Add-Warning "Strict doc may not mention DocSync Plus/docsync: $doc" }
  }
}

$result = [ordered]@{}
$result['ok'] = ($errors.Count -eq 0)
$result['repoRoot'] = $repo.Path
$result['strict'] = ([bool]$Strict)
$result['errors'] = @($errors.ToArray())
$result['warnings'] = @($warnings.ToArray())

if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
if (-not $result['ok']) { exit 1 }
