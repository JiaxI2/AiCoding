param(
    [string]$RepoRoot = ".",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$required = @(
  ".agents/plugins/marketplace.json",
  "plugins/aicoding-agent-dev-kit/.codex-plugin/plugin.json",
  "plugins/aicoding-agent-dev-kit/skills/aicoding-agent-dev-kit/SKILL.md",
  "plugins/aicoding-agent-dev-kit/hooks/hooks.json",
  ".codex/config.toml",
  ".codex/agents/spec-reviewer.toml",
  ".codex/agents/implementation-planner.toml",
  ".codex/agents/tdd-enforcer.toml",
  ".codex/agents/worktree-coordinator.toml",
  ".codex/agents/systematic-debugger.toml",
  ".codex/agents/requirement-clarifier.toml"
)
$missing = @()
foreach ($r in $required) {
  if (-not (Test-Path -LiteralPath (Join-Path $root $r))) { $missing += $r }
}
Write-AgentDevKitJson -Json:$Json -Data @{
  ok=($missing.Count -eq 0)
  missing=$missing
  adapter="codex-native"
  version="0.11.1"
  hasProjectCodexHooks=(Test-Path -LiteralPath (Join-Path $root ".codex/hooks.json"))
}
if ($missing.Count -gt 0) { exit 1 }
