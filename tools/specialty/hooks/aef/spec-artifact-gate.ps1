[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [string]$Event = "manual",
  [ValidateSet("warn","enforce")][string]$Mode = "warn",
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Hook($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; event=$Event; mode=$Mode; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 40 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 20 }
  if (-not $ok -and $Mode -eq "enforce") { exit 1 }
  exit 0
}

function Get-GitChangedFiles([string]$Root) {
  try {
    $inside = & git -C $Root rev-parse --is-inside-work-tree 2>$null
    if ($LASTEXITCODE -ne 0 -or $inside.Trim() -ne "true") { return @() }
    return @(& git -C $Root status --short 2>$null | ForEach-Object {
      if (-not [string]::IsNullOrWhiteSpace($_)) {
        $p = $_.Substring(3).Trim()
        if ($p.Contains(" -> ")) { $p = ($p -split " -> ")[-1].Trim() }
        $p -replace '\\','/'
      }
    })
  } catch { return @() }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..\..\..")).Path }
  $changed = @(Get-GitChangedFiles $RepoRoot)
  $implChanged = @($changed | Where-Object { $_ -match '^(tools/specialty|config|CodingKit|\.agents|\.github)/' })
  $required = @("docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md", "docs/decisions/plan-mode-overlay/TASKS.md", "docs/decisions/plan-mode-overlay/TRACEABILITY.md")
  $missing = @()
  if ($implChanged.Count -gt 0) {
    foreach ($rel in $required) {
      if (-not (Test-Path -LiteralPath (Join-Path $RepoRoot ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)) -PathType Leaf)) { $missing += $rel }
    }
  }
  $ok = ($missing.Count -eq 0)
  $message = if ($ok) { "Spec artifact 门禁验证通过。" } else { "检测到实现相关文件变更，但缺少计划、任务或可追溯性文档。" }
  $code = if ($ok) { "OK" } else { "SPEC_ARTIFACTS_MISSING" }
  $nextSteps = "请补齐 docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md、docs/decisions/plan-mode-overlay/TASKS.md 和 docs/decisions/plan-mode-overlay/TRACEABILITY.md 后重新运行门禁。"
  $data = @{ changedFiles=$changed; implementationChangedFiles=$implChanged; missing=$missing }
  if (-not $ok) { $data.nextSteps = $nextSteps }
  Out-Hook $ok $code $message $data
}
catch {
  Out-Hook $false "INTERNAL_ERROR" ("Spec artifact 门禁执行时发生内部错误：{0}" -f $_.Exception.Message)
}
