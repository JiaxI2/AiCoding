param([switch]$Json)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'lib\CodexKit.psm1') -Force
$repo = Get-AiCodingRoot $PSScriptRoot
$config = Read-CodexKitConfig $repo
$errors = New-Object System.Collections.Generic.List[string]
function Add-Err([string]$Message){ $script:errors.Add($Message) | Out-Null }
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
        & powershell -NoProfile -ExecutionPolicy Bypass -File $verifyPlugin -PluginPath $plugin
        if ($LASTEXITCODE -ne 0) { Add-Err 'Submodule plugin verifier failed.' }
    }
}
$result = [pscustomobject]@{ ok=($errors.Count -eq 0); errors=$errors }
if ($Json) { $result | ConvertTo-Json -Depth 5 } elseif ($errors.Count -eq 0) { Write-Host 'AiCoding Codex kit verification passed.' } else { $errors | ForEach-Object { Write-Error $_ } }
if ($errors.Count -gt 0) { exit 1 }