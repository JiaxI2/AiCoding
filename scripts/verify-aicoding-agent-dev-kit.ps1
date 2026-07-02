param(
  [string]$RepoRoot = "",
  [switch]$Json,
  [switch]$SkipPluginValidator
)

$ErrorActionPreference = "Stop"

function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 60 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 30; foreach ($w in $warnings) { Write-Warning $w } }
  if (-not $ok) { exit 1 }
}

function Quote-Arg([string]$Value) {
  if ($null -eq $Value) { return '""' }
  if ($Value.Length -eq 0) { return '""' }
  if ($Value -notmatch '[\s"]') { return $Value }
  return '"' + ($Value -replace '"', '\"') + '"'
}

function Invoke-Capture([string]$Command, [object[]]$Arguments, [string]$WorkingDirectory = "", $ExtraEnv = @{}) {
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $Command
  $psi.Arguments = (($Arguments | ForEach-Object { Quote-Arg ([string]$_) }) -join " ")
  if ($WorkingDirectory) { $psi.WorkingDirectory = $WorkingDirectory }
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.UseShellExecute = $false
  $psi.CreateNoWindow = $true
  foreach ($key in $ExtraEnv.Keys) { $psi.EnvironmentVariables[$key] = [string]$ExtraEnv[$key] }
  $p = New-Object System.Diagnostics.Process
  $p.StartInfo = $psi
  [void]$p.Start()
  $stdout = $p.StandardOutput.ReadToEnd()
  $stderr = $p.StandardError.ReadToEnd()
  $p.WaitForExit()
  return [ordered]@{ command=$Command; arguments=$Arguments; workingDirectory=$WorkingDirectory; returncode=$p.ExitCode; stdout=$stdout; stderr=$stderr }
}

function Remove-Cache([string]$Root) {
  $removed = @()
  if (-not (Test-Path -LiteralPath $Root)) { return $removed }
  Get-ChildItem -LiteralPath $Root -Recurse -Force -Directory -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -in @("__pycache__", ".pytest_cache", ".venv") } |
    ForEach-Object { $removed += $_.FullName; Remove-Item -LiteralPath $_.FullName -Recurse -Force -ErrorAction SilentlyContinue }
  Get-ChildItem -LiteralPath $Root -Recurse -Force -File -Filter "*.pyc" -ErrorAction SilentlyContinue |
    ForEach-Object { $removed += $_.FullName; Remove-Item -LiteralPath $_.FullName -Force -ErrorAction SilentlyContinue }
  return $removed
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $pluginPath = Join-Path $RepoRoot "dist\aicoding-agent-dev-kit\plugins\AiCodingAgentDevKit"
  $assetRoot = Join-Path $pluginPath "assets\aicoding-agent-dev-kit"

  $required = @(
    (Join-Path $pluginPath ".codex-plugin\plugin.json"),
    (Join-Path $pluginPath "skills\aicoding-agent-dev-kit\SKILL.md"),
    (Join-Path $pluginPath "hooks\hooks.json"),
    (Join-Path $assetRoot "pyproject.toml"),
    (Join-Path $assetRoot "src\aicoding_agent_kit\cli.py"),
    (Join-Path $assetRoot "scripts\verify-agent-dev-kit.ps1"),
    (Join-Path $assetRoot "scripts\invoke-agent-quality-gate.ps1")
  )
  $missing = @($required | Where-Object { -not (Test-Path -LiteralPath $_) })
  $warnings = @()
  $checks = [ordered]@{
    repoRoot=$RepoRoot
    pluginPath=$pluginPath
    assetRoot=$assetRoot
    missing=$missing
    pyCompile=$null
    sourceVerify=$null
    qualityGate=$null
    cliStatus=$null
    pluginValidator=$null
    cleanup=@()
  }
  $ok = ($missing.Count -eq 0)

  if ($ok) {
    $checks.pyCompile = Invoke-Capture "python" @("-m", "py_compile", (Join-Path $assetRoot "src\aicoding_agent_kit\cli.py")) $assetRoot
    if ($checks.pyCompile.returncode -ne 0) { $ok = $false }

    $checks.sourceVerify = Invoke-Capture "pwsh" @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", (Join-Path $assetRoot "scripts\verify-agent-dev-kit.ps1"), "-RepoRoot", $assetRoot, "-Json") $assetRoot
    if ($checks.sourceVerify.returncode -ne 0) { $ok = $false }

    $checks.qualityGate = Invoke-Capture "pwsh" @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", (Join-Path $assetRoot "scripts\invoke-agent-quality-gate.ps1"), "-RepoRoot", $assetRoot, "-Mode", "all", "-Json") $assetRoot
    if ($checks.qualityGate.returncode -ne 0) { $ok = $false }

    $checks.cliStatus = Invoke-Capture "python" @("-m", "aicoding_agent_kit.cli", "status", "--repo", $assetRoot) $assetRoot @{ PYTHONPATH=(Join-Path $assetRoot "src"); PYTHONDONTWRITEBYTECODE="1" }
    if ($checks.cliStatus.returncode -ne 0) { $ok = $false }

    if (-not $SkipPluginValidator) {
      $candidate = Join-Path $env:USERPROFILE ".codex\skills\.system\plugin-creator\scripts\validate_plugin.py"
      if (Test-Path -LiteralPath $candidate) {
        $checks.pluginValidator = Invoke-Capture "python" @($candidate, $pluginPath)
        if ($checks.pluginValidator.returncode -ne 0) { $ok = $false }
      } else {
        $warnings += "Codex plugin validator not found at $candidate; skipped."
        $checks.pluginValidator = [ordered]@{ skipped=$true; reason="validator not found"; path=$candidate }
      }
    } else {
      $checks.pluginValidator = [ordered]@{ skipped=$true; reason="SkipPluginValidator flag set" }
    }
  }

  $checks.cleanup = Remove-Cache $pluginPath
  Out-Result $ok ($(if ($ok) { "OK" } else { "VERIFY_FAILED" })) "AiCoding Agent Dev Kit verification completed" $checks $warnings
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
