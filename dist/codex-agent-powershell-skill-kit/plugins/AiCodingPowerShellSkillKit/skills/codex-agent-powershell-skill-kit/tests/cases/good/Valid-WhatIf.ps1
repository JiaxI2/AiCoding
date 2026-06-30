[CmdletBinding(SupportsShouldProcess)]
param([string]$Path = '.\temp-demo')

if ($PSCmdlet.ShouldProcess($Path, 'Remove demo directory')) {
    Remove-Item -LiteralPath $Path -Recurse -Force -WhatIf
}
