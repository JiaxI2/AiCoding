$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
Push-Location $repo
try {
  go test ./internal/asset ./cmd/assetkit
  go run ./cmd/assetkit validate ./CodingKit/kits/universal-asset-lifecycle
  $tmp = Join-Path ([IO.Path]::GetTempPath()) ('aicoding-asset-' + [guid]::NewGuid())
  New-Item -ItemType Directory -Force $tmp | Out-Null
  go run ./cmd/assetkit pack ./CodingKit/kits/universal-asset-lifecycle --out (Join-Path $tmp 'kit.aicoding.zip')
  if (-not (Test-Path (Join-Path $tmp 'kit.aicoding.zip'))) { throw 'package was not generated' }
  Write-Host 'Universal Asset Lifecycle validation passed.'
} finally { Pop-Location }
