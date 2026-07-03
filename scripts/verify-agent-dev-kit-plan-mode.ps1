[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$AllowPendingDecision
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 60 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 30; foreach ($w in $warnings) { Write-Warning $w } }
  if (-not $ok) { exit 1 }
}

function Test-PowerShellSyntax([string]$Path) {
  $tokens = $null
  $parseErrors = $null
  [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$parseErrors) | Out-Null
  return @($parseErrors).Count -eq 0
}

function Read-TextIfExists([string]$Path) {
  if (Test-Path -LiteralPath $Path -PathType Leaf) { return (Get-Content -LiteralPath $Path -Raw -Encoding UTF8) }
  return ""
}

function Has-AnyMarker([string]$Text, [string[]]$Markers) {
  foreach ($m in $Markers) {
    if ($Text -like "*$m*") { return $true }
  }
  return $false
}

function Get-GitChangedFiles([string]$Root) {
  $result = @()
  try {
    $inside = & git -C $Root rev-parse --is-inside-work-tree 2>$null
    if ($LASTEXITCODE -ne 0 -or $inside.Trim() -ne "true") { return @() }
    $lines = & git -C $Root status --short 2>$null
    foreach ($line in @($lines)) {
      if ([string]::IsNullOrWhiteSpace($line)) { continue }
      $path = $line.Substring(3).Trim()
      if ($path.Contains(" -> ")) { $path = ($path -split " -> ")[-1].Trim() }
      $result += ($path -replace '\\','/')
    }
  } catch {}
  return @($result)
}

