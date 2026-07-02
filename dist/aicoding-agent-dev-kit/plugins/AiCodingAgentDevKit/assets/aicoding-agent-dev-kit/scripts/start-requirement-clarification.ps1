param(
    [string]$RepoRoot = ".",
    [string]$Requirement = "",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
New-AgentDevKitDirectory (Join-Path $root "spec")
$path = Join-Path $root "spec/PRD_OPTIONS.md"
if (-not (Test-Path -LiteralPath $path)) {
@"
# PRD Options and Solution Matrix

## Requirement Question

$Requirement

## Constraints

- Safety:
- Compatibility:
- Resource limits:
- Toolchain:
- Migration:
- Rollback:
- Testing:

## Options

| Option ID | Name | Architecture | Flow | Pros | Cons | Risks | Validation | Effort | Recommended When |
|---|---|---|---|---|---|---|---|---|---|
| OPT-001 | Option A | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |
| OPT-002 | Option B | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |
| OPT-003 | Option C | TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD |

## Agent Recommendation

- Recommended option:
- Why:
- What must be verified before implementation:
- What user decision is required:

## Human Selection

- Selected Option:
- Selected By:
- Date:
- Notes:
"@ | Set-Content -Encoding UTF8 -LiteralPath $path
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; action="start-requirement-clarification"; path=$path; requirement=$Requirement }
