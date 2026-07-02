param(
    [string]$RepoRoot = ".",
    [string]$HookFile = "",
    [switch]$MergeExistingHook,
    [switch]$CreateIfMissing,
    [switch]$SetHooksPath,
    [switch]$DryRun,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot

$detectJson = & "$PSScriptRoot\detect-existing-hooks.ps1" -RepoRoot $root -Json
$detect = $detectJson | ConvertFrom-Json

if (-not $HookFile) {
    if ($detect.existingPreCommitHooks -and $detect.existingPreCommitHooks.Count -gt 0) {
        $HookFile = [string]$detect.existingPreCommitHooks[0]
    } else {
        $HookFile = Join-Path $root ".githooks/pre-commit"
    }
}

$exists = Test-Path -LiteralPath $HookFile
if ($exists -and -not $MergeExistingHook) {
    Write-AgentDevKitJson -Json:$Json -Data @{
        ok=$false
        error="Existing hook detected. Re-run with -MergeExistingHook to append a bridge block."
        hookFile=$HookFile
    }
    exit 1
}

if ((-not $exists) -and (-not $CreateIfMissing)) {
    Write-AgentDevKitJson -Json:$Json -Data @{
        ok=$false
        error="No existing hook. Re-run with -CreateIfMissing to create a new hook."
        hookFile=$HookFile
    }
    exit 1
}

$snippet = @'
# BEGIN AICODING_AGENT_DEV_KIT_BRIDGE
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
if [ -f "$repo_root/scripts/invoke-agent-quality-gate.ps1" ]; then
  pwsh -NoProfile -ExecutionPolicy Bypass -File "$repo_root/scripts/invoke-agent-quality-gate.ps1" -Mode pre-commit -Json
  code=$?
  if [ "$code" -ne 0 ]; then
    exit "$code"
  fi
fi
# END AICODING_AGENT_DEV_KIT_BRIDGE
'@

$current = ""
if ($exists) { $current = Get-Content -Raw -LiteralPath $HookFile }

if ($current -match "BEGIN AICODING_AGENT_DEV_KIT_BRIDGE") {
    Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; alreadyInstalled=$true; hookFile=$HookFile }
    exit 0
}

if (-not $DryRun) {
    New-AgentDevKitDirectory (Split-Path -Parent $HookFile)
    if (-not $exists) {
        "#!/usr/bin/env bash`n" | Set-Content -Encoding UTF8 -LiteralPath $HookFile
    }
    Add-Content -Encoding UTF8 -LiteralPath $HookFile -Value ("`n" + $snippet + "`n")
    if ($SetHooksPath) {
        git -C $root config core.hooksPath ".githooks" | Out-Null
    }
}

Write-AgentDevKitJson -Json:$Json -Data @{
    ok=$true
    dryRun=[bool]$DryRun
    action="install-hook-bridge"
    hookFile=$HookFile
    merged=$exists
    created=(-not $exists)
    setHooksPath=[bool]$SetHooksPath
}
