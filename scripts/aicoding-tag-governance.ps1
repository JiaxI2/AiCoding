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

function Get-CurrentTagSuggestion {
    param([Parameter(Mandatory = $true)][string]$Tag)

    if ($Tag -match '^v(\d{4}\.\d{2}\.\d{2})-(.+)$') {
        return "milestone/$($Matches[1])-$($Matches[2])"
    }

    if ($Tag -match '^v(\d{4}\.\d{2}\.\d{2})$') {
        return "milestone/$($Matches[1])-snapshot"
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
    $reason = 'Does not match the current platform, kit, or milestone tag policy.'
    $suggestion = ''

    if ($Tag -match '^kit/[A-Za-z0-9._-]+/v(?!\d{4}\.)(\d+)\.(\d+)\.(\d+)$') {
        $kind = 'kit'
        $reason = 'Namespaced kit/component semantic version tag.'
    } elseif ($Tag -match '^milestone/\d{4}\.\d{2}\.\d{2}-[A-Za-z0-9._-]+$') {
        $kind = 'milestone'
        $reason = 'Namespaced date milestone tag.'
    } elseif ($Tag -match '^v(?!\d{4}\.)(\d+)\.(\d+)\.(\d+)-[A-Za-z0-9._-]+$') {
        $kind = 'noncurrent-component'
        $reason = 'Component tag is outside the current kit/<kit-id>/vMAJOR.MINOR.PATCH namespace.'
        $suggestion = Get-CurrentTagSuggestion -Tag $Tag
    } elseif (($Tag -match '^v\d{4}\.\d{2}\.\d{2}$') -or ($Tag -match '^v\d{4}\.\d{2}\.\d{2}-[A-Za-z0-9._-]+$')) {
        $kind = 'noncurrent-date'
        $reason = 'Date tag is outside the current milestone/YYYY.MM.DD-<name> namespace.'
        $suggestion = Get-CurrentTagSuggestion -Tag $Tag
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
        nonCurrentDate = @($Rows | Where-Object { $_.kind -eq 'noncurrent-date' }).Count
        nonCurrentComponent = @($Rows | Where-Object { $_.kind -eq 'noncurrent-component' }).Count
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

    $nonCurrentRows = @($Rows | Where-Object { ($_.kind -eq 'noncurrent-component' -or $_.kind -eq 'noncurrent-date') -and -not [string]::IsNullOrWhiteSpace($_.suggestion) })
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add('# Tag Alignment Plan')
    $lines.Add('')
    $lines.Add('This plan is non-destructive. It does not delete, force-update, or push tags by itself.')
    $lines.Add('Review every command before running it.')
    $lines.Add('')

    if ($nonCurrentRows.Count -eq 0) {
        $lines.Add('No non-current tags with automatic suggestions were found.')
    } else {
        $lines.Add('## Suggested local tag creation commands')
        $lines.Add('')
        $lines.Add('```powershell')
        foreach ($row in $nonCurrentRows) {
            if (-not [string]::IsNullOrWhiteSpace($row.sha)) {
                $lines.Add(('git tag {0} {1} # from {2}' -f $row.suggestion, $row.sha, $row.tag))
            }
        }
        $lines.Add('```')
        $lines.Add('')
        $lines.Add('## Suggested push commands after human confirmation')
        $lines.Add('')
        $lines.Add('```powershell')
        foreach ($row in $nonCurrentRows) {
            $lines.Add(('git push origin {0}' -f $row.suggestion))
        }
        $lines.Add('```')
        $lines.Add('')
        $lines.Add('## Non-current tags retained')
        $lines.Add('')
        foreach ($row in $nonCurrentRows) {
            $lines.Add(('- `{0}` -> `{1}`' -f $row.tag, $row.suggestion))
        }
    }

    $path = Join-Path $OutputRoot 'tag-alignment-plan.md'
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
            Write-Host "Alignment plan written: $planPath"
        }
    }

    if ($Action -eq 'Verify' -and $Strict) {
        if ($summary.nonCurrentDate -gt 0 -or $summary.nonCurrentComponent -gt 0 -or $summary.unknown -gt 0) {
            throw "Strict tag governance failed. nonCurrentDate=$($summary.nonCurrentDate), nonCurrentComponent=$($summary.nonCurrentComponent), unknown=$($summary.unknown)"
        }
    }

    if ($Json) {
        $summary | ConvertTo-Json -Depth 8
    } else {
        Write-Host "Tag audit written: $outputRoot"
        Write-Host "Total=$($summary.total) Platform=$($summary.platform) Kit=$($summary.kit) Milestone=$($summary.milestone) NonCurrentDate=$($summary.nonCurrentDate) NonCurrentComponent=$($summary.nonCurrentComponent) Unknown=$($summary.unknown)"
        if ($summary.nonCurrentDate -gt 0 -or $summary.nonCurrentComponent -gt 0) {
            Write-Host ("Non-current tags were found. Run -Action Plan and review {0}." -f (Join-Path $outputRoot 'tag-alignment-plan.md'))
        }
    }
} finally {
    Pop-Location
}