param(
  [string]$KitRoot = '',
  [switch]$Dev
)
$ErrorActionPreference = 'Stop'
if (-not $KitRoot) { $KitRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path) }
Push-Location $KitRoot
try {
  if (Test-Path .git) { git pull --ff-only }
  if ($Dev) {
    Write-Warning "Updating in editable/dev mode. Do not delete this source directory."
    python -m pip install --force-reinstall -e .
  } else {
    python -m pip install --force-reinstall .
  }
  apatch install doctor
  apatch doctor
}
finally { Pop-Location }
