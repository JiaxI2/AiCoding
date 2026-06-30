[CmdletBinding(DefaultParameterSetName = 'Command')]
param(
    [Parameter(ParameterSetName = 'Command', Mandatory = $true)]
    [string]$Command,

    [Parameter(ParameterSetName = 'Path', Mandatory = $true)]
    [string]$Path,

    [ValidateSet('Json', 'Markdown')]
    [string]$Format = 'Json'
)

$ErrorActionPreference = 'Stop'

if ($PSCmdlet.ParameterSetName -eq 'Path') {
    $text = Get-Content -LiteralPath $Path -Raw -ErrorAction Stop
    $name = $Path
} else {
    $text = $Command
    $name = '<Command>'
}

$tokens = $null
$parseErrors = $null
$null = [System.Management.Automation.Language.Parser]::ParseInput($text, [ref]$tokens, [ref]$parseErrors)

$recommendations = New-Object System.Collections.Generic.List[object]
function Add-Recommendation {
    param([string]$Problem, [string]$Replacement, [string]$Why, [bool]$BlocksExecution = $false)
    $recommendations.Add([pscustomobject]@{
        Problem = $Problem
        Replacement = $Replacement
        Why = $Why
        BlocksExecution = $BlocksExecution
    }) | Out-Null
}

$mapping = @(
    @{ Pattern = '(?i)\bls\s+-la\b'; Problem = 'Bash ls -la'; Replacement = 'Get-ChildItem -Force'; Why = 'PowerShell uses objects and explicit cmdlets.'; Block = $true },
    @{ Pattern = '(?i)\bgrep\b'; Problem = 'grep usage'; Replacement = 'Select-String -Pattern <pattern> -Path <path>'; Why = 'Use PowerShell-native text search.'; Block = $true },
    @{ Pattern = '(?i)\brm\s+-rf\b'; Problem = 'rm -rf destructive usage'; Replacement = 'Remove-Item -LiteralPath <path> -Recurse -Force -WhatIf'; Why = 'Destructive operations require WhatIf/review.'; Block = $true },
    @{ Pattern = '(?i)\bcat\b'; Problem = 'cat alias'; Replacement = 'Get-Content -LiteralPath <path>'; Why = 'Avoid aliases and preserve path safety.'; Block = $true },
    @{ Pattern = '(?i)\bcp\b'; Problem = 'cp alias'; Replacement = 'Copy-Item -LiteralPath <source> -Destination <dest>'; Why = 'Avoid aliases.'; Block = $true },
    @{ Pattern = '(?i)\bmv\b'; Problem = 'mv alias'; Replacement = 'Move-Item -LiteralPath <source> -Destination <dest>'; Why = 'Avoid aliases.'; Block = $true },
    @{ Pattern = '(?i)curl\s+-X'; Problem = 'curl -X style HTTP call'; Replacement = 'Invoke-RestMethod -Method <method> -Uri <uri> -Body <body>'; Why = 'PowerShell HTTP calls should be explicit. Use curl.exe only when native curl is intended.'; Block = $false },
    @{ Pattern = '&&|\|\|'; Problem = 'native shell chain operator'; Replacement = 'Split into separate commands with explicit error handling.'; Why = 'Improves sandbox approval and failure localization.'; Block = $false },
    @{ Pattern = '\$[A-Za-z_][A-Za-z0-9_]*:'; Problem = 'unsafe variable interpolation before colon'; Replacement = 'Use ${var}: or "{0}:" -f $var'; Why = 'Prevents PowerShell from parsing the colon as part of a scoped variable expression.'; Block = $false }
)

foreach ($rule in $mapping) {
    if ($text -match $rule.Pattern) {
        Add-Recommendation -Problem $rule.Problem -Replacement $rule.Replacement -Why $rule.Why -BlocksExecution ([bool]$rule.Block)
    }
}

$findings = @()
if ($parseErrors.Count -gt 0) {
    foreach ($parseError in $parseErrors) {
        $findings += [pscustomobject]@{
            Severity = 'Error'
            Rule = 'ParseError'
            Message = $parseError.Message
            Line = $parseError.Extent.StartLineNumber
            Extent = $parseError.Extent.Text
            Block = $true
        }
    }
}

$block = @($recommendations.ToArray() | Where-Object { $_.BlocksExecution }).Count -gt 0
if ($parseErrors.Count -gt 0) { $block = $true }

$plan = [pscustomobject]@{
    Name = $name
    Block = [bool]$block
    Original = $text
    Findings = $findings
    Recommendations = $recommendations.ToArray()
    Validation = @(
        'Run AST gate before execution.',
        'Run safety gate before execution.',
        'Run PSScriptAnalyzer gate for script files.',
        'For destructive operations, use -WhatIf first and prepare rollback.'
    )
    SafeExecutionPolicy = 'This tool only generates a plan. It never executes the original command or rewrite.'
}

if ($Format -eq 'Json') {
    $plan | ConvertTo-Json -Depth 20
} else {
    "# Safe Rewrite Plan"
    ""
    "- Block execution: **$($plan.Block)**"
    "- Target: ``$($plan.Name)``"
    ""
    "## Original"
    ""
    '```powershell'
    $plan.Original
    '```'
    ""
    "## Recommendations"
    if ($plan.Recommendations.Count -eq 0) { "No rewrite recommendation generated." }
    foreach ($rec in $plan.Recommendations) {
        ""
        "### $($rec.Problem)"
        "- Replacement: ``$($rec.Replacement)``"
        "- Why: $($rec.Why)"
        "- Blocks execution: $($rec.BlocksExecution)"
    }
    ""
    "## Validation"
    foreach ($v in $plan.Validation) { "- $v" }
}

if ($block) { throw 'Safe rewrite plan blocks execution of the original command.' }
