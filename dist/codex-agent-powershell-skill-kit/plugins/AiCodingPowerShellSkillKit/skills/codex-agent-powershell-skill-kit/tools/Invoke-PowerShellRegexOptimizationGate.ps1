#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string[]]$Path,

    [switch]$Recurse,

    [ValidateSet('Json', 'Markdown')]
    [string]$Format = 'Json',

    [switch]$FailOnWarning
)

$ErrorActionPreference = 'Stop'
$modulePath = Join-Path $PSScriptRoot 'PowerShellRegexOptimizeKit.psm1'
Import-Module $modulePath -Force

$targets = New-Object System.Collections.Generic.List[string]

foreach ($item in $Path) {
    if (Test-Path -LiteralPath $item -PathType Leaf) {
        if ($item -match '\.ps(d|m)?1$') {
            $targets.Add((Resolve-Path -LiteralPath $item).Path) | Out-Null
        }
        continue
    }

    if (Test-Path -LiteralPath $item -PathType Container) {
        $gciParams = @{
            LiteralPath = $item
            File = $true
            Include = @('*.ps1', '*.psm1', '*.psd1')
        }
        if ($Recurse) {
            $gciParams.Recurse = $true
        }
        foreach ($file in Get-ChildItem @gciParams) {
            $targets.Add($file.FullName) | Out-Null
        }
    }
}

$findings = New-Object System.Collections.Generic.List[object]
foreach ($target in $targets) {
    $content = Get-Content -LiteralPath $target -Raw -ErrorAction Stop
    foreach ($finding in Test-PowerShellRegexOptimization -Text $content -Name $target) {
        $findings.Add($finding) | Out-Null
    }
}

$blocking = @($findings.ToArray() | Where-Object { $_.Severity -eq 'Error' -or ($FailOnWarning -and $_.Severity -eq 'Warning') })

$result = [pscustomobject]@{
    Name = 'PowerShell Regex Optimization Gate'
    Ok = ($blocking.Count -eq 0)
    TargetCount = $targets.Count
    FindingCount = $findings.Count
    Findings = $findings.ToArray()
}

if ($Format -eq 'Json') {
    $result | ConvertTo-Json -Depth 20
} else {
    "# PowerShell Regex Optimization Gate"
    ""
    "- OK: **$($result.Ok)**"
    "- Targets: $($result.TargetCount)"
    "- Findings: $($result.FindingCount)"
    ""
    foreach ($finding in $result.Findings) {
        "## $($finding.Rule)"
        "- Severity: $($finding.Severity)"
        "- File: ``$($finding.Name)``"
        "- Line: $($finding.Line)"
        "- Message: $($finding.Message)"
        "- Snippet: ``$($finding.Snippet)``"
        ""
    }
}

if (-not $result.Ok) {
    throw 'PowerShell Regex Optimization Gate failed.'
}
