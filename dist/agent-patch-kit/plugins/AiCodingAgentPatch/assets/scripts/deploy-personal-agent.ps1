param([ValidateSet('agents','codex','both')] [string]$Agent = 'both')
$ErrorActionPreference = 'Stop'
apatch deploy --scope user --agent $Agent
