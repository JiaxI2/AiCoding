[CmdletBinding(SupportsShouldProcess)]
param(
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$UninstallPip
)

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 20 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 10 }
  if (-not $ok) { exit 1 }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $marketplacePath = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  $stateDir = Join-Path $RepoRoot ".aicoding-agent-dev-kit"
  $changed = @()

  if (Test-Path -LiteralPath $marketplacePath) {
    $marketplace = Get-Content -LiteralPath $marketplacePath -Raw | ConvertFrom-Json
    $plugins = @()
    foreach ($p in $marketplace.plugins) { if ($p.name -ne "aicoding-agent-dev-kit") { $plugins += $p } }
    $marketplace | Add-Member -NotePropertyName plugins -NotePropertyValue $plugins -Force
    $marketplace | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $marketplacePath -Encoding UTF8
    $changed += ".agents/plugins/marketplace.json"
  }

  $pipUninstalled = $false
  if ($UninstallPip) {
    python -m pip uninstall -y aicoding-agent-dev-kit | Out-Host
    $pipUninstalled = $true
  }
  if (Test-Path -LiteralPath $stateDir) {
    if ($PSCmdlet.ShouldProcess($stateDir, "Remove local runtime state directory")) {
      Remove-Item -LiteralPath $stateDir -Recurse -Force
      $changed += ".aicoding-agent-dev-kit"
    }
  }

  Out-Result $true "OK" "AiCoding Agent Dev Kit uninstalled from local runtime state" @{ changed=$changed; pipUninstalled=$pipUninstalled }
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
