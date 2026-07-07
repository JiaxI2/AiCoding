# Deprecated: this fast-path check is superseded by bin\aicoding.exe verify repo-text --json.
# Kept as a temporary fallback for v0.1.x.
# Do not call from Taskfile smoke or Git hooks.

[CmdletBinding()]
param(
    [switch]$Json
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repo = (git rev-parse --show-toplevel 2>$null)
if (-not $repo) { $repo = (Get-Location).Path }
Set-Location $repo

$failures = New-Object System.Collections.Generic.List[string]
$warnings = New-Object System.Collections.Generic.List[string]

function Add-Failure([string]$Message) { $script:failures.Add($Message) | Out-Null }
function Add-Warning([string]$Message) { $script:warnings.Add($Message) | Out-Null }
function Test-FileExists([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        Add-Failure "missing file: $Path"
        return $false
    }
    return $true
}
function Get-LineCount([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return 0 }
    return @((Get-Content -LiteralPath $Path)).Count
}
function Test-MinLines([string]$Path, [int]$MinLines) {
    if (-not (Test-FileExists $Path)) { return }
    $count = Get-LineCount $Path
    if ($count -lt $MinLines) { Add-Failure "$Path has $count lines; expected at least $MinLines" }
}
function Test-NoCRLF([string]$Path) {
    if (-not (Test-FileExists $Path)) { return }
    $bytes = [System.IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $Path))
    for ($i = 0; $i -lt ($bytes.Length - 1); $i++) {
        if ($bytes[$i] -eq 13 -and $bytes[$i + 1] -eq 10) {
            Add-Failure "$Path must use LF only"
            return
        }
    }
}
function Test-Json([string]$Path) {
    if (-not (Test-FileExists $Path)) { return }
    try { Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json | Out-Null }
    catch { Add-Failure "$Path is not valid JSON: $($_.Exception.Message)" }
}
function Test-PowerShellAst([string]$Path) {
    if (-not (Test-FileExists $Path)) { return }
    $tokens = $null
    $errors = $null
    [System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path -LiteralPath $Path), [ref]$tokens, [ref]$errors) | Out-Null
    if ($errors.Count -gt 0) { Add-Failure "$Path has PowerShell parse errors: $($errors[0].Message)" }
}

Test-MinLines '.githooks/pre-commit' 20
Test-MinLines '.githooks/commit-msg' 20
Test-MinLines '.gitmodules' 6
Test-MinLines '.gitattributes' 8
Test-MinLines '.github/workflows/fast-path.yml' 20
Test-MinLines '.github/workflows/docs-sync.yml' 15
Test-MinLines '.github/repository-governance.toml' 20

if (Test-FileExists '.githooks/pre-commit') {
    $first = (Get-Content -LiteralPath '.githooks/pre-commit' -TotalCount 1)
    if ($first -ne '#!/bin/sh') { Add-Failure '.githooks/pre-commit first line must be exactly #!/bin/sh' }
    Test-NoCRLF '.githooks/pre-commit'
}
if (Test-FileExists '.githooks/commit-msg') {
    $first = (Get-Content -LiteralPath '.githooks/commit-msg' -TotalCount 1)
    if ($first -ne '#!/bin/sh') { Add-Failure '.githooks/commit-msg first line must be exactly #!/bin/sh' }
    Test-NoCRLF '.githooks/commit-msg'
}

Test-Json 'config/codex-kit.json'
Test-Json 'config/skill-sources.json'

Get-ChildItem -LiteralPath 'scripts' -Filter '*.ps1' -File -ErrorAction SilentlyContinue | ForEach-Object {
    Test-PowerShellAst $_.FullName
}

if (Get-Command go -ErrorAction SilentlyContinue) {
    $gofmt = (& gofmt -l . 2>$null) | Where-Object { $_ -match '\.go$' }
    if ($gofmt) { Add-Failure "gofmt required: $($gofmt -join ', ')" }
} else {
    Add-Warning 'go not found; skipped gofmt check'
}

if (Test-FileExists '.gitmodules') {
    $paths = @(git config -f .gitmodules --get-regexp '^submodule\..*\.path$' 2>$null)
    if ($LASTEXITCODE -ne 0 -or $paths.Count -eq 0) { Add-Failure '.gitmodules is not parseable by git config or has no submodule paths' }
}

if (Test-Path -LiteralPath 'config/codex-kit.json') {
    $raw = Get-Content -LiteralPath 'config/codex-kit.json' -Raw
    if ($raw -match '([A-Za-z]:\\)') { Add-Failure 'config/codex-kit.json contains machine-local absolute Windows path; use env/relative fallback instead' }
}

$result = [ordered]@{
    ok = ($failures.Count -eq 0)
    failures = @($failures)
    warnings = @($warnings)
}

if ($Json) {
    $result | ConvertTo-Json -Depth 5
} else {
    if ($result.ok) {
        Write-Host 'Repo text format checks passed.'
    } else {
        Write-Host 'Repo text format checks failed:'
        $failures | ForEach-Object { Write-Host "- $_" }
    }
    if ($warnings.Count -gt 0) {
        Write-Host 'Warnings:'
        $warnings | ForEach-Object { Write-Host "- $_" }
    }
}

if (-not $result.ok) { exit 1 }
