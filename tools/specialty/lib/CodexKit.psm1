function Get-AiCodingRoot {
    param([string]$ScriptRoot)
    $current = Get-Item -LiteralPath $ScriptRoot -ErrorAction Stop
    while ($null -ne $current) {
        $config = Join-Path $current.FullName 'config\codex-kit.json'
        if (Test-Path -LiteralPath $config -PathType Leaf) {
            return $current.FullName
        }
        $current = $current.Parent
    }
    throw "Unable to locate AiCoding root from: $ScriptRoot"
}

function Read-CodexKitConfig {
    param([string]$RepoRoot)
    $path = Join-Path $RepoRoot 'config\codex-kit.json'
    if (-not (Test-Path -LiteralPath $path)) { throw "Missing config: $path" }
    return Get-Content -Raw -LiteralPath $path | ConvertFrom-Json
}

function Resolve-KitPath {
    param([string]$RepoRoot, [string]$RelativePath)
    $rel = $RelativePath -replace '/', '\\'
    if ($rel.StartsWith('.\\')) { $rel = $rel.Substring(2) }
    return Join-Path $RepoRoot $rel
}

function Resolve-CodexKitRuntimePath {
    param([string]$RepoRoot, [string]$PathValue)
    if ([string]::IsNullOrWhiteSpace($PathValue)) { return $null }
    $expanded = $PathValue.Replace('%USERPROFILE%', $env:USERPROFILE)
    $expanded = [Environment]::ExpandEnvironmentVariables($expanded)
    if ([System.IO.Path]::IsPathRooted($expanded)) { return $expanded }
    if ([string]::IsNullOrWhiteSpace($RepoRoot)) { return $expanded }
    return [System.IO.Path]::GetFullPath((Join-Path $RepoRoot $expanded))
}

function Resolve-CodexKitConfiguredPath {
    param(
        $ConfigSection,
        [string]$RepoRoot,
        [string[]]$PropertyOrder = @('sourceRepository', 'defaultSourceRepository')
    )
    if (-not $ConfigSection) { return $null }
    $propertyNames = @($ConfigSection.PSObject.Properties.Name)
    if ($propertyNames -contains 'sourceRepositoryEnv') {
        $envName = [string]$ConfigSection.sourceRepositoryEnv
        if (-not [string]::IsNullOrWhiteSpace($envName)) {
            $envValue = [Environment]::GetEnvironmentVariable($envName)
            if (-not [string]::IsNullOrWhiteSpace($envValue)) {
                return Resolve-CodexKitRuntimePath -RepoRoot $RepoRoot -PathValue $envValue
            }
        }
    }
    foreach ($propertyName in $PropertyOrder) {
        if ($propertyNames -contains $propertyName) {
            $value = [string]$ConfigSection.$propertyName
            if (-not [string]::IsNullOrWhiteSpace($value)) {
                return Resolve-CodexKitRuntimePath -RepoRoot $RepoRoot -PathValue $value
            }
        }
    }
    return $null
}
function Find-CodexCliPath {
    $command = Get-Command codex -ErrorAction SilentlyContinue
    if ($command -and $command.Source) { return $command.Source }
    $binRoot = Join-Path $env:LOCALAPPDATA 'OpenAI\Codex\bin'
    if (Test-Path -LiteralPath $binRoot) {
        $candidate = Get-ChildItem -LiteralPath $binRoot -Recurse -Filter 'codex.exe' -File -ErrorAction SilentlyContinue |
            Sort-Object LastWriteTime -Descending |
            Select-Object -First 1
        if ($candidate) { return $candidate.FullName }
    }
    return $null
}

function Test-CodexPluginCli {
    $path = Find-CodexCliPath
    if (-not $path) { return [pscustomobject]@{ available = $false; path = $null; detail = 'codex CLI not found' } }
    try {
        $output = & $path plugin --help 2>&1
        return [pscustomobject]@{ available = ($LASTEXITCODE -eq 0); path = $path; detail = ($output -join "`n") }
    } catch {
        $binRoot = Join-Path $env:LOCALAPPDATA 'OpenAI\Codex\bin'
        $fallback = $null
        if (Test-Path -LiteralPath $binRoot) {
            $fallback = Get-ChildItem -LiteralPath $binRoot -Recurse -Filter 'codex.exe' -File -ErrorAction SilentlyContinue |
                Where-Object { $_.FullName -ne $path } |
                Sort-Object LastWriteTime -Descending |
                Select-Object -First 1
        }
        if ($fallback) {
            try {
                $output = & $fallback.FullName plugin --help 2>&1
                return [pscustomobject]@{ available = ($LASTEXITCODE -eq 0); path = $fallback.FullName; detail = ($output -join "`n") }
            } catch {
                return [pscustomobject]@{ available = $false; path = $fallback.FullName; detail = $_.Exception.Message }
            }
        }
        return [pscustomobject]@{ available = $false; path = $path; detail = $_.Exception.Message }
    }
}

function Get-SubmoduleStatus {
    param([string]$SubmodulePath)
    if (-not (Test-Path -LiteralPath $SubmodulePath)) { return [pscustomobject]@{ exists=$false } }
    $commit = (& git -C $SubmodulePath rev-parse HEAD).Trim()
    $status = @(& git -C $SubmodulePath status --porcelain)
    return [pscustomobject]@{ exists=$true; commit=$commit; clean=($status.Count -eq 0); status=$status }
}
