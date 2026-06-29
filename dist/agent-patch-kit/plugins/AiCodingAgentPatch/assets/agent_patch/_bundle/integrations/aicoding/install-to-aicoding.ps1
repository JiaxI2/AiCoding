param(
  [Parameter(Mandatory=$true)] [string]$AiCodingRoot,
  [ValidateSet('repo-skill','marketplace-sidecar','marketplace-merge')] [string]$Mode = 'repo-skill',
  [ValidateSet('agents','codex','both')] [string]$Agent = 'agents'
)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path $AiCodingRoot

if ($Mode -eq 'repo-skill') {
  apatch deploy --scope project --project $root --agent $Agent --write-agents-snippet
  $KitRoot = Resolve-Path (Join-Path $PSScriptRoot '../..')
  Copy-Item -Force (Join-Path $KitRoot 'config/agent-patch-kit.json') (Join-Path $root 'config/agent-patch-kit.json')
  Write-Host "Installed repo-scoped Agent Patch Kit skill into $root"
  exit 0
}

$dist = Join-Path $root 'dist/agent-patch-kit'
apatch package aicoding-plugin --out $dist
$sidecar = Join-Path $root '.agents/plugins/agent-patch-marketplace.json'
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $sidecar) | Out-Null
Copy-Item -Force (Join-Path $dist 'marketplace.agent-patch.json') $sidecar
Write-Host "Created marketplace sidecar: $sidecar"

if ($Mode -eq 'marketplace-merge') {
  $market = Join-Path $root '.agents/plugins/marketplace.json'
  if (-not (Test-Path $market)) { throw "marketplace.json not found: $market" }
  $base = Get-Content $market -Raw | ConvertFrom-Json
  $entry = (Get-Content $sidecar -Raw | ConvertFrom-Json).plugins[0]
  $plugins = @($base.plugins | Where-Object { $_.name -ne $entry.name }) + $entry
  $base.plugins = $plugins
  $base | ConvertTo-Json -Depth 20 | Set-Content -Encoding UTF8 $market
  Write-Host "Merged Agent Patch Kit plugin entry into $market"
}
