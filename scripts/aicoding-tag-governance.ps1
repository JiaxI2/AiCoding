[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [ValidateSet('Audit', 'Plan', 'Verify')]
    [string]$Action = 'Audit',

    [Parameter(Mandatory = $false)]
    [string]$RepoRoot = '.',

    [Parameter(Mandatory = $false)]
    [string]$OutputDir = '.aicoding/reports/release-governance',

    [Parameter(Mandatory = $false)]
    [switch]$Fetch,

    [Parameter(Mandatory = $false)]
    [switch]$Json,

    [Parameter(Mandatory = $false)]
    [switch]$Strict,

    [Parameter(Mandatory = $false)]
    [switch]$IncludeGhReleases
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    return (Resolve-Path -LiteralPath $Path -ErrorAction Stop).ProviderPath
}

function Invoke-GitLines {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)
    $output = & git @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "git command failed: git $($Arguments -join ' ')"
    }
    return @($output)
}

function Get-TagSha {
    param([Parameter(Mandatory = $true)][string]$Tag)
    $sha = (& git rev-list -n 1 $Tag 2>$null)
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($sha)) {
        return ''
    }
    return $sha.Trim()
}

function Get-LegacySuggestion {
    param([Parameter(Mandatory = $true)][string]$Tag)

    $known = @{
        'v1.3.0-powershell-skill-kit' = 'kit/powershell-skill-kit/v1.3.0'
        'v2.0.0-kit-system' = 'kit/system/v2.0.0'
        'v0.11.1-agent-dev-kit' = 'kit/agent-dev-kit/v0.11.1'
        'v2026.07.03-fast-path-v1' = 'milestone/2026.07.03-fast-path-v1'
        'v2026.07.03-skill-external-mvp' = 'milestone/2026.07.03-skill-external-mvp'
        'v2026.07.03-agent-dev-kit-plan-mode' = 'milestone/2026.07.03-agent-dev-kit-plan-mode'
        'v2026.07.03-agent-dev-kit-plan-mode-zh-cn-patch' = 'milestone/2026.07.03-agent-dev-kit-plan-mode-zh-cn-patch'
        'v2026.07.01-common-control' = 'milestone/2026.07.01-common-control'
        'v2026.07.01-ai-debug-repair-kit' = 'milestone/2026.07.01-ai-debug-repair-kit'
        'v2026.06.29-ai-debug-repair-kit' = 'milestone/2026.06.29-ai-debug-repair-kit'
        'v2026.06.26-runtime' = 'milestone/2026.06.26-runtime'
        'v2026.06.27' = 'milestone/2026.06.27-platform-snapshot'
        'v2026.06.26' = 'milestone/2026.06.26-platform-snapshot'
    }

    if ($known.ContainsKey($Tag)) {
        return $known[$Tag]
    }

    if ($Tag -match '^v(\d{4}\.\d{2}\.\d{2})-(.+)$') {
        return "milestone/$($Matches[1])-$($Matches[2])"
    }

    if ($Tag -match '^v(\d{4}\.\d{2}\.\d{2})$') {
        return "milestone/$($Matches[1])-platform-snapshot"
    }

    if ($Tag -match '^v(?!\d{4}\.)(\d+\.\d+\.\d+)-(.+)$') {
        $version = $Matches[1]
        $name = $Matches[2]
        $name = $name -replace '^kit-', ''
        return "kit/$name/v$version"
    }

    return ''
}

function Classify-Tag {
    param([Parameter(Mandatory = $true)][string]$Tag)

    $kind = 'unknown'
    $reason = 'Does not match platform, kit, milestone, legacy component, or historical date policy.'
    $suggestion = ''

    if ($Tag -match '^kit/[A-Za-z0-9._-]+/v(?!\d{4}\.)(\d+)\.(\d+)\.(\d+)$') {
        $kind = 'kit'
        $reason = 'Namespaced kit/component semantic version tag.'
    } elseif ($Tag -match '^milestone/\d{4}\.\d{2}\.\d{2}-[A-Za-z0-9._-]+$') {
        $kind = 'milestone'
        $reason = 'Namespaced date milestone tag.'
    } elseif ($Tag -match '^v(?!\d{4}\.)(\d+)\.(\d+)\.(\d+)-[A-Za-z0-9._-]+$') {
        $kind = 'legacy-misnamed'
        $reason = 'Legacy component tag used the platform v* namespace.'
        $suggestion = Get-LegacySuggestion -Tag $Tag
    } elseif (($Tag -match '^v\d{4}\.\d{2}\.\d{2}$') -or ($Tag -match '^v\d{4}\.\d{2}\.\d{2}-[A-Za-z0-9._-]+$')) {
        $kind = 'legacy-historical'
        $reason = 'Legacy bare-date or date-suffixed tag retained as a historical snapshot, not a platform semver release.'
        $suggestion = Get-LegacySuggestion -Tag $Tag
    } elseif ($Tag -match '^v(?!\d{4}\.)(\d+)\.(\d+)\.(\d+)$') {
        $kind = 'platform'
        $reason = 'Strict platform semantic version tag.'
    }

    return [pscustomobject]@{
        tag = $Tag
        kind = $kind
        sha = Get-TagSha -Tag $Tag
        suggestion = $suggestion
        reason = $reason
    }
}

