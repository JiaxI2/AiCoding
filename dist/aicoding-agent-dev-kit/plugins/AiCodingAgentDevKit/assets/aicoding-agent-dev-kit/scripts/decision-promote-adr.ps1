param(
    [string]$RepoRoot = ".",
    [Parameter(Mandatory=$true)][string]$DecisionId,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$decPath = Join-Path $root ".agent-memory/DECISIONS.md"
$adrDir = Join-Path $root "docs/adr"
New-AgentDevKitDirectory $adrDir
if (-not (Test-Path -LiteralPath $decPath)) {
  Write-AgentDevKitJson -Json:$Json -Data @{ ok=$false; error="DECISIONS.md not found" }
  exit 1
}
$text = Get-Content -Raw -LiteralPath $decPath
$pattern = "(?ms)^##\s+$([regex]::Escape($DecisionId)):\s+(.+?)(?=^##\s+D-\d{4}:|\z)"
$m = [regex]::Match($text, $pattern)
if (-not $m.Success) {
  Write-AgentDevKitJson -Json:$Json -Data @{ ok=$false; error="Decision not found"; id=$DecisionId }
  exit 1
}
$title = ($m.Groups[1].Value -split "`n")[0].Trim()
$slug = ($title.ToLowerInvariant() -replace "[^a-z0-9]+","-").Trim("-")
if (-not $slug) { $slug = $DecisionId.ToLowerInvariant() }
$adrPath = Join-Path $adrDir ("adr-" + $DecisionId.ToLowerInvariant().Replace("d-","") + "-" + $slug + ".md")
$body = @"
# ADR: $title

## Status

Proposed

## Source Decision

$DecisionId

## Context

Promoted from `.agent-memory/DECISIONS.md`.

## Decision

$m

## Consequences

TBD.

## Enforcement

TBD.
"@
$body | Set-Content -Encoding UTF8 -LiteralPath $adrPath
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; id=$DecisionId; adr=$adrPath }
