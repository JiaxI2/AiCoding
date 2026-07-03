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
# 计划模式会话（Plan Mode Session）：$Feature

Mode: Plan
Plan Status: Draft
Created: $now
Feature Slug: $slug

## 需求 / Request

$Description

## 必须执行的顺序

1. 澄清模糊点。
2. 明确用户意图和约束。
3. 生成实现计划。
4. 如果存在多条架构路线，先请求用户选择。
5. 将已选择计划拆分为任务。
6. 只有在决策和计划门禁通过后才能实现。
7. 按需要执行 Smoke / schema / golden / doc sync 验证。

## 当前决策状态

需要用户决策：$([bool]$NeedsDecision)

"@

  $implText = @"
# 实现计划（Implementation Plan）：$Feature

Plan Status: Draft

## 上下文

$Description

## 已选择架构

待用户选择。

## 约束

- 保留 `scripts/aicoding-kit.ps1` 作为 lifecycle 入口。
- 不新增 `*-all.ps1`。
- 默认使用 Smoke 验证。
- 写操作应在适用时支持 DryRun。
- 硬件动作默认拒绝，禁止 flash/reset/halt/run/loadProgram/erase/write-memory 等危险动作。

## 验证计划

- Smoke 验证。
- Schema 验证。
- Hook module 验证。
- 若行为涉及策略，执行 golden test。
- 文档同步验证。

## 回滚

实现前必须写明准确的回滚命令或文件移除路径。
"@

  $tasksText = @"
# 任务（Tasks）：$Feature

## Phase 0: 决策 / 计划

- [ ] 确认是否需要用户选择技术路线。
- [ ] 如需要，将用户选择记录到 `spec/SELECTED_SOLUTION.md` 和 `.agent-memory/DECISIONS.md`。

## Phase 1: 实现

- [ ] 应用最小 overlay 或代码变更。
- [ ] 保持现有 lifecycle 入口。

## Phase 2: 验证

- [ ] 运行 Smoke 验证。
- [ ] 运行 Plan Mode 门禁。
- [ ] 运行 hook 验证。
- [ ] 运行 `git diff --check`。

## Phase 3: 交接

- [ ] 总结已实现变更。
- [ ] 总结已验证内容。
- [ ] 总结回滚方法。
"@

  $traceText = @"
# 可追溯性（Traceability）：$Feature

| 需求 / 决策 | 计划章节 | 任务 | 验证 |
|---|---|---|---|
| 待补充 | 待补充 | 待补充 | 待补充 |
"@

  $checkText = @"
# 检查清单（Checklist）：$Feature

- [ ] 不再存在未解决的 `[NEEDS CLARIFICATION]` 标记。
- [ ] 如果架构路线模糊，已记录用户选择。
- [ ] 代码变更前实现计划已批准。
- [ ] 任务包含验证和回滚。
- [ ] 交接包含已验证 / 未验证 / 回滚。
"@

  $optionText = @"
# PRD 选项（PRD Options）：$Feature

Decision Status: Pending User Selection

## 上下文

$Description

## 选项

### Option A: 最小增量扩展

- 适用性：
- 影响：
- 验证：
- 回滚：
- 风险：

### Option B: registry 管理的扩展

- 适用性：
- 影响：
- 验证：
- 回滚：
- 风险：

### Option C: 完整 plugin/kit 扩展

- 适用性：
- 影响：
- 验证：
- 回滚：
- 风险：

## 需要用户选择

用户选择技术路线前，不允许继续实现。
"@

  $needsText = @"
# 需要用户决策（Needs User Decision）

Feature: $Feature
Created: $now

Agent 检测到架构路线存在歧义，或存在多条可行实现路径。

需要用户操作：请从 `spec/PRD_OPTIONS.md` 中选择一个技术路线。

用户选择后执行：

Command:
  pwsh scripts\confirm-agent-decision.ps1 -Title "$Feature" -SelectedOption "<用户选择的方案>" -Rationale "<选择理由>" -Json
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
    $files.Add([ordered]@{ path=$item.path; willWrite=(-not $DryRun) }) | Out-Null
    if (-not $DryRun) {
      $dir = Split-Path -Parent $item.path
      if (-not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
      Set-Content -LiteralPath $item.path -Value $item.content -Encoding UTF8
    }
  }

  if (-not $DryRun -and -not (Test-Path -LiteralPath $memoryDir)) { New-Item -ItemType Directory -Force -Path $memoryDir | Out-Null }

  $message = if ($DryRun) { "Plan Mode 会话 dry-run 完成，未写入文件。" } else { "Plan Mode 会话已创建。" }
  Out-Result $true "OK" $message ([ordered]@{
    repoRoot=$RepoRoot
    feature=$Feature
    needsDecision=[bool]$NeedsDecision
    files=@($files.ToArray())
  })
}
catch {
  Out-Result $false "INTERNAL_ERROR" ("创建 Plan Mode 会话时发生内部错误：{0}" -f $_.Exception.Message)
}
