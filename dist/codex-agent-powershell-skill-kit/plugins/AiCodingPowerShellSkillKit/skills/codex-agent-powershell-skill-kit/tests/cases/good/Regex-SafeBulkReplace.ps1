[CmdletBinding(SupportsShouldProcess)]
param()

$modulePath = Join-Path (Split-Path -Parent $PSScriptRoot) '..\..\tools\PowerShellRegexOptimizeKit.psm1'
$modulePath = (Resolve-Path -LiteralPath $modulePath).Path
Import-Module $modulePath -Force

$text = 'abc123'
$result = Invoke-SafeRegexReplace -InputText $text -Pattern '([a-z]+)(\d+)' -ReplaceToken '$1-$2'
if ($result -ne 'abc-123') {
    throw "unexpected safe replace result: $result"
}

$tmp = New-TemporaryFile
try {
    Set-Content -LiteralPath $tmp.FullName -Value 'api_v1_user' -NoNewline
    Update-FileContentBulk -FilePath $tmp.FullName -Pattern 'api_v(\d+)_user' -ReplaceToken 'api_v$1_account'
    $updated = Get-Content -LiteralPath $tmp.FullName -Raw
    if ($updated -ne 'api_v1_account') {
        throw "unexpected bulk replace result: $updated"
    }
} finally {
    if ($PSCmdlet.ShouldProcess($tmp.FullName, 'Remove temporary file')) {
        Remove-Item -LiteralPath $tmp.FullName -Force -ErrorAction SilentlyContinue
    }
}
