[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Args
)

$ErrorActionPreference = 'Stop'
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$exeName = if ($IsWindows -or $env:OS -eq 'Windows_NT') { 'aicoding.exe' } else { 'aicoding' }
$bin = Join-Path $repoRoot (Join-Path 'bin' $exeName)

if (Test-Path -LiteralPath $bin -PathType Leaf) {
    & $bin @Args
    exit $LASTEXITCODE
}

$go = Get-Command go -ErrorAction SilentlyContinue
if ($go) {
    Push-Location $repoRoot
    try {
        & $go.Source run ./cmd/aicoding @Args
        exit $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
}

throw "AiCoding fast CLI is not built and Go is not available. Build with: go build -o bin/$exeName ./cmd/aicoding"
