param([string]$Path = '.')
$ErrorActionPreference = 'Stop'
apatch doctor
apatch brief --format json | Out-Null
apatch state status --path $Path
apatch status --path $Path
apatch verify --path $Path