function Write-AuditFiles {
    param(
        [Parameter(Mandatory = $true)][object[]]$Rows,
        [Parameter(Mandatory = $true)][string]$OutputRoot
    )

    if (-not (Test-Path -LiteralPath $OutputRoot)) {
        New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null
    }

    $summary = [pscustomobject]@{
        generatedAt = (Get-Date).ToString('o')
        total = @($Rows).Count
        platform = @($Rows | Where-Object { $_.kind -eq 'platform' }).Count
        kit = @($Rows | Where-Object { $_.kind -eq 'kit' }).Count
        milestone = @($Rows | Where-Object { $_.kind -eq 'milestone' }).Count
        historical = @($Rows | Where-Object { $_.kind -eq 'legacy-historical' }).Count
        legacyMisnamed = @($Rows | Where-Object { $_.kind -eq 'legacy-misnamed' }).Count
        unknown = @($Rows | Where-Object { $_.kind -eq 'unknown' }).Count
        tags = $Rows
    }

    $jsonPath = Join-Path $OutputRoot 'tag-audit.json'
    $summary | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $jsonPath -Encoding UTF8

    $md = New-Object System.Collections.Generic.List[string]
    $md.Add('# Tag Audit')
    $md.Add('')
    $md.Add(('Generated: `{0}`' -f $summary.generatedAt))
    $md.Add('')
    $md.Add('| Tag | Kind | Suggestion | Commit | Reason |')
    $md.Add('|---|---|---|---|---|')
    foreach ($row in $Rows) {
        $sha = $row.sha
        if ($sha.Length -gt 12) { $sha = $sha.Substring(0, 12) }
        $md.Add(('| `{0}` | {1} | `{2}` | `{3}` | {4} |' -f $row.tag, $row.kind, $row.suggestion, $sha, ($row.reason -replace '\|', '/')))
    }
    Set-Content -LiteralPath (Join-Path $OutputRoot 'tag-audit.md') -Value $md -Encoding UTF8

    return $summary
}

function Write-PlanFile {
    param(
        [Parameter(Mandatory = $true)][object[]]$Rows,
        [Parameter(Mandatory = $true)][string]$OutputRoot
    )

    if (-not (Test-Path -LiteralPath $OutputRoot)) {
        New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null
    }

    $legacyRows = @($Rows | Where-Object { ($_.kind -eq 'legacy-misnamed' -or $_.kind -eq 'legacy-historical') -and -not [string]::IsNullOrWhiteSpace($_.suggestion) })
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add('# Tag Migration Plan')
    $lines.Add('')
    $lines.Add('This plan is non-destructive. It does not delete, force-update, or push tags by itself.')
    $lines.Add('Review every command before running it.')
    $lines.Add('')

    if ($legacyRows.Count -eq 0) {
        $lines.Add('No legacy misnamed tags with automatic suggestions were found.')
    } else {
        $lines.Add('## Suggested local tag creation commands')
        $lines.Add('')
        $lines.Add('```powershell')
        foreach ($row in $legacyRows) {
            if (-not [string]::IsNullOrWhiteSpace($row.sha)) {
                $lines.Add(('git tag {0} {1} # from {2}' -f $row.suggestion, $row.sha, $row.tag))
            }
        }
        $lines.Add('```')
        $lines.Add('')
        $lines.Add('## Suggested push commands after human confirmation')
        $lines.Add('')
        $lines.Add('```powershell')
        foreach ($row in $legacyRows) {
            $lines.Add(('git push origin {0}' -f $row.suggestion))
        }
        $lines.Add('```')
        $lines.Add('')
        $lines.Add('## Legacy tags retained')
        $lines.Add('')
        foreach ($row in $legacyRows) {
            $lines.Add(('- `{0}` -> `{1}`' -f $row.tag, $row.suggestion))
        }
    }

    $path = Join-Path $OutputRoot 'tag-migration-plan.md'
    Set-Content -LiteralPath $path -Value $lines -Encoding UTF8
    return $path
}

$RepoRootPath = Resolve-FullPath -Path $RepoRoot
Push-Location $RepoRootPath
try {
    if ($Fetch) {
        & git fetch --tags --prune | Out-Host
        if ($LASTEXITCODE -ne 0) {
            throw 'git fetch --tags --prune failed.'
        }
    }

    $tags = Invoke-GitLines -Arguments @('tag', '--list', '--sort=-creatordate')
    $rows = @()
    foreach ($tag in $tags) {
        if (-not [string]::IsNullOrWhiteSpace($tag)) {
            $rows += (Classify-Tag -Tag $tag.Trim())
        }
    }

    $outputRoot = Join-Path $RepoRootPath $OutputDir
    $summary = Write-AuditFiles -Rows $rows -OutputRoot $outputRoot

    if ($Action -eq 'Plan') {
        $planPath = Write-PlanFile -Rows $rows -OutputRoot $outputRoot
        if (-not $Json) {
            Write-Host "Migration plan written: $planPath"
        }
    }

    if ($Action -eq 'Verify' -and $Strict) {
        if ($summary.legacyMisnamed -gt 0 -or $summary.unknown -gt 0) {
            throw "Strict tag governance failed. legacyMisnamed=$($summary.legacyMisnamed), unknown=$($summary.unknown)"
        }
    }

    if ($Json) {
        $summary | ConvertTo-Json -Depth 8
    } else {
        Write-Host "Tag audit written: $outputRoot"
        Write-Host "Total=$($summary.total) Platform=$($summary.platform) Kit=$($summary.kit) Milestone=$($summary.milestone) Historical=$($summary.historical) Legacy=$($summary.legacyMisnamed) Unknown=$($summary.unknown)"
        if ($summary.legacyMisnamed -gt 0) {
            Write-Host ("Legacy misnamed tags were found. Run -Action Plan and review {0}." -f (Join-Path $outputRoot 'tag-migration-plan.md'))
        }
    }
} finally {
    Pop-Location
}
