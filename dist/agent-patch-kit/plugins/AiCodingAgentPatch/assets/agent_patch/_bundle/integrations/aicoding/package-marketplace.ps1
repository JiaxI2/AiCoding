param(
  [Parameter(Mandatory=$true)] [string]$AiCodingRoot,
  [switch]$Merge,
  [switch]$Zip
)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path $AiCodingRoot
$dist = Join-Path $root 'dist/agent-patch-kit'
$args = @('package','aicoding-plugin','--out',$dist)
if ($Zip) { $args += '--zip' }
apatch @args
$sidecar = Join-Path $root '.agents/plugins/agent-patch-marketplace.json'
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $sidecar) | Out-Null
Copy-Item -Force (Join-Path $dist 'marketplace.agent-patch.json') $sidecar
if ($Merge) {
  & (Join-Path $PSScriptRoot 'install-to-aicoding.ps1') -AiCodingRoot $root -Mode marketplace-merge
} else {
  Write-Host "Sidecar marketplace created: $sidecar"
}
