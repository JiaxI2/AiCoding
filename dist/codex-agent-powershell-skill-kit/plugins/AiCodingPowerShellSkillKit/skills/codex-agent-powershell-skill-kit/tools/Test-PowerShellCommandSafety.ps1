[CmdletBinding(DefaultParameterSetName = 'Path')]
param(
    [Parameter(ParameterSetName = 'Path')]
    [string[]]$Path,

    [Parameter(ParameterSetName = 'Definition')]
    [string]$ScriptDefinition,

    [Parameter(ParameterSetName = 'Command')]
    [string]$Command,

    [Parameter(ParameterSetName = 'Path')]
    [switch]$Recurse,

    [switch]$Json
)

$ErrorActionPreference = 'Stop'

$AliasMap = @{
    'ls' = 'Get-ChildItem'
    'cat' = 'Get-Content'
    'rm' = 'Remove-Item'
    'cp' = 'Copy-Item'
    'mv' = 'Move-Item'
    'grep' = 'Select-String'
    'awk' = 'PowerShell object pipeline / ForEach-Object'
    'sed' = 'PowerShell -replace / Set-Content with explicit encoding'
    'wget' = 'Invoke-WebRequest'
    'curl' = 'Invoke-RestMethod or curl.exe when native curl is intentional'
}

$Destructive = @(
    'Remove-Item', 'Clear-Content', 'Set-ExecutionPolicy', 'Set-ItemProperty', 'Remove-ItemProperty',
    'New-ItemProperty', 'Stop-Service', 'Restart-Service', 'Stop-Process', 'Format-Volume',
    'Resize-Partition', 'diskpart', 'reg', 'reg.exe', 'icacls', 'takeown', 'netsh', 'bcdedit'
)

function Get-ScriptTextFromPath {
    param([string]$Item)
    $resolved = Resolve-Path -LiteralPath $Item -ErrorAction Stop
    if (Test-Path -LiteralPath $resolved.Path -PathType Container) {
        Get-ChildItem -LiteralPath $resolved.Path -Include *.ps1, *.psm1, *.psd1 -File -Recurse:$Recurse | ForEach-Object {
            [pscustomobject]@{ Name = $_.FullName; Text = Get-Content -LiteralPath $_.FullName -Raw -ErrorAction Stop }
        }
    } else {
        [pscustomobject]@{ Name = $resolved.Path; Text = Get-Content -LiteralPath $resolved.Path -Raw -ErrorAction Stop }
    }
}

function Add-Finding {
    param(
        [System.Collections.Generic.List[object]]$List,
        [string]$Severity,
        [string]$Rule,
        [string]$Message,
        [string]$Extent,
        [int]$Line = 0,
        [bool]$Block = $false
    )
    $List.Add([pscustomobject]@{
        Severity = $Severity
        Rule = $Rule
        Message = $Message
        Line = $Line
        Extent = $Extent
        Block = $Block
    }) | Out-Null
}

function Test-TextSafety {
    param([string]$Text, [string]$Name)

    $findings = [System.Collections.Generic.List[object]]::new()
    $tokens = $null
    $errors = $null
    $ast = [System.Management.Automation.Language.Parser]::ParseInput($Text, [ref]$tokens, [ref]$errors)
    if ($errors.Count -gt 0) {
        Add-Finding -List $findings -Severity 'Error' -Rule 'ParseError' -Message 'Script has parse errors. Run AST gate first.' -Extent '<parse-error>' -Block $true
        return [pscustomobject]@{ Path = $Name; Ok = $false; Findings = $findings.ToArray() }
    }

    $scriptHasShouldProcess = ($Text -match 'SupportsShouldProcess') -and ($Text -match 'ShouldProcess\s*\(')
    $commands = $ast.FindAll({ param($node) $node -is [System.Management.Automation.Language.CommandAst] }, $true)
    foreach ($commandAst in $commands) {
        $name = $commandAst.GetCommandName()
        if ([string]::IsNullOrWhiteSpace($name)) { continue }
        $line = $commandAst.Extent.StartLineNumber
        $extentText = $commandAst.Extent.Text

        if ($AliasMap.ContainsKey($name)) {
            Add-Finding -List $findings -Severity 'Error' -Rule 'NoBashOrAliasLeakage' -Message "Command '$name' is not allowed. Prefer '$($AliasMap[$name])'." -Extent $extentText -Line $line -Block $true
        }

        if ($Destructive -contains $name) {
            $hasWhatIf = $extentText -match '(?i)(^|\s)-WhatIf(\s|$|:)'
            $hasConfirm = $extentText -match '(?i)(^|\s)-Confirm(\s|$|:)'
            if (-not ($hasWhatIf -or $hasConfirm -or $scriptHasShouldProcess)) {
                Add-Finding -List $findings -Severity 'Error' -Rule 'DestructiveRequiresSafety' -Message "Destructive command '$name' requires -WhatIf, -Confirm, ShouldProcess, or explicit approval." -Extent $extentText -Line $line -Block $true
            }
        }
    }

    if ($Text -match '&&|\|\|') {
        Add-Finding -List $findings -Severity 'Warning' -Rule 'AvoidNativeShellChains' -Message 'Command chain operators detected. Prefer separate commands or a reviewed script.' -Extent '&&/||' -Block $false
    }

    $semicolonMatches = [regex]::Matches($Text, ';')
    if ($semicolonMatches.Count -gt 3 -and ($Text -split "`n").Count -le 3) {
        Add-Finding -List $findings -Severity 'Warning' -Rule 'AvoidDenseOneLiners' -Message 'Dense one-line command detected. Prefer separate statements for debugging and approval.' -Extent ';' -Block $false
    }

    if ($Text -match '\$[A-Za-z_][A-Za-z0-9_]*:') {
        Add-Finding -List $findings -Severity 'Warning' -Rule 'SafeInterpolation' -Message 'Variable followed by colon detected. Prefer ${var}: or -f formatting.' -Extent '$var:' -Block $false
    }

    $blocking = @($findings.ToArray() | Where-Object { $_.Block }).Count -eq 0
    [pscustomobject]@{ Path = $Name; Ok = [bool]$blocking; Findings = $findings.ToArray() }
}

$targets = @()
switch ($PSCmdlet.ParameterSetName) {
    'Command' { $targets += [pscustomobject]@{ Name = '<Command>'; Text = $Command } }
    'Definition' { $targets += [pscustomobject]@{ Name = '<ScriptDefinition>'; Text = $ScriptDefinition } }
    default {
        foreach ($item in $Path) { $targets += Get-ScriptTextFromPath -Item $item }
    }
}

$results = foreach ($target in $targets) { Test-TextSafety -Text $target.Text -Name $target.Name }
$ok = -not ($results | Where-Object { -not $_.Ok })
$summary = [pscustomobject]@{
    Gate = 'Safety'
    Ok = [bool]$ok
    Count = $results.Count
    Failed = @($results | Where-Object { -not $_.Ok }).Count
    Results = @($results)
}

if ($Json) { $summary | ConvertTo-Json -Depth 20 } else { $summary | Format-List -Property Gate, Ok, Count, Failed; $results | ForEach-Object { if ($_.Findings.Count -gt 0) { $_ | Format-List } } }
if (-not $ok) { throw 'Gate failed.' }
