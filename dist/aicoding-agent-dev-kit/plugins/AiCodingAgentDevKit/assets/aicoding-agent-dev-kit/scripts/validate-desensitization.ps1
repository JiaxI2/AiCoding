param([string]$RepoRoot = ".", [switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$forbidden = @("PRIVATE_PROJECT_NAME","PRIVATE_COMPANY_NAME","token=","api_key","password=")
$ruleFiles = @(
  (Join-Path $root "config\quality-gate.json"),
  (Join-Path $root "scripts\validate-desensitization.ps1")
)
$hits = @()
$files = Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue |
  Where-Object {
    $_.FullName -notmatch "\\.git\\" -and
    $_.Length -lt 1048576 -and
    ($ruleFiles -notcontains $_.FullName)
  }
foreach ($file in $files) {
  try {
    $text = Get-Content -Raw -LiteralPath $file.FullName -ErrorAction Stop
    foreach ($term in $forbidden) {
      if ($text -match [regex]::Escape($term)) {
        $hits += "$($file.FullName):$term"
      }
    }
  } catch {}
}
Write-AgentDevKitJson -Json:$Json -Data @{ ok = ($hits.Count -eq 0); validator = "validate-desensitization"; hits = $hits }
if ($hits.Count -gt 0) { exit 1 }