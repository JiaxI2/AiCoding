param([string]$KitRoot = '')
$ErrorActionPreference = 'Stop'
if (-not $KitRoot) { $KitRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path) }
Push-Location $KitRoot
try {
  git pull --ff-only
  python -m pip install -e .
  apatch doctor
}
finally { Pop-Location }
