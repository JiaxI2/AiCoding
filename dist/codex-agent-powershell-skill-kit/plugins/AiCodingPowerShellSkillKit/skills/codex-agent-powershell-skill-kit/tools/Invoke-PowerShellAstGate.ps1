[CmdletBinding(DefaultParameterSetName = 'Path')]
param(
    [Parameter(ParameterSetName = 'Path', Mandatory = $true)]
    [string[]]$Path,

    [Parameter(ParameterSetName = 'Definition', Mandatory = $true)]
    [string]$ScriptDefinition,

    [Parameter(ParameterSetName = 'Path')]
    [switch]$Recurse,

    [switch]$Json
)

$ErrorActionPreference = 'Stop'

function Convert-ParseError {
    param([System.Management.Automation.Language.ParseError]$ErrorRecord)
    [pscustomobject]@{
        Message = $ErrorRecord.Message
        ErrorId = $ErrorRecord.ErrorId
        StartLineNumber = $ErrorRecord.Extent.StartLineNumber
        StartColumnNumber = $ErrorRecord.Extent.StartColumnNumber
        EndLineNumber = $ErrorRecord.Extent.EndLineNumber
        Text = $ErrorRecord.Extent.Text
    }
}

function Test-ScriptText {
    param(
        [string]$Text,
        [string]$Name
    )
    $tokens = $null
    $errors = $null
    $null = [System.Management.Automation.Language.Parser]::ParseInput($Text, [ref]$tokens, [ref]$errors)
    [pscustomobject]@{
        Path = $Name
        Ok = ($errors.Count -eq 0)
        ErrorCount = $errors.Count
        Errors = @($errors | ForEach-Object { Convert-ParseError $_ })
    }
}

function Test-ScriptFile {
    param([string]$FilePath)
    $resolved = Resolve-Path -LiteralPath $FilePath -ErrorAction Stop
    $tokens = $null
    $errors = $null
    $null = [System.Management.Automation.Language.Parser]::ParseFile($resolved.Path, [ref]$tokens, [ref]$errors)
    [pscustomobject]@{
        Path = $resolved.Path
        Ok = ($errors.Count -eq 0)
        ErrorCount = $errors.Count
        Errors = @($errors | ForEach-Object { Convert-ParseError $_ })
    }
}

$results = @()
if ($PSCmdlet.ParameterSetName -eq 'Definition') {
    $results += Test-ScriptText -Text $ScriptDefinition -Name '<ScriptDefinition>'
} else {
    foreach ($item in $Path) {
        if (Test-Path -LiteralPath $item -PathType Container) {
            $files = Get-ChildItem -LiteralPath $item -Include *.ps1, *.psm1, *.psd1 -File -Recurse:$Recurse
            foreach ($file in $files) { $results += Test-ScriptFile -FilePath $file.FullName }
        } else {
            $results += Test-ScriptFile -FilePath $item
        }
    }
}

$ok = -not ($results | Where-Object { -not $_.Ok })
$summary = [pscustomobject]@{
    Gate = 'AST'
    Ok = [bool]$ok
    Count = $results.Count
    Failed = @($results | Where-Object { -not $_.Ok }).Count
    Results = $results
}

if ($Json) { $summary | ConvertTo-Json -Depth 20 } else { $summary | Format-List -Property Gate, Ok, Count, Failed; $results | Where-Object { -not $_.Ok } | Format-List }
if (-not $ok) { throw 'Gate failed.' }
