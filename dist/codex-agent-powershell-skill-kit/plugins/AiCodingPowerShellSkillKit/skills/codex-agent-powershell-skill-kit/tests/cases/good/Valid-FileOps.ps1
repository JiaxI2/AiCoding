[CmdletBinding(SupportsShouldProcess)]
param(
    [string]$Root = (Get-Location).Path
)

$target = Join-Path -Path $Root -ChildPath 'README.md'
if (Test-Path -LiteralPath $target -PathType Leaf) {
    Get-Item -LiteralPath $target | Select-Object -Property FullName, Length
}
