param(
  [string]$RepoRoot = '.',
  [switch]$Json,
  [switch]$KeepTemp
)

$ErrorActionPreference = 'Stop'
$repo = Resolve-Path -LiteralPath $RepoRoot
$tests = New-Object System.Collections.Generic.List[object]

function Add-Test([string]$Name, [bool]$Passed, [string]$Message) {
  $tests.Add([pscustomobject]@{ name = $Name; passed = $Passed; message = $Message }) | Out-Null
}

# Static fixture-light tests. They validate the modules against the current repo without mutating git state.
try {
  & (Join-Path $repo.Path 'scripts/docsync/Get-MarkdownCommandIndex.ps1') -RepoRoot $repo.Path | Out-Null
  Add-Test 'markdown-command-index-runs' $true 'Markdown command index executed.'
} catch { Add-Test 'markdown-command-index-runs' $false $_.Exception.Message }

try {
  & (Join-Path $repo.Path 'scripts/docsync/Get-PowerShellSemanticDiff.ps1') -RepoRoot $repo.Path -Files @('scripts/check-documentation-sync.ps1') | Out-Null
  Add-Test 'powershell-semantic-diff-runs' $true 'PowerShell semantic diff executed.'
} catch { Add-Test 'powershell-semantic-diff-runs' $false $_.Exception.Message }

try {
  & (Join-Path $repo.Path 'scripts/docsync/Get-JsonPolicyDiff.ps1') -RepoRoot $repo.Path -Files @('config/docs-sync.policy.json','config/docs-sync.semantic.json') | Out-Null
  Add-Test 'json-policy-diff-runs' $true 'JSON policy diff executed.'
} catch { Add-Test 'json-policy-diff-runs' $false $_.Exception.Message }

try {
  $valid = & (Join-Path $repo.Path 'scripts/docsync/Test-DocSyncNoDocMarker.ps1') -Text 'DOCSYNC-NO-DOC-CHANGE: only changed internal whitespace in a fixture and no user-facing behavior changed.'
  Add-Test 'valid-no-doc-marker' ([bool]$valid.ok) 'Valid marker should pass.'
} catch { Add-Test 'valid-no-doc-marker' $false $_.Exception.Message }

try {
  $invalid = & (Join-Path $repo.Path 'scripts/docsync/Test-DocSyncNoDocMarker.ps1') -Text 'DOCSYNC-NO-DOC-CHANGE: skip'
  Add-Test 'invalid-no-doc-marker' (-not [bool]$invalid.ok) 'Invalid marker should fail.'
} catch { Add-Test 'invalid-no-doc-marker' $false $_.Exception.Message }

try {
  $fenced = @('```text','DOCSYNC-NO-DOC-CHANGE: skip','```') -join [Environment]::NewLine
  $ignored = & (Join-Path $repo.Path 'scripts/docsync/Test-DocSyncNoDocMarker.ps1') -Text $fenced
  Add-Test 'fenced-no-doc-marker-ignored' (-not [bool]$ignored.found -and [bool]$ignored.ok) 'Invalid marker examples inside fenced code should be ignored.'
} catch { Add-Test 'fenced-no-doc-marker-ignored' $false $_.Exception.Message }

try {
  $report = & (Join-Path $repo.Path 'scripts/docsync/Invoke-DocSyncPlus.ps1') -RepoRoot $repo.Path -Mode all -Format json
  $obj = $report | ConvertFrom-Json
  Add-Test 'invoke-docsync-plus-json' ($null -ne $obj.ok) 'Invoke-DocSyncPlus produced JSON.'
} catch { Add-Test 'invoke-docsync-plus-json' $false $_.Exception.Message }

$passed = @($tests | Where-Object { $_.passed }).Count
$total = $tests.Count
$result = [ordered]@{}
$result['ok'] = ($passed -eq $total)
$result['repoRoot'] = $repo.Path
$result['passed'] = $passed
$result['total'] = $total
$result['tests'] = @($tests.ToArray())

if ($Json) { $result | ConvertTo-Json -Depth 8 } else { $result | Format-List }
if (-not $result['ok']) { exit 1 }
