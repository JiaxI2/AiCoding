param(
    [string]$RepoRoot = ".",
    [switch]$InstallCodexHooks,
    [switch]$DryRun,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$kitRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$installed = New-Object System.Collections.Generic.List[string]

$items = @(
  ".agents/plugins/marketplace.json",
  ".codex/config.toml",
  ".codex/agents/spec-reviewer.toml",
  ".codex/agents/implementation-planner.toml",
  ".codex/agents/tdd-enforcer.toml",
  ".codex/agents/worktree-coordinator.toml",
  ".codex/agents/systematic-debugger.toml",
  ".codex/agents/requirement-clarifier.toml",
  "plugins/aicoding-agent-dev-kit"
)
if ($InstallCodexHooks) {
  $items += ".codex/hooks.json"
}

foreach ($item in $items) {
  $src = Join-Path $kitRoot $item
  $dst = Join-Path $root $item
  if (Test-Path -LiteralPath $src) {
    if (-not $DryRun) {
      if ((Get-Item -LiteralPath $src).PSIsContainer) {
        New-AgentDevKitDirectory (Split-Path -Parent $dst)
        Copy-Item -LiteralPath $src -Destination $dst -Recurse -Force
      } else {
        New-AgentDevKitDirectory (Split-Path -Parent $dst)
        Copy-Item -LiteralPath $src -Destination $dst -Force
      }
    }
    $installed.Add($item)
  }
}

Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  action="install-codex-native-adapter"
  dryRun=[bool]$DryRun
  installCodexHooks=[bool]$InstallCodexHooks
  installed=$installed.ToArray()
  note= if ($InstallCodexHooks) { "Restart Codex, then review /plugins and /hooks. Non-managed hooks require trust before they run." } else { "Restart Codex, then review /plugins. Project .codex/hooks.json was not installed." }
}
