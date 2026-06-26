function Get-AiCodingRoot {
    param([string]$ScriptRoot)
    return (Resolve-Path -LiteralPath (Join-Path $ScriptRoot '..')).Path
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
