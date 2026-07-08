[CmdletBinding()]
param(
    [string]$RepoRoot = "",
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
} else {
    $RepoRoot = Resolve-Path $RepoRoot
}

Push-Location $RepoRoot
try {
    $exeName = if ($IsWindows -or $env:OS -eq 'Windows_NT') { 'aicoding.exe' } else { 'aicoding' }
    $bin = Join-Path 'bin' $exeName
    if (-not (Test-Path -LiteralPath $bin -PathType Leaf)) {
        if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Fast binary not found and Go is not available.' }
        New-Item -ItemType Directory -Force -Path 'bin' | Out-Null
        go build -o $bin ./cmd/aicoding
    }
    $cases = @(
        @{ name = 'fast: doctor perf'; command = { & $bin doctor perf --json | Out-Null } },
        @{ name = 'fast: kit smoke'; command = { & $bin kit verify --all --profile Smoke --json | Out-Null } },
        @{ name = 'fast: governance lint'; command = { & $bin governance lint --json | Out-Null } }
    )
    if (Get-Command pwsh -ErrorAction SilentlyContinue) {
        $cases += @(
            @{ name = 'go: governance lint'; command = { & $bin governance lint --json | Out-Null } },
            @{ name = 'go: full smoke'; command = { &  full --json | Out-Null } }
        )
    }
    $results = foreach ($case in $cases) {
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        $ok = $true
        $err = $null
        try { & $case.command } catch { $ok = $false; $err = $_.Exception.Message }
        $sw.Stop()
        [ordered]@{ name = $case.name; ok = $ok; elapsedMs = $sw.ElapsedMilliseconds; error = $err }
    }
    if ($Json) { $results | ConvertTo-Json -Depth 20 } else { $results | Format-Table -AutoSize }
}
finally {
    Pop-Location
}
