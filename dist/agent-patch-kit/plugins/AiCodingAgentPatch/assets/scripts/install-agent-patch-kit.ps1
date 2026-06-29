param(
  [switch]$InstallMissing,
  [ValidateSet('none','system','user','project')] [string]$DeployScope = 'none',
  [ValidateSet('agents','codex','both')] [string]$Agent = 'both',
  [string]$ProjectRoot = '',
  [switch]$WriteAgentsSnippet
)

$ErrorActionPreference = 'Stop'
$KitRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)

function Test-Cmd($Name) { return [bool](Get-Command $Name -ErrorAction SilentlyContinue) }
function Install-Winget($Id) {
  if (-not (Test-Cmd winget)) { throw "winget not found; install $Id manually" }
  winget install --id $Id --exact --accept-package-agreements --accept-source-agreements
}

$required = @('git','python','rg')
$optional = @('task','lychee','ast-grep')

foreach ($cmd in $required) {
  if (-not (Test-Cmd $cmd)) {
    if (-not $InstallMissing) { throw "Missing required command: $cmd. Re-run with -InstallMissing or install it manually." }
    switch ($cmd) {
      'git' { Install-Winget 'Git.Git' }
      'python' { Install-Winget 'Python.Python.3.12' }
      'rg' { Install-Winget 'BurntSushi.ripgrep.MSVC' }
    }
  }
}

if ($InstallMissing) {
  if (-not (Test-Cmd task)) { try { Install-Winget 'Go-task.Task' } catch { Write-Warning $_ } }
  if (-not (Test-Cmd lychee)) { try { Install-Winget 'lycheeverse.lychee' } catch { Write-Warning $_ } }
  if (-not (Test-Cmd ast-grep)) {
    if (Test-Cmd npm) { npm install --global '@ast-grep/cli' }
    else { Write-Warning "ast-grep missing and npm not found. Install with: npm install --global @ast-grep/cli" }
  }
}

Push-Location $KitRoot
try {
  python -m pip install -e .
  apatch doctor
  if ($DeployScope -eq 'system') {
    apatch deploy --scope system --agent $Agent
  } elseif ($DeployScope -eq 'user') {
    apatch deploy --scope user --agent $Agent
  } elseif ($DeployScope -eq 'project') {
    if (-not $ProjectRoot) { throw '-ProjectRoot is required for -DeployScope project' }
    $args = @('deploy','--scope','project','--agent',$Agent,'--project',$ProjectRoot)
    if ($WriteAgentsSnippet) { $args += '--write-agents-snippet' }
    apatch @args
  }
}
finally { Pop-Location }
