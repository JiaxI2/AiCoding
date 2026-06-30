[CmdletBinding()]
param([Parameter(Mandatory=$true)][string]$Command)

$ErrorActionPreference = 'Stop'
$tool = Join-Path $PSScriptRoot '..\tools\Invoke-SafeRewritePlan.ps1'
& $tool -Command $Command -Format Markdown
