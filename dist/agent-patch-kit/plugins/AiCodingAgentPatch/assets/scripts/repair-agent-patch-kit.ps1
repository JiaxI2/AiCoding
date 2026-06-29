param(
  [switch]$Dev,
  [switch]$InstallMissing,
  [ValidateSet('none','system','user','project')] [string]$DeployScope = 'none',
  [ValidateSet('agents','codex','both')] [string]$Agent = 'both',
  [string]$ProjectRoot = '',
  [switch]$WriteAgentsSnippet
)

$ErrorActionPreference = 'Stop'
$KitRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)

Write-Host "Repairing Agent Patch Kit CLI install. This removes broken editable installs and reinstalls from this v0.2.2 kit."
python -m pip uninstall -y agent-patch-kit | Out-Host

$args = @('-NoProfile','-ExecutionPolicy','Bypass','-File',(Join-Path $KitRoot 'scripts/install-agent-patch-kit.ps1'))
if ($InstallMissing) { $args += '-InstallMissing' }
if ($Dev) { $args += '-Dev' }
$args += @('-DeployScope', $DeployScope, '-Agent', $Agent)
if ($ProjectRoot) { $args += @('-ProjectRoot', $ProjectRoot) }
if ($WriteAgentsSnippet) { $args += '-WriteAgentsSnippet' }

$shell = (Get-Command pwsh -ErrorAction SilentlyContinue).Source
if (-not $shell) { $shell = (Get-Command powershell -ErrorAction SilentlyContinue).Source }
if (-not $shell) { throw "PowerShell executable not found" }
& $shell @args
