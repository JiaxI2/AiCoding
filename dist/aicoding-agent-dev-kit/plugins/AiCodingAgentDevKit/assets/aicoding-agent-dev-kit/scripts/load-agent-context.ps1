param(
    [string]$RepoRoot = ".",
    [ValidateSet("L0","L1","L2","L3")]
    [string]$Stage = "L0",
    [switch]$Auto,
    [int]$MaxChars = 0,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot

$scope = $null
if ($Auto) {
  $scopeJson = & "$PSScriptRoot\analyze-change-scope.ps1" -RepoRoot $root -Json
  $scope = $scopeJson | ConvertFrom-Json
  $Stage = $scope.stage
}

$policyPath = Join-Path $root "config/loading-policy.json"
if (-not (Test-Path -LiteralPath $policyPath)) {
  $policyPath = Join-Path (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path "config/loading-policy.json"
}
$policy = Get-Content -Raw -LiteralPath $policyPath | ConvertFrom-Json
$stagePolicy = $policy.stages.$Stage
if (-not $stagePolicy) {
  Write-AgentDevKitJson -Json:$Json -Data @{ ok=$false; error="Unknown stage"; stage=$Stage }
  exit 1
}

if ($MaxChars -le 0) { $MaxChars = [int]$stagePolicy.maxChars }

$outDir = Join-Path $root ".agent-dev-kit/context"
New-AgentDevKitDirectory $outDir
$outPath = Join-Path $outDir "context-pack.md"
$manifestPath = Join-Path $outDir "context-manifest.json"

$changed = git -C $root diff --name-only HEAD
if (-not $changed) {
  $changed = git -C $root status --short | ForEach-Object { ($_ -replace "^\s*\S+\s+","") }
}
$changedFiles = @($changed | Where-Object { $_ -and ($_ -notmatch "^\s*$") } | Sort-Object -Unique)
$backslash = [string][char]92

$paths = New-Object System.Collections.Generic.List[string]
foreach ($f in $stagePolicy.includeFiles) { $paths.Add([string]$f) }
if ($stagePolicy.includeChangedFiles) {
  foreach ($f in $changedFiles) { $paths.Add([string]$f) }
}
if ($stagePolicy.includeGlobs) {
  foreach ($g in $stagePolicy.includeGlobs) {
    Get-ChildItem -Path (Join-Path $root $g) -File -ErrorAction SilentlyContinue | ForEach-Object {
      $paths.Add($_.FullName.Substring($root.Length + 1).Replace($backslash, '/'))
    }
  }
}

$builder = New-Object System.Text.StringBuilder
$fence = ([string][char]96) * 3
$included = New-Object System.Collections.Generic.List[object]
$skipped = New-Object System.Collections.Generic.List[object]
$truncated = New-Object System.Collections.Generic.List[object]
$seen = @{}

[void]$builder.AppendLine("# Agent Context Pack")
[void]$builder.AppendLine("")
[void]$builder.AppendLine("Stage: $Stage")
[void]$builder.AppendLine("Generated: $(Get-Date -Format o)")
if ($Auto -and $scope) { [void]$builder.AppendLine("Auto reason: $($scope.reason)") }
[void]$builder.AppendLine("")

foreach ($rel in $paths) {
  if (-not $rel) { continue }
  $rel = $rel.Replace($backslash, '/')
  if ($seen.ContainsKey($rel)) { continue }
  $seen[$rel] = $true

  $file = Join-Path $root $rel
  if (-not (Test-Path -LiteralPath $file)) {
    $skipped.Add([ordered]@{ path=$rel; reason="not found" })
    continue
  }
  $item = Get-Item -LiteralPath $file
  if ($item.PSIsContainer) {
    $skipped.Add([ordered]@{ path=$rel; reason="directory" })
    continue
  }
  if ($item.Length -gt 1048576) {
    $skipped.Add([ordered]@{ path=$rel; reason="too large"; bytes=$item.Length })
    continue
  }
  try {
    $text = Get-Content -Raw -LiteralPath $file -ErrorAction Stop
  } catch {
    $skipped.Add([ordered]@{ path=$rel; reason="read failed" })
    continue
  }

  $remaining = $MaxChars - $builder.Length
  if ($remaining -le 0) {
    $skipped.Add([ordered]@{ path=$rel; reason="context budget exhausted" })
    continue
  }

  $originalLen = $text.Length
  if ($text.Length -gt $remaining) {
    $text = $text.Substring(0, [Math]::Max(0, $remaining)) + "`n...[truncated]..."
    $truncated.Add([ordered]@{ path=$rel; originalChars=$originalLen; includedChars=$text.Length })
  }

  [void]$builder.AppendLine("## $rel")
  [void]$builder.AppendLine($fence + "text")
  [void]$builder.AppendLine($text)
  [void]$builder.AppendLine($fence)
  [void]$builder.AppendLine("")
  $included.Add([ordered]@{ path=$rel; chars=$text.Length })
}

$builder.ToString() | Set-Content -Encoding UTF8 -LiteralPath $outPath

$reason = "manual stage"
if ($scope) { $reason = $scope.reason }
$manifest = [ordered]@{
  schema="aicoding-agent-dev-kit.context-manifest.v1"
  version="0.8.0"
  generatedAt=(Get-Date).ToString("o")
  repoRoot=$root
  stage=$Stage
  auto=[bool]$Auto
  reason=$reason
  maxChars=$MaxChars
  chars=$builder.Length
  roughTokens=[int]($builder.Length / 4)
  changedFiles=$changedFiles
  includedFiles=$included
  skippedFiles=$skipped
  truncatedFiles=$truncated
  contextPack=$outPath
}
$manifest | ConvertTo-Json -Depth 10 | Set-Content -Encoding UTF8 -LiteralPath $manifestPath

Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  stage=$Stage
  auto=[bool]$Auto
  contextPack=$outPath
  manifest=$manifestPath
  chars=$builder.Length
  roughTokens=[int]($builder.Length / 4)
  includedCount=$included.Count
  skippedCount=$skipped.Count
  truncatedCount=$truncated.Count
}
