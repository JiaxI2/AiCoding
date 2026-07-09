[CmdletBinding()]
param(
  [string]$RepoRoot = "",
  [switch]$Json
)

$ErrorActionPreference = "Stop"

function Out-Result([bool]$Ok, [string]$Code, [string]$Message, $Data = @{}) {
  $obj = [ordered]@{
    schema_version = "1.0"
    ok = $Ok
    code = $Code
    message = $Message
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

function Test-PowerShellSyntax([string]$Path) {
  [System.Management.Automation.Language.Token[]]$tokens = $null
  [System.Management.Automation.Language.ParseError[]]$parseErrors = $null
  [System.Management.Automation.Language.Parser]::ParseFile($Path, [ref]$tokens, [ref]$parseErrors) | Out-Null
  return @($parseErrors).Count -eq 0
}

try {
  if (-not $RepoRoot) {
    $RepoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
  }
  $RepoRoot = (Resolve-Path -LiteralPath $RepoRoot).Path
  $checks = New-Object System.Collections.Generic.List[object]
  $errors = New-Object System.Collections.Generic.List[string]

  function Add-Check([string]$Name, [bool]$Ok, [string]$Message, $Data = $null) {
    $checks.Add([ordered]@{ name = $Name; ok = $Ok; message = $Message; data = $Data }) | Out-Null
    if (-not $Ok) {
      $errors.Add(("{0}: {1}" -f $Name, $Message)) | Out-Null
    }
  }

  $requiredFiles = @(
    "cmd/aicoding/main.go",
    "scripts/invoke-aicoding-agent-hook.ps1",
    "scripts/verify-agent-dev-kit-plan-mode.ps1",
    "scripts/hooks/aef/plan-mode-gate.ps1",
    "scripts/hooks/aef/spec-artifact-gate.ps1",
    "config/hooks-registry.json",
    "config/agent-dev-kit-plan-mode.registry.json"
  )

  foreach ($rel in $requiredFiles) {
    $path = Join-Path $RepoRoot ($rel -replace '/', [IO.Path]::DirectorySeparatorChar)
    $exists = Test-Path -LiteralPath $path -PathType Leaf
    Add-Check "required:$rel" $exists ("检查必要文件：{0}" -f $rel)
    if ($exists -and $rel.EndsWith(".ps1")) {
      Add-Check "syntax:$rel" (Test-PowerShellSyntax $path) ("PowerShell 语法检查：{0}" -f $rel)
    }
  }

  $hookRegistryPath = Join-Path $RepoRoot "config/hooks-registry.json"
  if (Test-Path -LiteralPath $hookRegistryPath -PathType Leaf) {
    try {
      $hookRegistry = Get-Content -LiteralPath $hookRegistryPath -Raw -Encoding UTF8 | ConvertFrom-Json
      Add-Check "hooks.parse" $true "hook 注册表解析通过。"
      $hookIds = @($hookRegistry.hooks | ForEach-Object { $_.id })
      Add-Check "hooks.plan-mode-gate" ($hookIds -contains "plan-mode-gate") "plan-mode-gate 已注册。"
      Add-Check "hooks.spec-artifact-gate" ($hookIds -contains "spec-artifact-gate") "spec-artifact-gate 已注册。"
    } catch {
      Add-Check "hooks.parse" $false ("hook 注册表解析失败：{0}" -f $_.Exception.Message)
    }
  }

  $ok = ($errors.Count -eq 0)
  $code = if ($ok) { "OK" } else { "AEF_VERIFY_FAILED" }
  $message = if ($ok) { "Agent Engineering Foundation 专项验证通过。" } else { "Agent Engineering Foundation 专项验证未通过。" }
  Out-Result $ok $code $message @{
    repoRoot = $RepoRoot
    checks = @($checks.ToArray())
    errors = @($errors.ToArray())
  }
}
catch {
  Out-Result $false "INTERNAL_ERROR" ("Agent Engineering Foundation 验证脚本内部错误：{0}" -f $_.Exception.Message) ([ordered]@{ scriptStackTrace = $_.ScriptStackTrace; category = $_.CategoryInfo.ToString() })
}
