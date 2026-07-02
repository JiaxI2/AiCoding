param([string]$RepoRoot = ".", [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
Write-AgentDevKitJson -Json:$Json -Data @{
    ok = $true
    repoRoot = $root
    installed = (Test-Path -LiteralPath (Join-Path $root ".agent-dev-kit/install-state.json"))
    hasSpecPack = (Test-Path -LiteralPath (Join-Path $root "spec/IMPLEMENTATION_PLAN.md"))
    hasMemory = (Test-Path -LiteralPath (Join-Path $root ".agent-memory/CURRENT.md"))
    hasHooks = (Test-Path -LiteralPath (Join-Path $root ".githooks/pre-commit"))
    hasWorkflow = (Test-Path -LiteralPath (Join-Path $root ".github/workflows/agent-dev-kit-ci.yml"))
    hasThinSkill = (Test-Path -LiteralPath (Join-Path $root ".agents/skills/aicoding-agent-dev-kit/SKILL.md"))
    hasSubagents = (Test-Path -LiteralPath (Join-Path $root ".agents/subagents/spec-reviewer.md"))
}

# v0.11.1 note: use load-agent-context.ps1 and show-context-manifest.ps1 for sequential loader status.
