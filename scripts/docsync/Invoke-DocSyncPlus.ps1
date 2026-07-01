param(
  [string]$RepoRoot = '.',

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
$repo = Resolve-Path -LiteralPath $RepoRoot
$moduleRoot = Split-Path -Parent $MyInvocation.MyCommand.Path

$changed = & (Join-Path $moduleRoot 'Get-DocSyncChangedFiles.ps1') -RepoRoot $repo.Path -Staged:$Staged
$changed = @($changed)

if ($changed.Count -eq 0) {
  $result = [pscustomobject]@{
    ok = $true; mode = $Mode; level = 'pass'; score = 0; threshold = 0
    changedFiles = @(); docChanged = @(); codeLikeChanged = @()
    reasons = @(); requiredDocs = @(); semanticChanges = @()
  }
  & (Join-Path $moduleRoot 'New-DocSyncReport.ps1') -Result $result -Format $Format -ReportPath $ReportPath
  exit 0
}

$pathImpact = & (Join-Path $moduleRoot 'Get-DocSyncPathImpact.ps1') -Files $changed

$semanticPolicyFile = Join-Path $repo.Path $SemanticPolicyPath
$semanticPolicy = $null
if (Test-Path -LiteralPath $semanticPolicyFile) { $semanticPolicy = Get-Content -Raw -LiteralPath $semanticPolicyFile | ConvertFrom-Json }

$psSurface = & (Join-Path $moduleRoot 'Get-PowerShellSemanticDiff.ps1') -RepoRoot $repo.Path -Files $changed
$jsonSurface = & (Join-Path $moduleRoot 'Get-JsonPolicyDiff.ps1') -RepoRoot $repo.Path -Files $changed
$mdCommands = & (Join-Path $moduleRoot 'Get-MarkdownCommandIndex.ps1') -RepoRoot $repo.Path -Files @($pathImpact.docChanged)

# Validate arguments only for scripts that belong to this repository. Cross-repository
# examples and package-internal paths are intentionally skipped.
$commandDriftReasons = New-Object System.Collections.Generic.List[string]
foreach ($cmd in @($mdCommands)) {
  $scriptPath = Join-Path $repo.Path $cmd.script
  if (-not (Test-Path -LiteralPath $scriptPath)) { continue }
  if ($cmd.script -match '\.ps1$' -and $cmd.args.Count -gt 0) {
    $surface = & (Join-Path $moduleRoot 'Get-PowerShellSemanticDiff.ps1') -RepoRoot $repo.Path -Files @($cmd.script)
    $paramNames = @($surface | Select-Object -ExpandProperty parameters -ErrorAction SilentlyContinue)
    foreach ($arg in @($cmd.args)) {
      if ($paramNames.Count -gt 0 -and ($paramNames -notcontains $arg)) {
        if (@('NoProfile','ExecutionPolicy','File') -notcontains $arg) {
          $commandDriftReasons.Add("Markdown command may use unsupported script parameter: $($cmd.doc):$($cmd.line) -> -$arg for $($cmd.script)") | Out-Null
        }
      }
    }
  }
}

# Marker quality scan.
$docText = ''
foreach ($doc in @($pathImpact.docChanged)) {
  $docPath = Join-Path $repo.Path $doc
  if (Test-Path -LiteralPath $docPath) { $docText += [Environment]::NewLine + (Get-Content -Raw -LiteralPath $docPath) }
}
$markerCfg = $null
if ($semanticPolicy -and $semanticPolicy.PSObject.Properties.Name -contains 'noDocMarker') { $markerCfg = $semanticPolicy.noDocMarker }
$marker = if ($markerCfg -and $markerCfg.marker) { [string]$markerCfg.marker } else { 'DOCSYNC-NO-DOC-CHANGE' }
$minReason = if ($markerCfg -and $markerCfg.minimumReasonLength) { [int]$markerCfg.minimumReasonLength } else { 20 }
$forbidden = if ($markerCfg -and $markerCfg.forbiddenReasons) { @($markerCfg.forbiddenReasons) } else { @('skip','none','no','n/a','no change') }
$markerResult = & (Join-Path $moduleRoot 'Test-DocSyncNoDocMarker.ps1') -Text $docText -Marker $marker -MinimumReasonLength $minReason -ForbiddenReasons $forbidden

$scoreObj = & (Join-Path $moduleRoot 'Measure-DocDriftScore.ps1') -PathImpact $pathImpact -PowerShellSurface $psSurface -JsonPolicySurface $jsonSurface -MarkdownCommands $mdCommands -SemanticPolicy $semanticPolicy -NoDocMarkerResult $markerResult
$score = [int]$scoreObj.score
$reasons = New-Object System.Collections.Generic.List[string]
foreach ($r in @($scoreObj.reasons)) { $reasons.Add($r) | Out-Null }
foreach ($r in @($commandDriftReasons)) { $reasons.Add($r) | Out-Null; $score += 10 }
if ($score -gt 100) { $score = 100 }

$thresholds = $null
if ($semanticPolicy -and $semanticPolicy.PSObject.Properties.Name -contains 'thresholds') { $thresholds = $semanticPolicy.thresholds }
$warnThreshold = if ($thresholds -and $thresholds.preCommitWarn) { [int]$thresholds.preCommitWarn } else { 10 }
$blockThreshold = 30
if ($Mode -eq 'pre-commit' -and $thresholds -and $thresholds.preCommitBlock) { $blockThreshold = [int]$thresholds.preCommitBlock }
elseif ($Mode -eq 'all' -and $thresholds -and $thresholds.allBlock) { $blockThreshold = [int]$thresholds.allBlock }
elseif ($Mode -eq 'ci' -and $thresholds -and $thresholds.ciBlock) { $blockThreshold = [int]$thresholds.ciBlock }
elseif ($Mode -eq 'release' -and $thresholds -and ($null -ne $thresholds.releaseBlock)) { $blockThreshold = [int]$thresholds.releaseBlock }

$level = 'pass'
$ok = $true
if ($score -gt $blockThreshold) { $level = 'block'; $ok = $false }
elseif ($score -gt $warnThreshold) { $level = 'warning' }

$requiredDocs = @($scoreObj.requiredDocs | Select-Object -Unique)
$semanticChanges = @(@($psSurface) + @($jsonSurface))
$result = [pscustomobject]@{
  ok = $ok
  mode = $Mode
  level = $level
  score = $score
  threshold = $blockThreshold
  changedFiles = @($changed)
  docChanged = @($pathImpact.docChanged)
  codeLikeChanged = @($pathImpact.codeLikeChanged)
  reasons = @($reasons.ToArray())
  requiredDocs = @($requiredDocs)
  semanticChanges = @($semanticChanges)
  marker = $markerResult
}

& (Join-Path $moduleRoot 'New-DocSyncReport.ps1') -Result $result -Format $Format -ReportPath $ReportPath
if (-not $ok) { exit 1 }
exit 0