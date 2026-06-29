param(
  [ValidateSet('system','user','project')] [string]$Scope = 'user',
  [string]$ProjectRoot = '.',
  [string]$Reason = 'enabled-by-script'
)

$ErrorActionPreference = 'Stop'
$KitRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Push-Location $KitRoot
try {
  if (-not (Get-Command apatch -ErrorAction SilentlyContinue)) {
    python -m pip install -e .
  }
  apatch state enable --scope $Scope --path $ProjectRoot --reason $Reason
  apatch state status --path $ProjectRoot
}
finally { Pop-Location }
