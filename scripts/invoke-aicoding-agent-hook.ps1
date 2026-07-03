[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [string]$Trigger = "manual",
  [string]$Event = "manual",
  [ValidateSet("warn","enforce")][string]$Mode = "warn",
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-BridgeResult([bool]$Ok, [string]$Code, [string]$Message, $Data = @{}) {
  $obj = [ordered]@{
    schema_version = "1.0"
    ok = $Ok
    code = $Code
    message = $Message
    trigger = $Trigger
    event = $Event
    mode = $Mode
    data = $Data
  }
  if ($Json) {
    $obj | ConvertTo-Json -Depth 60
  } else {
    Write-Host ("[{0}] {1}" -f $Code, $Message)
    $Data | ConvertTo-Json -Depth 30
  }
  if (-not $Ok) { exit 1 }
}

try {
  if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
  }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $registryPath = Join-Path $RepoRoot "config/hooks-registry.json"
  if (-not (Test-Path -LiteralPath $registryPath -PathType Leaf)) {
    Out-BridgeResult $false "MISSING_REGISTRY" "config/hooks-registry.json not found" @{ repoRoot = $RepoRoot }
  }

  $registry = Get-Content -LiteralPath $registryPath -Raw -Encoding UTF8 | ConvertFrom-Json
  $hooks = @($registry.hooks | Where-Object {
    $_.type -eq "agent-hook" -and
    $_.enabledByDefault -eq $true -and
    $_.trigger -eq $Trigger
  })

  $results = New-Object System.Collections.Generic.List[object]
  foreach ($hook in $hooks) {
    $hookPath = Join-Path $RepoRoot (($hook.path -replace '/', [IO.Path]::DirectorySeparatorChar))
    if (-not (Test-Path -LiteralPath $hookPath -PathType Leaf)) {
      $results.Add([ordered]@{ id = $hook.id; ok = $false; code = "MISSING_HOOK"; path = $hook.path }) | Out-Null
      continue
    }

    $hookMode = $Mode
    if ($hook.PSObject.Properties.Name -contains "mode" -and $hook.mode) {
      $hookMode = [string]$hook.mode
    }
    $capture = & pwsh -NoProfile -ExecutionPolicy Bypass -File $hookPath -RepoRoot $RepoRoot -Event $Event -Mode $hookMode -Json 2>&1
    $exitCode = $LASTEXITCODE
    $parsed = $null
    try {
      $parsed = ($capture | Out-String).Trim() | ConvertFrom-Json
    } catch {
      $parsed = [ordered]@{ raw = ($capture | Out-String) }
    }
    $results.Add([ordered]@{ id = $hook.id; ok = ($exitCode -eq 0); exitCode = $exitCode; result = $parsed }) | Out-Null
  }

  $failed = @($results | Where-Object { -not $_.ok })
  Out-BridgeResult ($failed.Count -eq 0) ($(if ($failed.Count -eq 0) { "OK" } else { "HOOK_FAILED" })) "AiCoding agent hook bridge completed" @{
    repoRoot = $RepoRoot
    hooks = @($results)
    matched = $hooks.Count
  }
}
catch {
  Out-BridgeResult $false "INTERNAL_ERROR" $_.Exception.Message
}
