# RETIRE-AFTER: one release after Phase 1 reference migration
[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

# Compatibility wrapper only. The former `aicoding full` compat route was removed in
# v1.0.0; the canonical gate is `bin\aicoding.exe test --profile Full --json`.
# Retirement of this wrapper is tracked in
# docs/decisions/verify-codex-kit-retirement/RETIREMENT_PLAN.md.
# The notice must go to stderr: a child pwsh renders Write-Warning onto stdout,
# which would corrupt the strict-JSON stdout contract for -Json callers.
[Console]::Error.WriteLine('WARNING: verify-codex-kit.ps1 is a compatibility wrapper; prefer bin\aicoding.exe test --profile Full --json (see docs/decisions/verify-codex-kit-retirement/RETIREMENT_PLAN.md).')

$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..')).Path
$bin = Join-Path $repo 'bin\aicoding.exe'
$cliArgs = @('test', '--profile', 'Full', '--json', '--repo-root', $repo)

# The CLI emits UTF-8, but non-interactive hosts default [Console]::OutputEncoding to
# the OEM codepage (e.g. CP936): native capture then corrupts multi-byte JSON, and the
# wrapper's own redirected stdout would re-encode it. Set UTF-8 for both directions;
# the change is process-scoped (this script runs via `pwsh -File`), so no restore.
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)

Push-Location $repo
try {
    if (Test-Path -LiteralPath $bin -PathType Leaf) {
        $raw = & $bin @cliArgs
    } elseif ($null -ne (Get-Command go -ErrorAction SilentlyContinue)) {
        $raw = & go @(@('run', './cmd/aicoding') + $cliArgs)
    } else {
        Write-Error -ErrorAction Continue 'Neither bin\aicoding.exe nor the go toolchain is available.'
        exit 1
    }
    $cliExit = $LASTEXITCODE
} finally {
    Pop-Location
}

$rawText = ($raw | Out-String).Trim()
try {
    $result = $rawText | ConvertFrom-Json
} catch {
    Write-Error -ErrorAction Continue "aicoding test --profile Full did not return parseable JSON (exit $cliExit): $rawText"
    if ($cliExit -ne 0) { exit $cliExit } else { exit 1 }
}

function Get-ResultField {
    param(
        [Parameter(Mandatory = $true)]$InputObject,
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $false)]$Default = ''
    )
    $prop = $InputObject.PSObject.Properties[$Name]
    if ($null -ne $prop) { return $prop.Value }
    return $Default
}

$ok = [bool](Get-ResultField -InputObject $result -Name 'ok' -Default $false)
$errorKind = [string](Get-ResultField -InputObject $result -Name 'errorKind')
$message = [string](Get-ResultField -InputObject $result -Name 'message' -Default 'aicoding test --profile Full')
$errors = @(Get-ResultField -InputObject $result -Name 'errors' -Default @())
$elapsed = Get-ResultField -InputObject $result -Name 'elapsedMs' -Default 0

if ($Json) {
    $rawText
} else {
    $status = if ($ok) { 'OK' } else { 'FAIL' }
    Write-Host ("[{0}] {1} ({2} ms)" -f $status, $message, $elapsed)
    foreach ($entry in $errors) { Write-Host ("  - {0}" -f $entry) }
    if (-not $ok -and $errorKind) { Write-Host ("  errorKind: {0}" -f $errorKind) }
}

# Verdict comes from the JSON contract: ok=true -> 0; errorKind=usage -> 2; other failures -> 1.
if ($ok) {
    if ($cliExit -ne 0) {
        Write-Error -ErrorAction Continue "JSON reports ok=true but the CLI exited with $cliExit; failing closed."
        exit 1
    }
    exit 0
}
if ($errorKind -eq 'usage') { exit 2 }
exit 1
