param([string]$RepoRoot = ".", [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$required = @("spec/PRD.md","spec/APP_FLOW.md","spec/TECH_STACK.md","spec/CODING_GUIDELINES.md","spec/PROJECT_STRUCTURE.md","spec/IMPLEMENTATION_PLAN.md","spec/TEST_STRATEGY.md")
$missing = @()
foreach ($r in $required) { if (-not (Test-Path -LiteralPath (Join-Path $root $r))) { $missing += $r } }
Write-AgentDevKitJson -Json:$Json -Data @{ ok = ($missing.Count -eq 0); validator = "validate-spec-pack"; missing = $missing }
if ($missing.Count -gt 0) { exit 1 }
