function Get-RepoRoot {
    try {
        $root = git rev-parse --show-toplevel 2>$null
        if ($LASTEXITCODE -eq 0 -and $root) { return $root.Trim() }
    } catch {}
    return (Get-Location).Path
}

function Invoke-KitScript {
    param(
        [string]$ScriptName,
        [string[]]$ArgsList = @()
    )
    $repo = Get-RepoRoot
    $script = Join-Path $repo ("scripts/" + $ScriptName)
    if (Test-Path -LiteralPath $script) {
        & pwsh -NoProfile -ExecutionPolicy Bypass -File $script @ArgsList
        return $LASTEXITCODE
    }
    return 0
}
