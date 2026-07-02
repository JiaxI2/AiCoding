param(
    [string]$RepoRoot = ".",
    [switch]$DryRun,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$targets = @(
  ".agents/plugins/marketplace.json",
  ".codex/config.toml",
  ".codex/hooks.json",
  ".codex/agents/spec-reviewer.toml",
  ".codex/agents/implementation-planner.toml",
  ".codex/agents/tdd-enforcer.toml",
  ".codex/agents/worktree-coordinator.toml",
  ".codex/agents/systematic-debugger.toml",
  "plugins/aicoding-agent-dev-kit"
)
$removed = @()
foreach ($t in $targets) {
  $path = Join-Path $root $t
  if (Test-Path -LiteralPath $path) {
    if (-not $DryRun) { Remove-Item -LiteralPath $path -Recurse -Force }
    $removed += $t
  }
}
Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  action="uninstall-codex-native-adapter"
  dryRun=[bool]$DryRun
  removed=$removed
}