function Match-AnyPath([string]$Path, [string[]]$Patterns) {
  $p = $Path -replace '\\','/'
  foreach ($raw in $Patterns) {
    $pattern = ($raw -replace '\\','/')
    if ($pattern.EndsWith('/')) {
      if ($p.StartsWith($pattern)) { return $true }
    } elseif ($pattern.Contains('*')) {
      $regex = '^' + [Regex]::Escape($pattern).Replace('\*','.*') + '$'
      if ($p -match $regex) { return $true }
    } else {
      if ($p -eq $pattern -or $p.StartsWith($pattern)) { return $true }
    }
  }
  return $false
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $checks = New-Object System.Collections.Generic.List[object]
  $errors = New-Object System.Collections.Generic.List[string]
  $warnings = New-Object System.Collections.Generic.List[string]

  function Add-Check([string]$Name, [bool]$Ok, [string]$Message, $Data = $null, [bool]$WarningOnly = $false) {
    $checks.Add([ordered]@{ name=$Name; ok=$Ok; message=$Message; data=$Data; warningOnly=$WarningOnly }) | Out-Null
    if (-not $Ok) {
      if ($WarningOnly) { $warnings.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null }
      else { $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null }
    }
  }

  $registryPath = Join-Path $RepoRoot "config/agent-dev-kit-plan-mode.registry.json"
  $schemaPath = Join-Path $RepoRoot "config/schemas/agent-dev-kit-plan-mode.registry.schema.json"
  $registry = $null

  Add-Check "registry.exists" (Test-Path -LiteralPath $registryPath -PathType Leaf) "config/agent-dev-kit-plan-mode.registry.json"
  if (Test-Path -LiteralPath $registryPath -PathType Leaf) {
    try {
      $registry = Get-Content -LiteralPath $registryPath -Raw -Encoding UTF8 | ConvertFrom-Json
      Add-Check "registry.parse" $true "parsed"
    } catch {
      Add-Check "registry.parse" $false $_.Exception.Message
    }
  }
  Add-Check "schema.exists" (Test-Path -LiteralPath $schemaPath -PathType Leaf) "config/schemas/agent-dev-kit-plan-mode.registry.schema.json"
  if (Test-Path -LiteralPath $schemaPath -PathType Leaf) {
    try { Get-Content -LiteralPath $schemaPath -Raw -Encoding UTF8 | ConvertFrom-Json | Out-Null; Add-Check "schema.parse" $true "parsed" }
    catch { Add-Check "schema.parse" $false $_.Exception.Message }
  }

  $requiredFiles = @(
    "docs/AGENT_DEV_KIT_PLAN_MODE.md",
    "docs/SPEC_KIT_ADAPTATION.md",
    "docs/SUPERPOWER_SKILL_ADAPTATION.md",
    "scripts/new-agent-plan-mode-session.ps1",
    "scripts/verify-agent-dev-kit-plan-mode.ps1",
    "scripts/hooks/aef/plan-mode-gate.ps1",
    "scripts/hooks/aef/spec-artifact-gate.ps1"
  )
  foreach ($rel in $requiredFiles) {
    $path = Join-Path $RepoRoot ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)
    Add-Check "required:$rel" (Test-Path -LiteralPath $path -PathType Leaf) $rel
    if ((Test-Path -LiteralPath $path -PathType Leaf) -and $rel.EndsWith(".ps1")) {
      Add-Check "syntax:$rel" (Test-PowerShellSyntax $path) "PowerShell AST parser"
    }
  }

  $needsPath = Join-Path $RepoRoot "spec/NEEDS_USER_DECISION.md"
  $hasPendingDecision = Test-Path -LiteralPath $needsPath -PathType Leaf
  Add-Check "decision.pending" ((-not $hasPendingDecision) -or $AllowPendingDecision) "spec/NEEDS_USER_DECISION.md blocks implementation" @{ exists=$hasPendingDecision }

  $changed = @(Get-GitChangedFiles $RepoRoot)
  $sensitive = @()
  $patterns = @()
  if ($registry -and $registry.architectureSensitivePaths) { $patterns = @($registry.architectureSensitivePaths) }
  foreach ($file in $changed) {
    if (Match-AnyPath $file $patterns) { $sensitive += $file }
  }

  $decisionText = ""
  foreach ($rel in @("spec/SELECTED_SOLUTION.md", ".agent-memory/DECISIONS.md")) {
    $decisionText += "`n" + (Read-TextIfExists (Join-Path $RepoRoot ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)))
  }
  foreach ($dirRel in @("docs/decisions", "docs/adr")) {
    $dir = Join-Path $RepoRoot ($dirRel -replace '/', [IO.Path]::DirectorySeparatorChar)
    if (Test-Path -LiteralPath $dir -PathType Container) {
      Get-ChildItem -LiteralPath $dir -File -Filter "*.md" -ErrorAction SilentlyContinue | ForEach-Object { $decisionText += "`n" + (Get-Content -LiteralPath $_.FullName -Raw -Encoding UTF8) }
    }
  }
  $selectedMarkers = if ($registry -and $registry.requiredMarkers.selectedDecision) { @($registry.requiredMarkers.selectedDecision) } else { @("Decision Status: Selected","Status: Accepted","Selected option:") }
  $hasSelectedDecision = Has-AnyMarker $decisionText $selectedMarkers

  $planText = Read-TextIfExists (Join-Path $RepoRoot "spec/IMPLEMENTATION_PLAN.md")
  $planMarkers = if ($registry -and $registry.requiredMarkers.approvedPlan) { @($registry.requiredMarkers.approvedPlan) } else { @("Plan Status: Approved","Status: Accepted") }
  $hasPlan = -not [string]::IsNullOrWhiteSpace($planText)
  $hasApprovedPlan = Has-AnyMarker $planText $planMarkers

  $hasTasks = Test-Path -LiteralPath (Join-Path $RepoRoot "spec/TASKS.md") -PathType Leaf
  $hasTrace = Test-Path -LiteralPath (Join-Path $RepoRoot "spec/TRACEABILITY.md") -PathType Leaf

  if ($sensitive.Count -gt 0) {
    Add-Check "architecture.decision" $hasSelectedDecision "architecture-sensitive changes require selected decision record" @{ sensitive=$sensitive }
    Add-Check "implementation.plan" $hasPlan "architecture-sensitive changes require spec/IMPLEMENTATION_PLAN.md" @{ approved=$hasApprovedPlan }
    Add-Check "implementation.tasks" $hasTasks "architecture-sensitive changes require spec/TASKS.md"
    Add-Check "implementation.traceability" $hasTrace "architecture-sensitive changes require spec/TRACEABILITY.md"
  } else {
    Add-Check "architecture.changed" $true "no architecture-sensitive changed files detected" @{ changed=$changed }
  }

  if (@($changed | Where-Object { $_ -eq "spec/PRD_OPTIONS.md" }).Count -gt 0) {
    Add-Check "options.selected" $hasSelectedDecision "changed PRD_OPTIONS requires selected solution"
  }

  $ok = ($errors.Count -eq 0)
  $code = if ($ok) { "OK" } else { "PLAN_MODE_GATE_FAILED" }
  Out-Result $ok $code "AiCoding Agent Dev Kit Plan Mode verification completed" ([ordered]@{
    repoRoot=$RepoRoot
    checks=@($checks.ToArray())
    changedFiles=$changed
    architectureSensitiveChangedFiles=$sensitive
    hasPendingDecision=$hasPendingDecision
    hasSelectedDecision=$hasSelectedDecision
    hasPlan=$hasPlan
    hasApprovedPlan=$hasApprovedPlan
    hasTasks=$hasTasks
    hasTraceability=$hasTrace
    errors=@($errors.ToArray())
  }) @($warnings.ToArray())
}
catch {
  Out-Result $false "INTERNAL_ERROR" $_.Exception.Message ([ordered]@{ scriptStackTrace = $_.ScriptStackTrace; category = $_.CategoryInfo.ToString() })
}
