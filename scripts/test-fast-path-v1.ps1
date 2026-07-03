[CmdletBinding()]
param(
    [string]$RepoRoot = "",
    [switch]$SkipRepoSmoke,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
} else {
    $RepoRoot = Resolve-Path $RepoRoot
}

$steps = New-Object System.Collections.Generic.List[object]
function Invoke-Step([string]$Name, [scriptblock]$Body) {
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        & $Body
        $sw.Stop()
        $steps.Add([ordered]@{ name = $Name; ok = $true; elapsedMs = $sw.ElapsedMilliseconds })
    } catch {
        $sw.Stop()
        $steps.Add([ordered]@{ name = $Name; ok = $false; elapsedMs = $sw.ElapsedMilliseconds; error = $_.Exception.Message })
        throw
    }
}

Push-Location $RepoRoot
try {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) { throw 'Go is required for test-fast-path-v1.ps1' }
    $exeName = if ($IsWindows -or $env:OS -eq 'Windows_NT') { 'aicoding.exe' } else { 'aicoding' }
    $bin = Join-Path 'bin' $exeName

    Invoke-Step 'go test ./...' { & $go.Source test ./...; if ($LASTEXITCODE -ne 0) { throw 'go test failed' } }
    Invoke-Step 'go build' {
        New-Item -ItemType Directory -Force -Path 'bin' | Out-Null
        & $go.Source build -o $bin ./cmd/aicoding
        if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
    }
    Invoke-Step 'aicoding version' { & $bin version | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'version failed' } }
    if (-not $SkipRepoSmoke) {
        Invoke-Step 'kit list' { & $bin kit list --json | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'kit list failed' } }
        Invoke-Step 'kit smoke verify' { & $bin kit verify --all --profile Smoke --json | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'kit smoke verify failed' } }
        Invoke-Step 'governance lint' { & $bin governance lint --json | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'governance lint failed' } }
        Invoke-Step 'doctor perf' { & $bin doctor perf --json | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'doctor perf failed' } }
    }
    $result = [ordered]@{ schemaVersion = 1; command = 'test-fast-path-v1'; ok = $true; steps = $steps }
    if ($Json) { $result | ConvertTo-Json -Depth 20 } else { $steps | Format-Table -AutoSize }
}
finally {
    Pop-Location
}
