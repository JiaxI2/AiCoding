[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [string]$RepoRoot = '.',

    [Parameter(Mandatory = $false)]
    [switch]$Json,

    [Parameter(Mandatory = $false)]
    [switch]$SkipGo,

    [Parameter(Mandatory = $false)]
    [switch]$StrictTags
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    return (Resolve-Path -LiteralPath $Path -ErrorAction Stop).ProviderPath
}

function Add-Check {
    param(
        [Parameter(Mandatory = $true)][AllowEmptyCollection()][System.Collections.Generic.List[object]]$Checks,
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][bool]$Ok,
        [Parameter(Mandatory = $false)][string]$Message = ''
    )
    $Checks.Add([pscustomobject]@{ name = $Name; ok = $Ok; message = $Message }) | Out-Null
}

function Test-CommandExists {
    param([Parameter(Mandatory = $true)][string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

$RepoRootPath = Resolve-FullPath -Path $RepoRoot
$checks = New-Object System.Collections.Generic.List[object]
Push-Location $RepoRootPath
try {
    $required = @(
        'Taskfile.yml',
        'docs/TAGGING_POLICY.md',
        'docs/RELEASE_POLICY.md',
        'docs/RELEASE_GOVERNANCE_OVERLAY.md',
        'tools/specialty/aicoding-tag-governance.ps1',
        'tools/specialty/verify-release-governance-overlay.ps1',
        'config/tagging-policy.json',
        'config/kits/release-governance-overlay-kit.json',
        '.aicoding/templates/perf-cache-plan.json'
    )

    foreach ($path in $required) {
        Add-Check -Checks $checks -Name "exists:$path" -Ok (Test-Path -LiteralPath $path) -Message $path
    }

    if (Test-Path -LiteralPath 'tools/specialty/aicoding-tag-governance.ps1') {
        $args = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', 'tools/specialty/aicoding-tag-governance.ps1', '-Action', 'Verify', '-Json')
        if ($StrictTags) { $args += '-Strict' }
        & pwsh @args | Out-Null
        Add-Check -Checks $checks -Name 'tag-governance:verify' -Ok ($LASTEXITCODE -eq 0) -Message 'aicoding-tag-governance.ps1 Verify'
    }

    if (-not $SkipGo) {
        if ((Test-Path -LiteralPath 'go.mod') -and (Test-CommandExists -Name 'go')) {
            & go test ./... | Out-Host
            Add-Check -Checks $checks -Name 'go:test' -Ok ($LASTEXITCODE -eq 0) -Message 'go test ./...'
            if (-not (Test-Path -LiteralPath 'bin')) { New-Item -ItemType Directory -Path 'bin' -Force | Out-Null }
            & go build -o bin/aicoding.exe ./cmd/aicoding | Out-Host
            Add-Check -Checks $checks -Name 'go:build' -Ok ($LASTEXITCODE -eq 0) -Message 'go build -o bin/aicoding.exe ./cmd/aicoding'
        } else {
            Add-Check -Checks $checks -Name 'go:skip' -Ok $true -Message 'go.mod or go command not found; skipped.'
        }
    }

    if (Test-Path -LiteralPath 'bin/aicoding.exe') {
        & bin/aicoding.exe kit verify --all --profile Smoke --json | Out-Host
        Add-Check -Checks $checks -Name 'aicoding:kit-smoke' -Ok ($LASTEXITCODE -eq 0) -Message 'bin/aicoding.exe kit verify --all --profile Smoke --json'
        & bin/aicoding.exe governance lint --json | Out-Host
        Add-Check -Checks $checks -Name 'aicoding:governance' -Ok ($LASTEXITCODE -eq 0) -Message 'bin/aicoding.exe governance lint --json'
        & bin/aicoding.exe doctor perf --json | Out-Host
        Add-Check -Checks $checks -Name 'aicoding:perf' -Ok ($LASTEXITCODE -eq 0) -Message 'bin/aicoding.exe doctor perf --json'
    } else {
        Add-Check -Checks $checks -Name 'aicoding:skip' -Ok $true -Message 'bin/aicoding.exe not found; skipped fast path runtime checks.'
    }

    if (Test-CommandExists -Name 'task') {
        & task --list | Out-Null
        Add-Check -Checks $checks -Name 'task:available' -Ok ($LASTEXITCODE -eq 0) -Message 'Task CLI available.'
    } else {
        Add-Check -Checks $checks -Name 'task:skip' -Ok $true -Message 'Task CLI not installed; skipped.'
    }

    $failed = @($checks | Where-Object { -not $_.ok })
    $result = [pscustomobject]@{
        ok = ($failed.Count -eq 0)
        repoRoot = $RepoRootPath
        failed = $failed.Count
        checks = $checks
        safety = [pscustomobject]@{
            remoteTagsDeleted = $false
            forcePush = $false
            dssOrXdsActions = $false
            flashOrMemoryWrite = $false
        }
    }

    if ($Json) {
        $result | ConvertTo-Json -Depth 8
    } else {
        if ($result.ok) {
            Write-Host 'Release governance overlay verification passed.'
        } else {
            Write-Host "Release governance overlay verification failed: $($failed.Count) failed checks."
        }
    }

    if (-not $result.ok) { exit 1 }
} finally {
    Pop-Location
}
