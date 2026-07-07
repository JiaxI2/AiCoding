[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$RepoRoot = "",
    [switch]$UnsetHooksPath,
    [switch]$RemoveBinary,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
} else {
    $RepoRoot = Resolve-Path $RepoRoot
}

$actions = New-Object System.Collections.Generic.List[string]
Push-Location $RepoRoot
try {
    if ($UnsetHooksPath) {
        git config --unset core.hooksPath 2>$null
        $actions.Add('unset core.hooksPath')
    }
    if ($RemoveBinary) {
        foreach ($p in @('bin/aicoding.exe','bin/aicoding')) {
            if (Test-Path -LiteralPath $p) {
                if ($PSCmdlet.ShouldProcess($p, "Remove Fast Path binary")) {
                    Remove-Item -LiteralPath $p -Force
                    $actions.Add("removed $p")
                }
            }
        }
    }
    $result = [ordered]@{ schemaVersion = 1; command = 'rollback-fast-path-v1'; ok = $true; actions = $actions }
    if ($Json) { $result | ConvertTo-Json -Depth 20 } else { $actions | ForEach-Object { Write-Host $_ } }
}
finally {
    Pop-Location
}
