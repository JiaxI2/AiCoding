#requires -Version 7.0
[CmdletBinding()]
param(
  [string]$Python = ""
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $true
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Resolve-Python {
  param([string]$Requested)

  $candidates = @()
  if ($Requested) {
    $candidates += $Requested
  }
  if ($env:VISIO_MCP_PYTHON) {
    $candidates += $env:VISIO_MCP_PYTHON
  }
  $pythonCommand = Get-Command python -ErrorAction SilentlyContinue
  if ($pythonCommand) {
    $candidates += $pythonCommand.Source
  }
  if ($env:LOCALAPPDATA) {
    $candidates += Get-ChildItem -Path (Join-Path $env:LOCALAPPDATA "Programs\Python\Python3*\python.exe") -ErrorAction SilentlyContinue |
      Sort-Object FullName -Descending |
      Select-Object -ExpandProperty FullName
  }

  foreach ($candidate in $candidates | Select-Object -Unique) {
    if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
      continue
    }
    $version = & $candidate -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')"
    if ([version]$version -ge [version]"3.10") {
      return $candidate
    }
  }
  throw "未找到 Python 3.10 或更高版本。可通过 -Python 或 VISIO_MCP_PYTHON 指定解释器。"
}

$pythonExe = Resolve-Python -Requested $Python
if (-not (Test-Path -LiteralPath ".venv\Scripts\python.exe" -PathType Leaf)) {
  & $pythonExe -m venv .venv
}
& ".venv\Scripts\python.exe" -m pip install -r requirements-windows.txt
& ".venv\Scripts\python.exe" -m visio_mcp doctor --json
