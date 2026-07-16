#requires -Version 7.0
$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
if (-not (Test-Path -LiteralPath ".venv\Scripts\python.exe" -PathType Leaf)) {
  throw "缺少 .venv。请先运行 scripts\install.ps1。"
}
& ".venv\Scripts\python.exe" -m pytest -q -p no:cacheprovider
& ".venv\Scripts\python.exe" tools\benchmark.py --renderer mock --iterations 30
