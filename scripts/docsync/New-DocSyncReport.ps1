param(
  [Parameter(Mandatory = $true)]
  [object]$Result,

  [ValidateSet('text','json','markdown')]
  [string]$Format = 'text',

  [string]$ReportPath
)

if ($Format -eq 'json') {
  $out = ($Result | ConvertTo-Json -Depth 12)
} elseif ($Format -eq 'markdown') {
  $lines = New-Object System.Collections.Generic.List[string]
  $lines.Add('# DocSync Plus Report') | Out-Null
  $lines.Add('') | Out-Null
  $lines.Add("Mode: $($Result.mode)  ") | Out-Null
  $lines.Add("Score: $($Result.score) / 100  ") | Out-Null
  $lines.Add("Status: $($Result.level.ToUpperInvariant())") | Out-Null
  $lines.Add('') | Out-Null
  $lines.Add('## Reasons') | Out-Null
  if ($Result.reasons.Count -eq 0) { $lines.Add('- No drift detected.') | Out-Null } else { foreach ($r in $Result.reasons) { $lines.Add("- $r") | Out-Null } }
  $lines.Add('') | Out-Null
  $lines.Add('## Required Docs') | Out-Null
  if ($Result.requiredDocs.Count -eq 0) { $lines.Add('- None') | Out-Null } else { foreach ($d in $Result.requiredDocs) { $lines.Add(('- `{0}`' -f $d)) | Out-Null } }
  $out = ($lines -join [Environment]::NewLine)
} else {
  $lines = New-Object System.Collections.Generic.List[string]
  $lines.Add("[docsync-plus] Mode: $($Result.mode)") | Out-Null
  $lines.Add("[docsync-plus] Changed files: $($Result.changedFiles.Count)") | Out-Null
  $lines.Add("[docsync-plus] Doc drift score: $($Result.score)/100") | Out-Null
  $lines.Add("[docsync-plus] Status: $($Result.level.ToUpperInvariant())") | Out-Null
  if ($Result.reasons.Count -gt 0) {
    $lines.Add('') | Out-Null
    $lines.Add('Reasons:') | Out-Null
    foreach ($r in $Result.reasons) { $lines.Add("  - $r") | Out-Null }
  }
  if ($Result.requiredDocs.Count -gt 0) {
    $lines.Add('') | Out-Null
    $lines.Add('Required docs:') | Out-Null
    foreach ($d in $Result.requiredDocs) { $lines.Add("  - $d") | Out-Null }
  }
  $out = ($lines -join [Environment]::NewLine)
}

if ($ReportPath) {
  $dir = Split-Path -Parent $ReportPath
  if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
  Set-Content -LiteralPath $ReportPath -Value $out -Encoding UTF8
}
$out
