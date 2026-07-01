param(
  [string]$RepoRoot = '.',
  [Parameter(Mandatory = $true)]
  [string[]]$Files
)

$repo = Resolve-Path -LiteralPath $RepoRoot
$results = New-Object System.Collections.Generic.List[object]
foreach ($rel in @($Files | Where-Object { $_ -match '\.ps1$' })) {
  $path = Join-Path $repo.Path $rel
  if (-not (Test-Path -LiteralPath $path)) { continue }
  $text = Get-Content -Raw -LiteralPath $path
  $tokens = $null
  $parseErrors = $null
  $ast = [System.Management.Automation.Language.Parser]::ParseInput($text, [ref]$tokens, [ref]$parseErrors)

  $paramNames = New-Object System.Collections.Generic.List[string]
  $validateSets = New-Object System.Collections.Generic.List[object]
  if ($ast.ParamBlock) {
    foreach ($param in @($ast.ParamBlock.Parameters)) {
      $name = $param.Name.VariablePath.UserPath
      if ($name -and -not $paramNames.Contains($name)) { $paramNames.Add($name) | Out-Null }
      foreach ($attr in @($param.Attributes)) {
        if ($attr.TypeName.FullName -eq 'ValidateSet') {
          $values = @($attr.PositionalArguments | ForEach-Object { $_.SafeGetValue() })
          $validateSets.Add([pscustomobject]@{ parameter = $name; values = $values }) | Out-Null
        }
      }
    }
  }

  $functions = New-Object System.Collections.Generic.List[string]
  $functionAsts = $ast.FindAll({ param($node) $node -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true)
  foreach ($fn in @($functionAsts)) {
    if ($fn.Name -and -not $functions.Contains($fn.Name)) { $functions.Add($fn.Name) | Out-Null }
  }

  $results.Add([pscustomobject]@{
    kind = 'powershell-script-surface'
    path = ($rel -replace '\\','/')
    parameters = @($paramNames.ToArray())
    validateSets = @($validateSets.ToArray())
    functions = @($functions.ToArray())
    parseErrors = @($parseErrors)
  }) | Out-Null
}
@($results.ToArray())