#requires -Version 7.0

$modulePath = Join-Path (Split-Path -Parent $PSScriptRoot) '..\..\tools\PowerShellRegexOptimizeKit.psm1'
$modulePath = (Resolve-Path -LiteralPath $modulePath).Path
Import-Module $modulePath -Force

$source = 'api_v1_user'
$result = Update-CodeDynamically -SourceCode $source -Pattern '(?:^|_)(\w)' -Callback {
    $_.Groups[1].Value.ToUpperInvariant()
}

if ($result -ne 'ApiV1User') {
    throw "unexpected dynamic transform result: $result"
}
