param(
  [string]$RepoRoot = '.',
  [Parameter(Mandatory = $true)]
  [string[]]$Files
)

$repo = Resolve-Path -LiteralPath $RepoRoot
$results = New-Object System.Collections.Generic.List[object]
foreach ($rel in @($Files | Where-Object { $_ -match '\.json$' })) {
  $path = Join-Path $repo.Path $rel
  if (-not (Test-Path -LiteralPath $path)) { continue }
  try {
    $json = Get-Content -Raw -LiteralPath $path | ConvertFrom-Json
    $rules = @()
    if ($json.PSObject.Properties.Name -contains 'rules') {
      $rules = @($json.rules | ForEach-Object { $_.name })
    }
    $thresholds = @()
    if ($json.PSObject.Properties.Name -contains 'thresholds') {
      $thresholds = @($json.thresholds.PSObject.Properties.Name)
    }
    $results.Add([pscustomobject]@{
      kind = 'json-policy-surface'
      path = ($rel -replace '\\','/')
      valid = $true
      version = $json.version
      rules = $rules
      thresholds = $thresholds
    }) | Out-Null
  } catch {
    $results.Add([pscustomobject]@{
      kind = 'json-policy-surface'
      path = ($rel -replace '\\','/')
      valid = $false
      error = $_.Exception.Message
    }) | Out-Null
  }
}
@($results.ToArray())
