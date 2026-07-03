param([switch]$Json)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$errors = New-Object System.Collections.Generic.List[string]
function Add-Err([string]$Message){ $script:errors.Add($Message) | Out-Null }
function Invoke-Capture {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [hashtable]$Environment = @{}
    )

    $psi = [System.Diagnostics.ProcessStartInfo]::new()
    $psi.FileName = $FilePath
    foreach ($argument in $Arguments) {
        [void]$psi.ArgumentList.Add($argument)
    }
    $psi.WorkingDirectory = $repo
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    foreach ($key in $Environment.Keys) {
        $psi.Environment[$key] = [string]$Environment[$key]
    }

    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $psi
    [void]$process.Start()
    $stdout = $process.StandardOutput.ReadToEnd()
    $stderr = $process.StandardError.ReadToEnd()
    $process.WaitForExit()

    [pscustomobject]@{
        exitCode = $process.ExitCode
        stdout = $stdout
        stderr = $stderr
    }
}
$submodule = Resolve-KitPath $repo $config.agents.skillsSubmodule
$plugin = Resolve-KitPath $repo $config.agents.pluginPath
$marketplace = Resolve-KitPath $repo $config.agents.marketplacePath
$sub = Get-SubmoduleStatus $submodule
if (-not $sub.exists) { Add-Err "Missing submodule: $submodule" } elseif (-not $sub.clean) { Add-Err "Submodule working tree is dirty: $submodule" }
if (-not (Test-Path -LiteralPath $plugin)) { Add-Err "Missing plugin package: $plugin" }
if (-not (Test-Path -LiteralPath $marketplace)) { Add-Err "Missing marketplace: $marketplace" }
foreach ($prop in $config.assets.PSObject.Properties) {
    $path = Resolve-KitPath $repo $prop.Value
    if (-not (Test-Path -LiteralPath $path)) { Add-Err "Missing CodingKit asset directory: $($prop.Name) -> $path" }
}
if (Test-Path -LiteralPath $plugin) {
    $obsidian = @(Get-ChildItem -LiteralPath (Join-Path $plugin 'skills') -Directory -ErrorAction SilentlyContinue | Where-Object { $_.Name -like 'obsidian-*' })
    if ($obsidian.Count -gt 0) { Add-Err 'Obsidian skills must not be packaged in AiCoding plugin.' }
    $verifyPlugin = Join-Path $submodule 'scripts\verify-plugin.ps1'
    if (Test-Path -LiteralPath $verifyPlugin) {
        $pluginCapture = Invoke-Capture 'powershell' @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $verifyPlugin, '-PluginPath', $plugin)
        if ($pluginCapture.exitCode -ne 0) {
            Add-Err "Submodule plugin verifier failed: $($pluginCapture.stderr.Trim())"
        } elseif (-not $Json -and $pluginCapture.stdout.Trim()) {
            Write-Host $pluginCapture.stdout.Trim()
        }
    }
}
$kitLifecycleResult = $null
$freshCloneResult = $null
if ($env:AICODING_SKIP_KIT_LIFECYCLE -ne '1') {
    $kitLifecycleVerifier = Join-Path $PSScriptRoot 'verify-kit-lifecycle.ps1'
    if (Test-Path -LiteralPath $kitLifecycleVerifier) {
        $pwshCommand = Get-Command pwsh -ErrorAction SilentlyContinue
        $shell = if ($pwshCommand) { $pwshCommand.Source } else { 'powershell' }
        $kitLifecycleCapture = Invoke-Capture $shell @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $kitLifecycleVerifier, '-RepoRoot', $repo, '-Json') @{ AICODING_SKIP_KIT_LIFECYCLE = '1'; AICODING_SKIP_FRESH_CLONE = '1' }
        if ($kitLifecycleCapture.stdout.Trim()) {
            try {
                $kitLifecycleResult = $kitLifecycleCapture.stdout | ConvertFrom-Json
            }
            catch {
                Add-Err "Kit lifecycle verifier returned invalid JSON: $($_.Exception.Message)"
            }
        }
        if ($kitLifecycleCapture.exitCode -ne 0) {
            Add-Err "Kit lifecycle verifier failed: $($kitLifecycleCapture.stderr.Trim())"
        }
    } else {
        Add-Err "Missing kit lifecycle verifier: $kitLifecycleVerifier"
    }
}
if ($env:AICODING_SKIP_FRESH_CLONE -ne '1') {
    $freshCloneVerifier = Join-Path $PSScriptRoot 'test-kit-fresh-clone.ps1'
    if (Test-Path -LiteralPath $freshCloneVerifier) {
        $pwshCommand = Get-Command pwsh -ErrorAction SilentlyContinue
        $shell = if ($pwshCommand) { $pwshCommand.Source } else { 'powershell' }
        $freshCloneCapture = Invoke-Capture $shell @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $freshCloneVerifier, '-Profile', 'Smoke', '-Json') @{ AICODING_SKIP_FRESH_CLONE = '1'; AICODING_SKIP_KIT_LIFECYCLE = '1' }
        if ($freshCloneCapture.stdout.Trim()) {
            try {
                $freshCloneResult = $freshCloneCapture.stdout | ConvertFrom-Json
            }
            catch {
                Add-Err "Fresh clone verifier returned invalid JSON: $($_.Exception.Message)"
            }
        }
        if ($freshCloneCapture.exitCode -ne 0) {
            Add-Err "Fresh clone verifier failed: $($freshCloneCapture.stderr.Trim())"
        }
    } else {
        Add-Err "Missing fresh clone verifier: $freshCloneVerifier"
    }
}
$result = [pscustomobject]@{ ok=($errors.Count -eq 0); errors=$errors; kitLifecycle=$kitLifecycleResult; freshClone=$freshCloneResult }
if ($Json) { $result | ConvertTo-Json -Depth 30 } elseif ($errors.Count -eq 0) { Write-Host 'AiCoding Codex kit verification passed.' } else { $errors | ForEach-Object { Write-Error $_ } }
if ($errors.Count -gt 0) { exit 1 }