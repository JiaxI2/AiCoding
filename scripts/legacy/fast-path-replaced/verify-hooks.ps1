# Deprecated: this fast-path check is superseded by bin\aicoding.exe verify hooks --json.
# Kept as a temporary fallback for v0.1.x.
# Do not call from Taskfile smoke or Git hooks.

[CmdletBinding()]
param(
    [string]$RepoRoot = "",
    [switch]$Json
)

$ErrorActionPreference = "Stop"
if ($RepoRoot) {
    $repo = (Resolve-Path -LiteralPath $RepoRoot).Path
} else {
    $gitRoot = (& git -C $PSScriptRoot rev-parse --show-toplevel 2>$null)
    if ($LASTEXITCODE -eq 0 -and $gitRoot) {
        $repo = (Resolve-Path -LiteralPath $gitRoot.Trim()).Path
    } else {
        $repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..\..")).Path
    }
}
$checks = New-Object System.Collections.Generic.List[object]
$errors = New-Object System.Collections.Generic.List[string]

function Add-Check {
    param([string]$Name, [bool]$Ok, [string]$Message, [object]$Data = $null)
    $checks.Add([pscustomobject]@{ name=$Name; ok=$Ok; message=$Message; data=$Data }) | Out-Null
    if (-not $Ok) { $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null }
}
function Resolve-RepoPath([string]$Path) { Join-Path $repo ($Path -replace '/', '\') }
function Test-PowerShellSyntax([string]$Path) {
    $tokens = $null
    $parseErrors = $null
    [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$parseErrors) | Out-Null
    return @($parseErrors).Count -eq 0
}

$configPath = Resolve-RepoPath 'config/hooks-registry.json'
$kitRegistryPath = Resolve-RepoPath 'config/kit-registry.json'
$kitIds = @()
if (Test-Path -LiteralPath $kitRegistryPath -PathType Leaf) {
    $kitRegistry = Get-Content -LiteralPath $kitRegistryPath -Raw -Encoding UTF8 | ConvertFrom-Json
    $kitIds = @($kitRegistry.kits | ForEach-Object { $_.id })
}

if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
    Add-Check 'hooks.registry.exists' $false 'missing config/hooks-registry.json'
} else {
    try {
        $registry = Get-Content -LiteralPath $configPath -Raw -Encoding UTF8 | ConvertFrom-Json
        Add-Check 'hooks.registry.parse' $true 'parsed'
        $seen = @{}
        foreach ($hook in @($registry.hooks)) {
            if ($seen.ContainsKey($hook.id)) { Add-Check "hook.id.unique:$($hook.id)" $false 'duplicate hook id' } else { $seen[$hook.id] = $true; Add-Check "hook.id.unique:$($hook.id)" $true 'unique' }
            Add-Check "hook.owner.exists:$($hook.id)" ($kitIds -contains $hook.owner) ([string]$hook.owner)
            Add-Check "hook.trigger:$($hook.id)" (-not [string]::IsNullOrWhiteSpace([string]$hook.trigger)) ([string]$hook.trigger)
            $hookPath = Resolve-RepoPath ([string]$hook.path)
            Add-Check "hook.path:$($hook.id)" (Test-Path -LiteralPath $hookPath -PathType Leaf) ([string]$hook.path)
            if ((Test-Path -LiteralPath $hookPath -PathType Leaf) -and $hookPath.EndsWith('.ps1', [System.StringComparison]::OrdinalIgnoreCase)) {
                Add-Check "hook.syntax:$($hook.id)" (Test-PowerShellSyntax $hookPath) 'PowerShell parser'
            }
        }
    } catch {
        Add-Check 'hooks.registry.parse' $false $_.Exception.Message
    }
}

$result = [pscustomobject]@{ schemaVersion=1; ok=($errors.Count -eq 0); repoRoot=$repo; checks=$checks; errors=@($errors) }
if ($Json) { $result | ConvertTo-Json -Depth 20 } elseif ($result.ok) { Write-Host 'AiCoding hook verification passed.' } else { $errors | ForEach-Object { Write-Error $_ } }
if (-not $result.ok) { exit 1 }