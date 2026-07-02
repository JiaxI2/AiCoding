param(
    [string]$RepoRoot = ".",
    [Parameter(Mandatory=$true)][string]$OptionId,
    [string]$OptionName = "",
    [string]$Reason = "",
    [switch]$CreateAdr,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
New-AgentDevKitDirectory (Join-Path $root "spec")
$selected = Join-Path $root "spec/SELECTED_SOLUTION.md"
$date = Get-Date -Format yyyy-MM-dd
$selectedText = @"
# Selected Solution

## Selected Option

- Option ID: $OptionId
- Option Name: $OptionName
- Status: Accepted

## Human Decision

- Decision owner: human
- Date: $date
- Reason: $Reason

## Final Architecture

TBD. Update this section from the selected option.

## Final Flow

Start -> TBD -> Done

## Impacts

- PRD: review required
- APP_FLOW: review required
- TECH_STACK: review if architecture changes
- IMPLEMENTATION_PLAN: update required
- ADR: $CreateAdr
- Tests: update required

## Required Document Sync

When this file changes, update the relevant official docs:

- spec/PRD.md
- spec/APP_FLOW.md
- spec/IMPLEMENTATION_PLAN.md
- docs/adr/*.md
- docs/traceability/TRACEABILITY_MATRIX.md
"@
$selectedText | Set-Content -Encoding UTF8 -LiteralPath $selected

& "$PSScriptRoot\decision-add.ps1" -RepoRoot $root -Type human -Title "Select solution option $OptionId" -Decision "Selected $OptionId $OptionName. $Reason" -Context "Requirement clarification option selection" -Impact "PRD, app flow, implementation plan, ADR and tests must align with selected solution." -Link "spec/SELECTED_SOLUTION.md" -Json | Out-Null

$adrPath = ""
if ($CreateAdr) {
    New-AgentDevKitDirectory (Join-Path $root "docs/adr")
    $slug = ($OptionName.ToLowerInvariant() -replace "[^a-z0-9]+","-").Trim("-")
    if (-not $slug) { $slug = $OptionId.ToLowerInvariant() }
    $adrPath = Join-Path $root ("docs/adr/adr-" + $OptionId.ToLowerInvariant() + "-" + $slug + ".md")
    $adrText = @"
# ADR: Select $OptionId $OptionName

## Status

Accepted

## Context

The requirement had multiple viable implementation options.

## Decision

Select $OptionId $OptionName.

Reason: $Reason

## Consequences

- PRD must reflect the selected behavior.
- APP_FLOW must reflect the selected runtime flow.
- IMPLEMENTATION_PLAN must split the selected solution into small TDD tasks.
- Tests must validate the selected flow.

## Linked Artifacts

- spec/PRD_OPTIONS.md
- spec/SELECTED_SOLUTION.md
- spec/IMPLEMENTATION_PLAN.md
"@
    $adrText | Set-Content -Encoding UTF8 -LiteralPath $adrPath
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; selectedSolution=$selected; optionId=$OptionId; optionName=$OptionName; adr=$adrPath }