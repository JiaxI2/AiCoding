[CmdletBinding()]
param([string]$LogPath)

$ErrorActionPreference = 'Stop'
if ($LogPath -and (Test-Path -LiteralPath $LogPath)) {
    Select-String -LiteralPath $LogPath -Pattern 'ParserError|Access is denied|deny-read|Cannot find path|CommandNotFoundException|PSScriptAnalyzer' -SimpleMatch:$false
}
