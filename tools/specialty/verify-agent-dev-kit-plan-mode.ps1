[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$AllowPendingDecision
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 80 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 30; foreach ($w in $warnings) { Write-Warning $w } }
  if (-not $ok) { exit 1 }
}

function Test-PowerShellSyntax([string]$Path) {
  [System.Management.Automation.Language.Token[]]$tokens = $null
  [System.Management.Automation.Language.ParseError[]]$parseErrors = $null
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

function Test-ZhFirstLanguagePolicy([string]$Root) {
  $targets = @(
    "AGENTS.md",
    "docs/governance/AGENT_LANGUAGE_POLICY.md",
    "docs/decisions/AGENT_DEV_KIT_PLAN_MODE.md",
    ".agents/skills/aicoding-agent-dev-kit-plan-mode/prompts/apply-devkit-plan-mode-overlay.prompt.md",
    "tools/migration/aicoding-agent-dev-kit.SKILL.insert.md",
    "tools/specialty/new-agent-plan-mode-session.ps1",
    "tools/specialty/verify-agent-dev-kit-plan-mode.ps1",
    "tools/specialty/hooks/aef/plan-mode-gate.ps1",
    "tools/specialty/hooks/aef/spec-artifact-gate.ps1"
  )
  $patterns = @(
    ("Allow " + "reading"),
    ("before " + "validation"),
    ("Validation " + "passed"),
    ("Missing " + "selected"),
    ("Implementation " + "blocked"),
    ("Architecture-" + "sensitive"),
    ("without " + "accepted decision"),
    ("No " + "selected decision")
  )
  $findings = New-Object System.Collections.Generic.List[object]
  foreach ($rel in $targets) {
    $path = Join-Path $Root ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { continue }
    $lines = Get-Content -LiteralPath $path -Encoding UTF8
    for ($i = 0; $i -lt $lines.Count; $i++) {
      $line = [string]$lines[$i]
      foreach ($pattern in $patterns) {
        if ($line.Contains($pattern)) {
          $severity = if ($rel -like "tools/specialty/*.ps1" -or $rel -like "tools/specialty/hooks/*.ps1") { "error" } else { "warning" }
          $findings.Add([ordered]@{ path=$rel; line=($i + 1); pattern=$pattern; severity=$severity; text=$line.Trim() }) | Out-Null
        }
      }
    }
  }
  return @($findings.ToArray())
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

  Add-Check "registry.exists" (Test-Path -LiteralPath $registryPath -PathType Leaf) "找到 Plan Mode registry。"
  if (Test-Path -LiteralPath $registryPath -PathType Leaf) {
    try {
      $registry = Get-Content -LiteralPath $registryPath -Raw -Encoding UTF8 | ConvertFrom-Json
      Add-Check "registry.parse" $true "Plan Mode registry 解析通过。"
      Add-Check "registry.languagePolicy" ($registry.PSObject.Properties.Name -contains "languagePolicy") "Plan Mode registry 已声明中文优先 languagePolicy。"
    } catch {
      Add-Check "registry.parse" $false ("Plan Mode registry 解析失败：{0}" -f $_.Exception.Message)
    }
  }
  Add-Check "schema.exists" (Test-Path -LiteralPath $schemaPath -PathType Leaf) "找到 Plan Mode registry schema。"
  if (Test-Path -LiteralPath $schemaPath -PathType Leaf) {
    try { Get-Content -LiteralPath $schemaPath -Raw -Encoding UTF8 | ConvertFrom-Json | Out-Null; Add-Check "schema.parse" $true "Plan Mode registry schema 解析通过。" }
    catch { Add-Check "schema.parse" $false ("Plan Mode registry schema 解析失败：{0}" -f $_.Exception.Message) }
  }

  $requiredFiles = @(
    "docs/governance/AGENT_LANGUAGE_POLICY.md",
    "docs/decisions/AGENT_DEV_KIT_PLAN_MODE.md",
    "docs/decisions/SPEC_KIT_ADAPTATION.md",
    "docs/decisions/SUPERPOWER_SKILL_ADAPTATION.md",
    ".agents/skills/aicoding-agent-dev-kit-plan-mode/prompts/apply-devkit-plan-mode-overlay.prompt.md",
    "tools/migration/aicoding-agent-dev-kit.SKILL.insert.md",
    "tools/specialty/new-agent-plan-mode-session.ps1",
    "tools/specialty/confirm-agent-decision.ps1",
    "tools/specialty/verify-agent-dev-kit-plan-mode.ps1",
    "tools/specialty/hooks/aef/plan-mode-gate.ps1",
    "tools/specialty/hooks/aef/spec-artifact-gate.ps1"
  )
  foreach ($rel in $requiredFiles) {
    $path = Join-Path $RepoRoot ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)
    Add-Check "required:$rel" (Test-Path -LiteralPath $path -PathType Leaf) ("检查必要文件：{0}" -f $rel)
    if ((Test-Path -LiteralPath $path -PathType Leaf) -and $rel.EndsWith(".ps1")) {
      Add-Check "syntax:$rel" (Test-PowerShellSyntax $path) ("PowerShell 语法检查：{0}" -f $rel)
    }
  }

  $languageFindings = @(Test-ZhFirstLanguagePolicy $RepoRoot)
  $languageErrors = @($languageFindings | Where-Object { $_.severity -eq "error" })
  $languageWarnings = @($languageFindings | Where-Object { $_.severity -eq "warning" })
  Add-Check "language.zh-first" ($languageErrors.Count -eq 0) "中文优先轻量扫描完成，脚本用户提示未发现明显英文摘要。" @{ findings=$languageFindings }
  if ($languageWarnings.Count -gt 0) {
    $warnings.Add(("中文优先轻量扫描发现文档提示项，请人工确认：{0}" -f ($languageWarnings.Count))) | Out-Null
  }

  $needsPath = Join-Path $RepoRoot "docs/decisions/plan-mode-overlay/NEEDS_USER_DECISION.md"
  $hasPendingDecision = Test-Path -LiteralPath $needsPath -PathType Leaf
  Add-Check "decision.pending" ((-not $hasPendingDecision) -or $AllowPendingDecision) "检测到 docs/decisions/plan-mode-overlay/NEEDS_USER_DECISION.md，用户尚未选择技术路线，禁止继续实现。" @{ exists=$hasPendingDecision }

  $changed = @(Get-GitChangedFiles $RepoRoot)
  $sensitive = @()
  $patterns = @()
  if ($registry -and $registry.architectureSensitivePaths) { $patterns = @($registry.architectureSensitivePaths) }
  foreach ($file in $changed) {
    if (Match-AnyPath $file $patterns) { $sensitive += $file }
  }

  $decisionText = ""
  foreach ($rel in @("docs/decisions/plan-mode-overlay/SELECTED_SOLUTION.md", ".aicoding/memory/DECISIONS.md")) {
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

  $planText = Read-TextIfExists (Join-Path $RepoRoot "docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md")
  $planMarkers = if ($registry -and $registry.requiredMarkers.approvedPlan) { @($registry.requiredMarkers.approvedPlan) } else { @("Plan Status: Approved","Status: Accepted") }
  $hasPlan = -not [string]::IsNullOrWhiteSpace($planText)
  $hasApprovedPlan = Has-AnyMarker $planText $planMarkers

  $hasTasks = Test-Path -LiteralPath (Join-Path $RepoRoot "docs/decisions/plan-mode-overlay/TASKS.md") -PathType Leaf
  $hasTrace = Test-Path -LiteralPath (Join-Path $RepoRoot "docs/decisions/plan-mode-overlay/TRACEABILITY.md") -PathType Leaf

  if ($sensitive.Count -gt 0) {
    Add-Check "architecture.decision" $hasSelectedDecision "检测到架构敏感文件变更，但未找到已接受的用户决策记录。" @{ sensitive=$sensitive }
    Add-Check "implementation.plan" $hasPlan "架构敏感变更需要 docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md。" @{ approved=$hasApprovedPlan }
    Add-Check "implementation.tasks" $hasTasks "架构敏感变更需要 docs/decisions/plan-mode-overlay/TASKS.md。"
    Add-Check "implementation.traceability" $hasTrace "架构敏感变更需要 docs/decisions/plan-mode-overlay/TRACEABILITY.md。"
  } else {
    Add-Check "architecture.changed" $true "未检测到架构敏感文件变更。" @{ changed=$changed }
  }

  if (@($changed | Where-Object { $_ -eq "docs/decisions/plan-mode-overlay/PRD_OPTIONS.md" }).Count -gt 0) {
    Add-Check "options.selected" $hasSelectedDecision "PRD_OPTIONS 已变更，需要先记录用户选择的技术路线。"
  }

  $ok = ($errors.Count -eq 0)
  $code = if ($ok) { "OK" } else { "PLAN_MODE_GATE_FAILED" }
  $message = if ($ok) { "AiCoding Agent Dev Kit Plan Mode 验证通过。" } else { "AiCoding Agent Dev Kit Plan Mode 验证未通过。" }
  Out-Result $ok $code $message ([ordered]@{
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
    languageFindings=$languageFindings
    errors=@($errors.ToArray())
  }) @($warnings.ToArray())
}
catch {
  Out-Result $false "INTERNAL_ERROR" ("Plan Mode 验证脚本内部错误：{0}" -f $_.Exception.Message) ([ordered]@{ scriptStackTrace = $_.ScriptStackTrace; category = $_.CategoryInfo.ToString() })
}
