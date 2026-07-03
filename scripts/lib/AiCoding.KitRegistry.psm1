function Resolve-AiCodingKitRepoRoot {
    param([string]$RepoRoot = "")

    if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
        $RepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..\..")).Path
    }
    return (Resolve-Path -LiteralPath $RepoRoot -ErrorAction Stop).Path
}

function Resolve-AiCodingKitPath {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [Parameter(Mandatory=$true)][string]$RelativePath
    )

    $rel = $RelativePath -replace '/', '\'
    if ($rel.StartsWith(".\")) { $rel = $rel.Substring(2) }
    return Join-Path $RepoRoot $rel
}

function Read-AiCodingKitJson {
    param([Parameter(Mandatory=$true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) { throw "Missing JSON file: $Path" }
    return Get-Content -LiteralPath $Path -Raw -Encoding utf8 | ConvertFrom-Json
}

function Get-AiCodingKitRegistry {
    param(
        [string]$RepoRoot = "",
        [switch]$Enabled,
        [string]$Kit = ""
    )

    $root = Resolve-AiCodingKitRepoRoot -RepoRoot $RepoRoot
    $registryPath = Join-Path $root "config\kit-registry.json"
    $registry = Read-AiCodingKitJson -Path $registryPath
    $entries = @($registry.kits)

    if ($Enabled) { $entries = @($entries | Where-Object { $_.enabled -eq $true }) }
    if ($Kit) { $entries = @($entries | Where-Object { $_.id -eq $Kit }) }

    $result = @()
    foreach ($entry in ($entries | Sort-Object order, id)) {
        $manifestPath = Resolve-AiCodingKitPath -RepoRoot $root -RelativePath $entry.manifest
        $manifest = Read-AiCodingKitJson -Path $manifestPath
        if ($manifest.id -ne $entry.id) {
            throw "Registry id '$($entry.id)' does not match manifest id '$($manifest.id)' in $($entry.manifest)"
        }

        $result += [pscustomobject]@{
            id = $entry.id
            enabled = [bool]$entry.enabled
            order = [int]$entry.order
            manifestPath = $manifestPath
            manifestRelativePath = $entry.manifest
            manifest = $manifest
        }
    }

    return $result
}

function Test-AiCodingKitRegistry {
    param([string]$RepoRoot = "")

    $root = Resolve-AiCodingKitRepoRoot -RepoRoot $RepoRoot
    $kits = @(Get-AiCodingKitRegistry -RepoRoot $root)
    $ids = @{}
    $errors = @()

    foreach ($kit in $kits) {
        if ($ids.ContainsKey($kit.id)) {
            $errors += "Duplicate kit id: $($kit.id)"
        } else {
            $ids[$kit.id] = $true
        }

        if (-not $kit.manifest.schemaVersion) { $errors += "Missing schemaVersion: $($kit.id)" }
        if (-not $kit.manifest.mode) { $errors += "Missing mode: $($kit.id)" }
        if (-not $kit.manifest.commands) { $errors += "Missing commands: $($kit.id)" }
    }

    return [pscustomobject]@{
        schemaVersion = 2
        ok = ($errors.Count -eq 0)
        repoRoot = $root
        total = $kits.Count
        errors = $errors
    }
}

Export-ModuleMember -Function Resolve-AiCodingKitRepoRoot, Resolve-AiCodingKitPath, Read-AiCodingKitJson, Get-AiCodingKitRegistry, Test-AiCodingKitRegistry
