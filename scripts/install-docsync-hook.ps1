param(
  [string]$AiCodingPath = '.',
  [switch]$Force
)

$ErrorActionPreference = 'Stop'

$repo = Resolve-Path $AiCodingPath
$hookDir = Join-Path $repo '.githooks'
$hookPath = Join-Path $hookDir 'pre-commit'

New-Item -ItemType Directory -Force -Path $hookDir | Out-Null

$docSyncBlock = @'

# AiCoding documentation synchronization gate
if command -v pwsh >/dev/null 2>&1; then
  pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode pre-commit -Staged
elif command -v powershell.exe >/dev/null 2>&1; then
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode pre-commit -Staged
else
  echo "PowerShell is required for scripts/check-documentation-sync.ps1" >&2
  exit 1
fi
'@

if (Test-Path -LiteralPath $hookPath) {
  $content = Get-Content -LiteralPath $hookPath -Raw
  if ($content -notmatch 'AiCoding documentation synchronization gate') {
    Add-Content -LiteralPath $hookPath -Value $docSyncBlock
  }
} else {
  Set-Content -LiteralPath $hookPath -Value "#!/bin/sh`nset -eu`n$docSyncBlock" -Encoding utf8
}

git -C $repo config core.hooksPath .githooks
Write-Host "[docsync] Installed docs-sync pre-commit hook in $hookPath"
