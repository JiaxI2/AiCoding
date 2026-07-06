#requires -Version 7.0
Set-StrictMode -Version Latest

function Invoke-SafeRegexReplace {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$InputText,
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][string]$ReplaceToken,
        [System.Text.RegularExpressions.RegexOptions]$Options = [System.Text.RegularExpressions.RegexOptions]::None
    )

    process {
        $regex = [System.Text.RegularExpressions.Regex]::new($Pattern, $Options)
        return $regex.Replace($InputText, $ReplaceToken)
    }
}

function Update-FileContentBulk {
    [CmdletBinding(SupportsShouldProcess = $true)]
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][string]$ReplaceToken,
        [System.Text.Encoding]$Encoding = [System.Text.UTF8Encoding]::new($false)
    )

    process {
        $resolved = Resolve-Path -LiteralPath $FilePath -ErrorAction Stop
        $target = $resolved.Path
        if ($PSCmdlet.ShouldProcess($target, 'Bulk regex replace')) {
            $content = [System.IO.File]::ReadAllText($target, $Encoding)
            $updated = [System.Text.RegularExpressions.Regex]::Replace($content, $Pattern, $ReplaceToken)
            [System.IO.File]::WriteAllText($target, $updated, $Encoding)
        }
    }
}

function Update-CodeDynamically {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$SourceCode,
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][scriptblock]$Callback
    )

    process {
        if ($PSVersionTable.PSVersion.Major -lt 7) {
            throw 'Dynamic callback regex replacement requires PowerShell 7+.'
        }

        return $SourceCode -creplace $Pattern, $Callback
    }
}

function Test-PowerShellRegexOptimization {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$Text,
        [string]$Name = '<memory>'
    )

    $findings = New-Object System.Collections.Generic.List[object]
    $lines = $Text -split "`r?`n"

    for ($i = 0; $i -lt $lines.Count; $i++) {
        $line = $lines[$i]
        $lineNo = $i + 1
        if ($line -match '-(?:c|i)?replace\b[^,\r\n]*,\s*"[^"\r\n]*\$(?:\d+|\{[A-Za-z_][A-Za-z0-9_]*\})') {
            $findings.Add([pscustomobject]@{
                Rule = 'PSRegex001.DoubleQuotedCaptureReplacement'
                Severity = 'Error'
                Name = $Name
                Line = $lineNo
                Message = 'Regex replacement capture tokens must use single-quoted literals, for example ''$1'' or ''${Name}''.'
                Snippet = $line.Trim()
            }) | Out-Null
        }

        $normalized = $line.ToLowerInvariant()
        if ($normalized.Contains('get-content') -and
            $normalized.Contains('foreach-object') -and
            $normalized.Contains('-replace') -and
            -not $normalized.Contains('-raw')) {
            $findings.Add([pscustomobject]@{
                Rule = 'PSRegex002.LinePipelineReplace'
                Severity = 'Error'
                Name = $Name
                Line = $lineNo
                Message = 'Avoid Get-Content | ForEach-Object line-by-line regex replacement. Use Get-Content -Raw and a bulk replace.'
                Snippet = $line.Trim()
            }) | Out-Null
        }

        if (($line -match '-(?:c|i)?replace\b[^,\r\n]*,\s*\{') -and ($Text -notmatch '#requires\s+-Version\s+7')) {
            $findings.Add([pscustomobject]@{
                Rule = 'PSRegex003.DynamicCallbackRequiresPS7'
                Severity = 'Warning'
                Name = $Name
                Line = $lineNo
                Message = 'ScriptBlock replacement requires PowerShell 7+. Add #requires -Version 7.0 for portable agent execution.'
                Snippet = $line.Trim()
            }) | Out-Null
        }
    }

    return $findings.ToArray()
}

Export-ModuleMember -Function Invoke-SafeRegexReplace, Update-FileContentBulk, Update-CodeDynamically, Test-PowerShellRegexOptimization
