param(
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$SkipPytest
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 40 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 20 }
  if (-not $ok) { exit 1 }
}

function Quote-ProcessArgument([string]$Value) {
  if ($null -eq $Value -or $Value.Length -eq 0) { return '""' }
  if ($Value -notmatch '[\s"]') { return $Value }
  $s = $Value -replace '(\\*)"', '$1$1\"'
  $s = $s -replace '(\\+)$', '$1$1'
  return '"' + $s + '"'
}

function Invoke-ProcessCapture {
  param([string]$Command, [object[]]$Arguments, [string]$WorkingDirectory = "", $ExtraEnv = @{})
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $Command
  $parts = @()
  foreach ($arg in $Arguments) { $parts += Quote-ProcessArgument ([string]$arg) }
  $psi.Arguments = ($parts -join " ")
  if ($WorkingDirectory) { $psi.WorkingDirectory = $WorkingDirectory }
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.UseShellExecute = $false
  $psi.CreateNoWindow = $true
  foreach ($key in $ExtraEnv.Keys) {
    if ($psi.EnvironmentVariables.ContainsKey($key)) { $psi.EnvironmentVariables[$key] = [string]$ExtraEnv[$key] }
    else { $psi.EnvironmentVariables.Add($key, [string]$ExtraEnv[$key]) }
  }
  $p = New-Object System.Diagnostics.Process
  $p.StartInfo = $psi
  [void]$p.Start()
  $stdout = $p.StandardOutput.ReadToEnd()
  $stderr = $p.StandardError.ReadToEnd()
  $p.WaitForExit()
  return [ordered]@{ command=$Command; arguments=$Arguments; argumentLine=$psi.Arguments; workingDirectory=$WorkingDirectory; returncode=$p.ExitCode; stdout=$stdout; stderr=$stderr }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $pluginPath = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  $assetRoot = Join-Path $pluginPath "assets\ai-debug-repair-kit"
  $testsPath = Join-Path $assetRoot "tests"
  $srcPath = Join-Path $assetRoot "src"

  $data = [ordered]@{
    repoRoot = $RepoRoot
    pluginPath = $pluginPath
    assetRoot = $assetRoot
    testsPath = $testsPath
    pytest = $null
  }

  if (-not (Test-Path -LiteralPath $assetRoot)) { Out-Result $false "ASSET_ROOT_MISSING" "AI Debug Repair Kit asset root not found" $data }
  if (-not (Test-Path -LiteralPath $testsPath)) { Out-Result $false "TESTS_MISSING" "AI Debug Repair Kit tests directory not found" $data }

  if ($SkipPytest) {
    $data.pytest = [ordered]@{ skipped=$true; reason="SkipPytest flag set" }
    Out-Result $true "OK" "AI Debug Repair Kit tests skipped" $data
  }

  $extraEnv = @{ PYTHONPATH=$srcPath; PYTHONDONTWRITEBYTECODE="1"; PYTEST_DISABLE_PLUGIN_AUTOLOAD="1" }
  $pytestRun = Invoke-ProcessCapture "python" @("-m", "pytest", "-q", "-p", "no:cacheprovider", $testsPath) $assetRoot $extraEnv
  $data.pytest = $pytestRun
  $ok = ($pytestRun.returncode -eq 0)
  Out-Result $ok ($(if ($ok) { "OK" } else { "TEST_FAILED" })) "AI Debug Repair Kit tests completed" $data
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }