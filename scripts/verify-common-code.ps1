[CmdletBinding()]
param(
    [string]$RepoRoot = "",
    [switch]$Json
)

$ErrorActionPreference = "Stop"
$repo = if ($RepoRoot) { (Resolve-Path -LiteralPath $RepoRoot).Path } else { (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path }
$checks = New-Object System.Collections.Generic.List[object]
$errors = New-Object System.Collections.Generic.List[string]

function Add-Check {
    param([string]$Name, [bool]$Ok, [string]$Message, [object]$Data = $null)
    $checks.Add([pscustomobject]@{ name=$Name; ok=$Ok; message=$Message; data=$Data }) | Out-Null
    if (-not $Ok) { $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null }
}
function Resolve-RepoPath([string]$Path) { Join-Path $repo ($Path -replace '/', '\') }

$configPath = Resolve-RepoPath 'config/common-registry.json'
if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
    Add-Check 'common.registry.exists' $false 'missing config/common-registry.json'
} else {
    try {
        $registry = Get-Content -LiteralPath $configPath -Raw -Encoding UTF8 | ConvertFrom-Json
        Add-Check 'common.registry.parse' $true 'parsed'
        foreach ($module in @($registry.modules)) {
            $modulePath = Resolve-RepoPath ([string]$module.path)
            Add-Check "common.module.path:$($module.id)" (Test-Path -LiteralPath $modulePath -PathType Container) ([string]$module.path)
            $readmes = @($module.readmePaths | Where-Object { Test-Path -LiteralPath (Resolve-RepoPath ([string]$_)) -PathType Leaf })
            if ($readmes.Count -eq 0 -and (Test-Path -LiteralPath $modulePath)) {
                $readmes = @(Get-ChildItem -LiteralPath $modulePath -Recurse -File -Filter README.md -ErrorAction SilentlyContinue | ForEach-Object { $_.FullName })
            }
            Add-Check "common.module.readme:$($module.id)" ($readmes.Count -gt 0) 'README or module docs exist' @{ count=$readmes.Count }
            $tests = @($module.tests | Where-Object { Test-Path -LiteralPath (Resolve-RepoPath ([string]$_)) -PathType Container })
            if ($tests.Count -eq 0 -and (Test-Path -LiteralPath $modulePath)) {
                $tests = @(Get-ChildItem -LiteralPath $modulePath -Recurse -Directory -Filter tests -ErrorAction SilentlyContinue | ForEach-Object { $_.FullName })
            }
            Add-Check "common.module.tests:$($module.id)" ($tests.Count -gt 0) 'tests directory exists' @{ count=$tests.Count }
        }

        $kitFiles = @(Get-ChildItem -LiteralPath (Resolve-RepoPath 'config/kits') -Filter *.json -File)
        foreach ($kitFile in $kitFiles) {
            $kit = Get-Content -LiteralPath $kitFile.FullName -Raw -Encoding UTF8 | ConvertFrom-Json
            foreach ($dep in @($kit.commonDependencies)) {
                $exists = @($registry.modules | Where-Object { $_.id -eq $dep.id }).Count -gt 0
                Add-Check "common.dependency.registered:$($kit.id).$($dep.id)" $exists 'common dependency is registered'
            }
        }
    } catch {
        Add-Check 'common.registry.parse' $false $_.Exception.Message
    }
}

$result = [pscustomobject]@{ schemaVersion=1; ok=($errors.Count -eq 0); repoRoot=$repo; checks=$checks; errors=@($errors) }
if ($Json) { $result | ConvertTo-Json -Depth 20 } elseif ($result.ok) { Write-Host 'AiCoding common code verification passed.' } else { $errors | ForEach-Object { Write-Error $_ } }
if (-not $result.ok) { exit 1 }