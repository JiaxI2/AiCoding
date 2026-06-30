[CmdletBinding(DefaultParameterSetName = 'Path')]
param(
    [Parameter(ParameterSetName = 'Path', Mandatory = $true)]
    [string[]]$Path,

    [Parameter(ParameterSetName = 'Definition', Mandatory = $true)]
    [string]$ScriptDefinition,

    [Parameter(ParameterSetName = 'Command', Mandatory = $true)]
    [string]$Command,

    [Parameter(ParameterSetName = 'Path')]
    [switch]$Recurse,

    [switch]$InstallMissingTools,
    [switch]$FailOnWarning,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
$toolRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$results = New-Object System.Collections.Generic.List[object]

function Invoke-Gate {
    param([string]$Name, [scriptblock]$Action)
    try {
        & $Action
        $results.Add([pscustomobject]@{ Gate = $Name; Ok = $true; Message = 'passed' }) | Out-Null
    } catch {
        $results.Add([pscustomobject]@{ Gate = $Name; Ok = $false; Message = $_.Exception.Message }) | Out-Null
    }
}

Invoke-Gate -Name 'Runtime' -Action { & (Join-Path $toolRoot 'Test-PowerShellRuntime.ps1') | Out-Null }

switch ($PSCmdlet.ParameterSetName) {
    'Command' {
        Invoke-Gate -Name 'AST' -Action { & (Join-Path $toolRoot 'Invoke-PowerShellAstGate.ps1') -ScriptDefinition $Command | Out-Null }
        Invoke-Gate -Name 'Safety' -Action { & (Join-Path $toolRoot 'Test-PowerShellCommandSafety.ps1') -Command $Command | Out-Null }
        Invoke-Gate -Name 'RewritePlan' -Action { & (Join-Path $toolRoot 'Invoke-SafeRewritePlan.ps1') -Command $Command -Format Json | Out-Null }
    }
    'Definition' {
        Invoke-Gate -Name 'AST' -Action { & (Join-Path $toolRoot 'Invoke-PowerShellAstGate.ps1') -ScriptDefinition $ScriptDefinition | Out-Null }
        Invoke-Gate -Name 'Safety' -Action { & (Join-Path $toolRoot 'Test-PowerShellCommandSafety.ps1') -ScriptDefinition $ScriptDefinition | Out-Null }
        Invoke-Gate -Name 'PSScriptAnalyzer' -Action { & (Join-Path $toolRoot 'Invoke-PSScriptAnalyzerGate.ps1') -ScriptDefinition $ScriptDefinition -InstallMissingTools:$InstallMissingTools -FailOnWarning:$FailOnWarning | Out-Null }
    }
    default {
        Invoke-Gate -Name 'AST' -Action { & (Join-Path $toolRoot 'Invoke-PowerShellAstGate.ps1') -Path $Path -Recurse:$Recurse | Out-Null }
        Invoke-Gate -Name 'Safety' -Action { & (Join-Path $toolRoot 'Test-PowerShellCommandSafety.ps1') -Path $Path -Recurse:$Recurse | Out-Null }
        Invoke-Gate -Name 'PSScriptAnalyzer' -Action { & (Join-Path $toolRoot 'Invoke-PSScriptAnalyzerGate.ps1') -Path $Path -Recurse:$Recurse -InstallMissingTools:$InstallMissingTools -FailOnWarning:$FailOnWarning | Out-Null }
    }
}

$ok = (@($results.ToArray() | Where-Object { -not $_.Ok }).Count -eq 0)
$summary = [pscustomobject]@{ Gate = 'FullPowerShellSkillKit'; Ok = [bool]$ok; Results = $results.ToArray() }
if ($Json) { $summary | ConvertTo-Json -Depth 20 } else { $summary.Results | Format-Table -AutoSize }
if (-not $ok) { throw 'Gate failed.' }
