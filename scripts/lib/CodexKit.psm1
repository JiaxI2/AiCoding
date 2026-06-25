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

function Test-CodexPluginCli {
    try {
        $output = & codex plugin --help 2>&1
        return [pscustomobject]@{ available = ($LASTEXITCODE -eq 0); detail = ($output -join "`n") }
    } catch {
        return [pscustomobject]@{ available = $false; detail = $_.Exception.Message }
    }
}

function Get-SubmoduleStatus {
    param([string]$SubmodulePath)
    if (-not (Test-Path -LiteralPath $SubmodulePath)) { return [pscustomobject]@{ exists=$false } }
    $commit = (& git -C $SubmodulePath rev-parse HEAD).Trim()
    $status = @(& git -C $SubmodulePath status --porcelain)
    return [pscustomobject]@{ exists=$true; commit=$commit; clean=($status.Count -eq 0); status=$status }
}