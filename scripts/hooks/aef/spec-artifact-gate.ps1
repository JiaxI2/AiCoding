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
  $implChanged = @($changed | Where-Object { $_ -match '^(scripts|config|CodingKit|dist|\.agents|\.github)/' })
  $required = @("spec/IMPLEMENTATION_PLAN.md", "spec/TASKS.md", "spec/TRACEABILITY.md")
  $missing = @()
  if ($implChanged.Count -gt 0) {
    foreach ($rel in $required) {
      if (-not (Test-Path -LiteralPath (Join-Path $RepoRoot ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)) -PathType Leaf)) { $missing += $rel }
    }
  }
  $ok = ($missing.Count -eq 0)
  Out-Hook $ok ($(if ($ok) { "OK" } else { "SPEC_ARTIFACTS_MISSING" })) "Spec artifact gate completed" @{ changedFiles=$changed; implementationChangedFiles=$implChanged; missing=$missing }
}
catch {
  Out-Hook $false "INTERNAL_ERROR" $_.Exception.Message
}
