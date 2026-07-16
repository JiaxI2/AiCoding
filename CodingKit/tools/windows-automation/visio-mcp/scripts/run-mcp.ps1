#requires -Version 7.0
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
& ".venv\Scripts\python.exe" -m visio_mcp server
