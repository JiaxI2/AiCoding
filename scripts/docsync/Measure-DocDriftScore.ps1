param(
  [Parameter(Mandatory = $true)]
  [object]$PathImpact,

  [object[]]$PowerShellSurface = @(),
  [object[]]$JsonPolicySurface = @(),
  [object[]]$MarkdownCommands = @(),
  [object]$SemanticPolicy,
  [object]$NoDocMarkerResult
)

$score = 0
$reasons = New-Object System.Collections.Generic.List[string]
$requiredDocs = New-Object System.Collections.Generic.List[string]

function Add-Score([int]$Points, [string]$Reason) {
  $script:score += $Points
  $script:reasons.Add($Reason) | Out-Null
}
function Add-Doc([string]$Doc) {
  if ($Doc -and -not $script:requiredDocs.Contains($Doc)) { $script:requiredDocs.Add($Doc) | Out-Null }
}

if ($PathImpact.hasCodeLikeChanged -and -not $PathImpact.hasDocChanged) {
  Add-Score 45 'Code/config/platform changes detected without documentation changes.'
  Add-Doc 'README.md'; Add-Doc 'CHANGELOG.md'; Add-Doc 'docs/MAINTENANCE_METHOD.md'
}

foreach ($ps in @($PowerShellSurface)) {
  if ($ps.path -match '^scripts/' -and $ps.parameters.Count -gt 0) {
    if (-not $PathImpact.hasDocChanged) { Add-Score 15 "PowerShell public parameter surface changed or requires review: $($ps.path)" }
    Add-Doc 'README.md'; Add-Doc 'docs/MAINTENANCE_METHOD.md'; Add-Doc 'docs/DOC_SYNC_PLUS_SPEC.md'
  }
}

foreach ($json in @($JsonPolicySurface | Where-Object { $_ })) {
  if (-not $json.valid) { Add-Score 30 "Invalid JSON policy/config file: $($json.path)"; continue }
  if ($json.path -match 'config/docs-sync\.(policy|semantic)\.json') {
    if (-not $PathImpact.hasDocChanged) { Add-Score 20 "DocSync policy changed or requires review: $($json.path)" }
    Add-Doc 'docs/DOC_SYNC_PLUS_SPEC.md'; Add-Doc 'docs/MAINTENANCE_METHOD.md'; Add-Doc 'README.md'; Add-Doc 'CHANGELOG.md'
  }
}

foreach ($cmd in @($MarkdownCommands)) {
  # This is a low-cost command index. Deep argument validation is done in Invoke-DocSyncPlus.
  if (-not $cmd.script) { continue }
}

if ($NoDocMarkerResult -and $NoDocMarkerResult.found -and -not $NoDocMarkerResult.ok) {
  Add-Score 25 'DOCSYNC-NO-DOC-CHANGE marker is present but invalid or too short.'
}

if ($score -gt 100) { $score = 100 }

[pscustomobject]@{
  score = $score
  reasons = @($reasons.ToArray())
  requiredDocs = @($requiredDocs.ToArray())
}
