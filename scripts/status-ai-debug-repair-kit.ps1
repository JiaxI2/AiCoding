param([string]$RepoRoot = "", [switch]$Json)
function Out-Result($ok, $code, $message, $data = @{}, $warnings = @()) {
  $obj = [ordered]@{ schema_version="1.0"; ok=[bool]$ok; code=$code; message=$message; data=$data; warnings=$warnings }
  if ($Json) { $obj | ConvertTo-Json -Depth 20 } else { Write-Host "[$code] $message"; $data | ConvertTo-Json -Depth 10 }
  if (-not $ok) { exit 1 }
}

function Resolve-AirepairCommand {
  $checked = @()

  $cmd = Get-Command airepair -ErrorAction SilentlyContinue
  if ($cmd) {
    return [ordered]@{ path=$cmd.Source; source="PATH"; checked=@($cmd.Source) }
  }

  $pythonRoots = @()
  if ($env:APPDATA) { $pythonRoots += (Join-Path $env:APPDATA "Python") }
  if ($env:LOCALAPPDATA) { $pythonRoots += (Join-Path $env:LOCALAPPDATA "Programs\Python") }
  foreach ($root in ($pythonRoots | Select-Object -Unique)) {
    if (-not (Test-Path -LiteralPath $root)) { continue }
    $matches = Get-ChildItem -LiteralPath $root -Recurse -Filter "airepair.exe" -ErrorAction SilentlyContinue | Select-Object -ExpandProperty FullName
    foreach ($candidate in $matches) {
      if (-not $candidate) { continue }
      $checked += [string]$candidate
      if (Test-Path -LiteralPath $candidate) {
        return [ordered]@{ path=[string]$candidate; source="python-user-scripts"; checked=($checked | Select-Object -Unique) }
      }
    }
  }

  foreach ($pythonCommand in @(@("python"), @("py", "-3"))) {
    try {
      $args = @($pythonCommand + @("-c", "import sysconfig; print(sysconfig.get_path('scripts', scheme='nt_user') or '')"))
      $exe = $args[0]
      $exeArgs = @($args | Select-Object -Skip 1)
      $scriptPath = (& $exe @exeArgs 2>$null | Select-Object -First 1)
      if ($LASTEXITCODE -eq 0 -and $scriptPath) {
        $candidate = Join-Path ([string]$scriptPath) "airepair.exe"
        $checked += $candidate
        if (Test-Path -LiteralPath $candidate) {
          return [ordered]@{ path=$candidate; source="python-user-scripts"; checked=($checked | Select-Object -Unique) }
        }
      }
    } catch {}
  }

  return [ordered]@{ path=$null; source=$null; checked=($checked | Select-Object -Unique) }
}

try {
  if (-not $RepoRoot) { $RepoRoot = (Get-Location).Path }
  $RepoRoot = (Resolve-Path $RepoRoot).Path
  $pluginPath = Join-Path $RepoRoot "dist\ai-debug-repair-kit\plugins\AiCodingAIDebugRepairKit"
  $manifest = Join-Path $pluginPath ".codex-plugin\plugin.json"
  $marketplace = Join-Path $RepoRoot ".agents\plugins\marketplace.json"
  $state = Join-Path $RepoRoot ".ai-debug-repair\install-state.json"
  $airepairInfo = Resolve-AirepairCommand
  $data = [ordered]@{
    repoRoot=$RepoRoot
    pluginExists=(Test-Path $pluginPath)
    manifestExists=(Test-Path $manifest)
    marketplaceExists=(Test-Path $marketplace)
    stateExists=(Test-Path $state)
    airepair=$airepairInfo.path
    airepairSource=$airepairInfo.source
    airepairChecked=$airepairInfo.checked
  }
  $ok = $data.pluginExists -and $data.manifestExists -and $data.marketplaceExists
  Out-Result $ok ($(if ($ok) { "OK" } else { "PARTIAL" })) "AI Debug Repair Kit status" $data
}
catch { Out-Result $false "INTERNAL_ERROR" $_.Exception.Message }
