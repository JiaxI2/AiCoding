param(
  [ValidateSet('none','user','project')] [string]$DeployScope = 'none',
  [ValidateSet('agents','codex','both')] [string]$Agent = 'both',
  [string]$ProjectRoot = ''
)
$ErrorActionPreference = 'Stop'
python -m pip uninstall -y agent-patch-kit
$skill = 'aicoding-agent-patch-kit'
$targets = @()
if ($DeployScope -eq 'user') {
  if ($Agent -in @('agents','both')) { $targets += Join-Path $HOME ".agents/skills/$skill" }
  if ($Agent -in @('codex','both')) { $targets += Join-Path $HOME ".codex/skills/$skill" }
} elseif ($DeployScope -eq 'project') {
  if (-not $ProjectRoot) { throw '-ProjectRoot is required for project uninstall' }
  if ($Agent -in @('agents','both')) { $targets += Join-Path $ProjectRoot ".agents/skills/$skill" }
  if ($Agent -in @('codex','both')) { $targets += Join-Path $ProjectRoot ".codex/skills/$skill" }
}
foreach ($t in $targets) { if (Test-Path $t) { Remove-Item -Recurse -Force $t; Write-Host "removed: $t" } }
