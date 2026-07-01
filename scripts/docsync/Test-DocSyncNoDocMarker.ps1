param(
  [string]$Text,
  [string]$Marker = 'DOCSYNC-NO-DOC-CHANGE',
  [int]$MinimumReasonLength = 20,
  [string[]]$ForbiddenReasons = @('skip','none','no','n/a','no change')
)

function Remove-FencedCodeBlocks {
  param([string]$InputText)

  $lines = $InputText -split "\r?\n"
  $kept = New-Object System.Collections.Generic.List[string]
  $inFence = $false

  foreach ($line in $lines) {
    if ($line -match '^\s*```') {
      $inFence = -not $inFence
      continue
    }

    if (-not $inFence) { $kept.Add($line) | Out-Null }
  }

  return ($kept.ToArray() -join "`n")
}

$scanText = Remove-FencedCodeBlocks -InputText $Text
$matches = [regex]::Matches($scanText, [regex]::Escape($Marker) + '\s*:?(?<reason>.*)')
if ($matches.Count -eq 0) {
  return [pscustomobject]@{ ok = $true; found = $false; reason = $null; errors = @() }
}

$errors = New-Object System.Collections.Generic.List[string]
foreach ($m in $matches) {
  $reason = $m.Groups['reason'].Value.Trim()
  if ($reason.Length -lt $MinimumReasonLength) { $errors.Add("No-doc marker reason is too short: '$reason'") | Out-Null }
  foreach ($bad in $ForbiddenReasons) {
    if ($reason.ToLowerInvariant() -eq $bad.ToLowerInvariant()) { $errors.Add("No-doc marker reason is forbidden: '$reason'") | Out-Null }
  }
}

[pscustomobject]@{
  ok = ($errors.Count -eq 0)
  found = $true
  reason = $matches[$matches.Count - 1].Groups['reason'].Value.Trim()
  errors = @($errors.ToArray())
}
