param(
    [string]$RepoRoot = ".",
    [switch]$InstallSpecPack,
    [switch]$InstallMemory,
    [switch]$InstallHooks,
    [switch]$InstallHookBridge,
    [switch]$ForceHookOverwrite,
    [switch]$InstallWorkflow,
    [switch]$InstallThinSkill,
    [switch]$InstallSubagents,
    [switch]$DryRun,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$kitRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$files = New-Object System.Collections.Generic.List[string]

$baseDirs = @("scripts","config","docs/adr","docs/traceability","specs/bdd","specs/tdd")
foreach ($d in $baseDirs) { if (-not $DryRun) { New-AgentDevKitDirectory (Join-Path $root $d) } }

if ($InstallSpecPack) {
    if (-not $DryRun) { Copy-AgentDevKitTree (Join-Path $kitRoot "spec") (Join-Path $root "spec") }
    $files.Add("spec")
}
if ($InstallMemory) {
    if (-not $DryRun) { Copy-AgentDevKitTree (Join-Path $kitRoot ".agent-memory") (Join-Path $root ".agent-memory") }
    $files.Add(".agent-memory/CURRENT.md")
    $files.Add(".agent-memory/DECISIONS.md")
}
if ($InstallSubagents) {
    if (-not $DryRun) { Copy-AgentDevKitTree (Join-Path $kitRoot "subagents") (Join-Path $root ".agents/subagents") }
    $files.Add(".agents/subagents")
}

if ($InstallHooks) {
    $detectJson = & "$PSScriptRoot\detect-existing-hooks.ps1" -RepoRoot $root -Json
    $detect = $detectJson | ConvertFrom-Json
    if ($detect.hasExistingHook -and (-not $ForceHookOverwrite)) {
        Write-AgentDevKitJson -Json:$Json -Data @{
            ok=$false
            error="Existing repository hook detected. v0.11.1 will not overwrite hooks by default. Use -InstallHookBridge or -ForceHookOverwrite."
            existingPreCommitHooks=$detect.existingPreCommitHooks
        }
        exit 1
    }
    if (-not $DryRun) {
        New-AgentDevKitDirectory (Join-Path $root ".githooks")
        Copy-Item -LiteralPath (Join-Path $kitRoot ".githooks/pre-commit") -Destination (Join-Path $root ".githooks/pre-commit") -Force
        Copy-Item -LiteralPath (Join-Path $kitRoot ".githooks/commit-msg") -Destination (Join-Path $root ".githooks/commit-msg") -Force
        git -C $root config core.hooksPath .githooks | Out-Null
    }
    $files.Add(".githooks/pre-commit"); $files.Add(".githooks/commit-msg")
}

if ($InstallHookBridge) {
    if (-not $DryRun) {
        & "$PSScriptRoot\install-hook-bridge.ps1" -RepoRoot $root -MergeExistingHook -Json | Out-Null
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    $files.Add("hook-bridge")
}

if ($InstallWorkflow) {
    if (-not $DryRun) {
        New-AgentDevKitDirectory (Join-Path $root ".github/workflows")
        Copy-Item -LiteralPath (Join-Path $kitRoot ".github/workflows/agent-dev-kit-ci.yml") -Destination (Join-Path $root ".github/workflows/agent-dev-kit-ci.yml") -Force
    }
    $files.Add(".github/workflows/agent-dev-kit-ci.yml")
}
if ($InstallThinSkill) {
    if (-not $DryRun) {
        New-AgentDevKitDirectory (Join-Path $root ".agents/skills/aicoding-agent-dev-kit")
        Copy-Item -LiteralPath (Join-Path $kitRoot "thin-skills/aicoding-agent-dev-kit/SKILL.md") -Destination (Join-Path $root ".agents/skills/aicoding-agent-dev-kit/SKILL.md") -Force
    }
    $files.Add(".agents/skills/aicoding-agent-dev-kit/SKILL.md")
}
if (-not $DryRun) {
    Copy-Item -LiteralPath (Join-Path $kitRoot "config/agent-dev-kit.json") -Destination (Join-Path $root "config/agent-dev-kit.json") -Force
    $files.Add("config/agent-dev-kit.json")
    Write-AgentDevKitState -RepoRoot $root -Files $files.ToArray()
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok = $true; action = "install"; dryRun = [bool]$DryRun; repoRoot = $root; installedFiles = $files.ToArray() }
