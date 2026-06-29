param(
  [Parameter(Mandatory=$true)] [string]$ProjectRoot,
  [ValidateSet('agents','codex','both')] [string]$Agent = 'both',
  [switch]$WriteAgentsSnippet
)
$ErrorActionPreference = 'Stop'
$args = @('deploy','--scope','project','--agent',$Agent,'--project',$ProjectRoot)
if ($WriteAgentsSnippet) { $args += '--write-agents-snippet' }
apatch @args
