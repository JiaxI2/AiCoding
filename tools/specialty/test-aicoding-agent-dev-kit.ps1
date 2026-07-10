[CmdletBinding(SupportsShouldProcess)]
param([string]$RepoRoot = "", [switch]$Json)

$ErrorActionPreference = "Stop"
if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
$RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
$assetRoot = Join-Path $RepoRoot "dist\aicoding-agent-dev-kit\plugins\AiCodingAgentDevKit\assets\aicoding-agent-dev-kit"
$srcPath = Join-Path $assetRoot "src"
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("aicoding-agent-dev-kit-test-" + [System.Guid]::NewGuid().ToString("N"))
$results = New-Object System.Collections.Generic.List[object]

function Add-Result($name, $ok, $message) { $results.Add([pscustomobject]@{ name=$name; ok=[bool]$ok; message=$message }) | Out-Null }
function Invoke-Step($name, [scriptblock]$body) {
  try { & $body; Add-Result $name $true "passed" } catch { Add-Result $name $false $_.Exception.Message }
}
function Invoke-Cli([string[]]$CliArgs) {
  $savedPythonPath = $env:PYTHONPATH
  $savedNoBytecode = $env:PYTHONDONTWRITEBYTECODE
  try {
    $env:PYTHONPATH = $srcPath
    $env:PYTHONDONTWRITEBYTECODE = "1"
    python -m aicoding_agent_kit.cli @CliArgs | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "CLI failed: $($CliArgs -join ' ')" }
  } finally {
    $env:PYTHONPATH = $savedPythonPath
    $env:PYTHONDONTWRITEBYTECODE = $savedNoBytecode
  }
}

try {
  New-Item -ItemType Directory -Force -Path $temp | Out-Null
  Invoke-Step "verify-package" { & (Join-Path $RepoRoot "tools\specialty\verify-aicoding-agent-dev-kit.ps1") -RepoRoot $RepoRoot -Json | Out-Null }
  Invoke-Step "cli-install" { Invoke-Cli @("install", "--repo", $temp, "--spec-pack", "--memory", "--workflow", "--thin-skill", "--subagents") }
  Invoke-Step "cli-verify" { Invoke-Cli @("verify", "--repo", $temp) }
  Invoke-Step "cli-load" { Invoke-Cli @("load", "--repo", $temp, "--auto") }
  Invoke-Step "cli-hook-detect" { Invoke-Cli @("hook", "detect", "--repo", $temp) }
}
finally {
  if (Test-Path -LiteralPath $temp) {
    if ($PSCmdlet.ShouldProcess($temp, "Remove temporary test workspace")) {
      Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue
    }
  }
}

$ok = (@($results | Where-Object { -not $_.ok }).Count -eq 0)
$summary = [pscustomobject]@{ schema_version="1.0"; ok=[bool]$ok; name="aicoding-agent-dev-kit"; version="0.11.1"; results=$results.ToArray() }
if ($Json) { $summary | ConvertTo-Json -Depth 20 } else { $summary.results | Format-Table -AutoSize }
if (-not $ok) { exit 1 }
