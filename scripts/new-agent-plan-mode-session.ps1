[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [Parameter(Mandatory=$true)][string]$Feature,
  [string]$Description = "",
  [switch]$NeedsDecision,
  [switch]$DryRun,
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 30 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 20 }
  if (-not $ok) { exit 1 }
}

function Safe-Slug([string]$Text) {
  $s = ($Text.ToLowerInvariant() -replace '[^a-z0-9\u4e00-\u9fa5]+','-').Trim('-')
  if ([string]::IsNullOrWhiteSpace($s)) { return "plan-mode-session" }
  if ($s.Length -gt 64) { return $s.Substring(0,64).Trim('-') }
  return $s
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $specDir = Join-Path $RepoRoot "spec"
  $memoryDir = Join-Path $RepoRoot ".agent-memory"

  $planMode = Join-Path $specDir "PLAN_MODE.md"
  $implPlan = Join-Path $specDir "IMPLEMENTATION_PLAN.md"
  $tasks = Join-Path $specDir "TASKS.md"
  $trace = Join-Path $specDir "TRACEABILITY.md"
  $checklist = Join-Path $specDir "CHECKLIST.md"
  $options = Join-Path $specDir "PRD_OPTIONS.md"
  $needs = Join-Path $specDir "NEEDS_USER_DECISION.md"

  $slug = Safe-Slug $Feature
  $now = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss zzz")

  $files = New-Object System.Collections.Generic.List[object]

  $planText = @"
# Plan Mode Session: $Feature

Mode: Plan
Plan Status: Draft
Created: $now
Feature Slug: $slug

## Request

$Description

## Required sequence

1. Clarify ambiguity.
2. Specify intent and constraints.
3. Produce plan.
4. Ask user to select when multiple architecture routes exist.
5. Break selected plan into tasks.
6. Implement only after decision and plan gates pass.
7. Verify with Smoke / schema / golden / doc sync as applicable.

## Current decision state

Decision required: $([bool]$NeedsDecision)

"@

  $implText = @"
# Implementation Plan: $Feature

Plan Status: Draft

## Context

$Description

## Selected architecture

Pending.

## Constraints

- Keep `scripts/aicoding-kit.ps1` as lifecycle entrypoint.
- Do not add `*-all.ps1`.
- Default to Smoke verification.
- Write operations must support DryRun where applicable.
- Hardware actions remain deny-by-default.

## Validation plan

- Smoke verify
- Schema validation
- Hook module validation
- Golden test if behavior is policy-sensitive
- Documentation synchronization

## Rollback

Describe exact rollback command or file removal path before implementation.
"@

  $tasksText = @"
# Tasks: $Feature

## Phase 0: Decision / Plan

- [ ] Confirm whether user selection is required.
- [ ] Record selected option in `spec/SELECTED_SOLUTION.md` and `.agent-memory/DECISIONS.md` if needed.

## Phase 1: Implementation

- [ ] Apply minimal overlay or code change.
- [ ] Keep existing lifecycle entrypoints.

## Phase 2: Verification

- [ ] Run Smoke verify.
- [ ] Run plan-mode gate.
- [ ] Run hook verification.
- [ ] Run `git diff --check`.

## Phase 3: Handoff

- [ ] Summarize implemented changes.
- [ ] Summarize verification.
- [ ] Summarize rollback.
"@

  $traceText = @"
# Traceability: $Feature

| Requirement / Decision | Plan section | Task | Verification |
|---|---|---|---|
| Pending | Pending | Pending | Pending |
"@

  $checkText = @"
# Checklist: $Feature

- [ ] No unresolved `[NEEDS CLARIFICATION]` markers remain.
- [ ] If architecture was fuzzy, user selection is recorded.
- [ ] Implementation plan is approved before code changes.
- [ ] Tasks include verification and rollback.
- [ ] Handoff includes verified / not verified / rollback.
"@

  $optionText = @"
# PRD Options: $Feature

Decision Status: Pending User Selection

## Context

$Description

## Options

### Option A: Minimal incremental extension

- Fit:
- Impact:
- Verification:
- Rollback:
- Risk:

### Option B: Registry-backed extension

- Fit:
- Impact:
- Verification:
- Rollback:
- Risk:

### Option C: Full plugin/kit extension

- Fit:
- Impact:
- Verification:
- Rollback:
- Risk:

## User selection required

Do not implement until the user selects one option.
"@

  $needsText = @"
# Needs User Decision

Feature: $Feature
Created: $now

The Agent detected ambiguous architecture or multiple viable implementation routes.

Required action: user must choose one option in `spec/PRD_OPTIONS.md`.

After selection, run:

Command:
  pwsh scripts\confirm-agent-decision.ps1 -Title "$Feature" -SelectedOption "<chosen option>" -Rationale "<why>" -Json
"@

  $planned = @(
    @{ path=$planMode; content=$planText },
    @{ path=$implPlan; content=$implText },
    @{ path=$tasks; content=$tasksText },
    @{ path=$trace; content=$traceText },
    @{ path=$checklist; content=$checkText }
  )
  if ($NeedsDecision) {
    $planned += @{ path=$options; content=$optionText }
    $planned += @{ path=$needs; content=$needsText }
  }

  foreach ($item in $planned) {
    $rel = Resolve-Path -LiteralPath (Split-Path -Parent $item.path) -ErrorAction SilentlyContinue
    $files.Add([ordered]@{ path=$item.path; willWrite=(-not $DryRun) }) | Out-Null
    if (-not $DryRun) {
      $dir = Split-Path -Parent $item.path
      if (-not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
      Set-Content -LiteralPath $item.path -Value $item.content -Encoding UTF8
    }
  }

  if (-not $DryRun -and -not (Test-Path -LiteralPath $memoryDir)) { New-Item -ItemType Directory -Force -Path $memoryDir | Out-Null }

  Out-Result $true "OK" ($(if ($DryRun) { "Plan mode session dry-run completed" } else { "Plan mode session created" })) ([ordered]@{
    repoRoot=$RepoRoot
    feature=$Feature
    needsDecision=[bool]$NeedsDecision
    files=@($files)
  })
}
catch {
  Out-Result $false "INTERNAL_ERROR" $_.Exception.Message
}
