[CmdletBinding(DefaultParameterSetName = 'Path')]
param(
    [Parameter(ParameterSetName = 'Path', Mandatory = $true)]
    [string[]]$Path,

    [Parameter(ParameterSetName = 'Definition', Mandatory = $true)]
    [string]$ScriptDefinition,

    [Parameter(ParameterSetName = 'Path')]
    [switch]$Recurse,

    [switch]$InstallMissingTools,
    [switch]$FailOnWarning,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
$module = Get-Module -ListAvailable -Name PSScriptAnalyzer | Sort-Object Version -Descending | Select-Object -First 1
if (-not $module) {
    if ($InstallMissingTools) {
        Install-Module -Name PSScriptAnalyzer -Scope CurrentUser -Force -AllowClobber -ErrorAction Stop
    } else {
        throw 'PSScriptAnalyzer is required but not installed. Re-run with -InstallMissingTools or install it with: Install-Module PSScriptAnalyzer -Scope CurrentUser'
    }
}
Import-Module PSScriptAnalyzer -ErrorAction Stop

$settings = Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) '..\PSScriptAnalyzerSettings.psd1'
$settings = (Resolve-Path -LiteralPath $settings).Path

$diagnostics = @()
if ($PSCmdlet.ParameterSetName -eq 'Definition') {
    $diagnostics = @(Invoke-ScriptAnalyzer -ScriptDefinition $ScriptDefinition -Settings $settings -ErrorAction Stop)
} else {
    foreach ($item in $Path) {
        $resolved = Resolve-Path -LiteralPath $item -ErrorAction Stop
        $diagnostics += @(Invoke-ScriptAnalyzer -Path $resolved.Path -Recurse:$Recurse -Settings $settings -ErrorAction Stop)
    }
}

$failSeverities = @('Error')
if ($FailOnWarning) { $failSeverities += 'Warning' }
$failed = @($diagnostics | Where-Object { $failSeverities -contains $_.Severity.ToString() })

$result = [pscustomobject]@{
    Gate = 'PSScriptAnalyzer'
    Ok = ($failed.Count -eq 0)
    Count = $diagnostics.Count
    Failed = $failed.Count
    Diagnostics = @($diagnostics | Select-Object RuleName, Severity, Message, ScriptName, Line, Column)
}

if ($Json) { $result | ConvertTo-Json -Depth 20 } else { $result | Format-List -Property Gate, Ok, Count, Failed; $result.Diagnostics | Format-Table -AutoSize }
if ($failed.Count -gt 0) { throw 'PSScriptAnalyzer gate failed.' }
