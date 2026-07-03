[CmdletBinding()]
param(
    [string]$RepoRoot = "",
    [switch]$SkipGoTest,
    [switch]$NoHooksPath,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'

function New-Result($Ok, $Message, $Data = @{}) {
    [ordered]@{
        schemaVersion = 1
        command = 'install-fast-path-v1'
        ok = [bool]$Ok
        message = $Message
        data = $Data
    }
}

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
} else {
    $RepoRoot = Resolve-Path $RepoRoot
}

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    throw 'Go is required. Install Go, then rerun this script.'
}

$git = Get-Command git -ErrorAction SilentlyContinue
if (-not $git) {
    throw 'Git is required. Install Git for Windows, then rerun this script.'
}

Push-Location $RepoRoot
try {
    if (-not (Test-Path -LiteralPath 'cmd/aicoding/main.go' -PathType Leaf)) {
        throw 'cmd/aicoding/main.go not found. Apply the Fast Path V1 package at the repository root first.'
    }

    if (-not $SkipGoTest) {
        & $go.Source test ./...
        if ($LASTEXITCODE -ne 0) { throw 'go test ./... failed' }
    }

    New-Item -ItemType Directory -Force -Path 'bin' | Out-Null
    $exeName = if ($IsWindows -or $env:OS -eq 'Windows_NT') { 'aicoding.exe' } else { 'aicoding' }
    $outPath = Join-Path 'bin' $exeName
    & $go.Source build -o $outPath ./cmd/aicoding
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

    if (-not $NoHooksPath) {
        & $git.Source config core.hooksPath .githooks
        if ($LASTEXITCODE -ne 0) { throw 'git config core.hooksPath failed' }
    }

    & $outPath version | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'built CLI version check failed' }

    $result = New-Result $true 'AiCoding Fast Path V1 installed' ([ordered]@{
        repoRoot = (Get-Location).Path
        binary = $outPath
        hooksPathConfigured = (-not $NoHooksPath)
    })
    if ($Json) { $result | ConvertTo-Json -Depth 20 } else { Write-Host $result.message; Write-Host ("Binary: {0}" -f $outPath) }
}
finally {
    Pop-Location
}
