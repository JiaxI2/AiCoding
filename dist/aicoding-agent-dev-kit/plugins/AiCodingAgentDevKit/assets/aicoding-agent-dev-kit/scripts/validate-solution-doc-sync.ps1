param([string]$RepoRoot='.',[switch]$Json)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root=Resolve-AgentDevKitRepoRoot $RepoRoot
$errors=@(); $selected=Join-Path $root 'spec/SELECTED_SOLUTION.md'
if(Test-Path -LiteralPath $selected){ $text=Get-Content -Raw -LiteralPath $selected; if($text -match 'Status:\s*Accepted'){ foreach($r in @('spec/PRD.md','spec/APP_FLOW.md','spec/IMPLEMENTATION_PLAN.md','docs/traceability/TRACEABILITY_MATRIX.md')){ if(-not(Test-Path -LiteralPath (Join-Path $root $r))){$errors+="$r missing after selected solution"} }; $adrCount=@(Get-ChildItem -LiteralPath (Join-Path $root 'docs/adr') -Filter '*.md' -ErrorAction SilentlyContinue).Count; if($adrCount -eq 0){$errors+='docs/adr/*.md missing after accepted architecture option'} } }
$global:LASTEXITCODE = 0
$insideGit = $false
try {
  git -C $root rev-parse --is-inside-work-tree 2>$null | Out-Null
  $insideGit = ($LASTEXITCODE -eq 0)
} catch { $insideGit = $false }
if($insideGit){
  $global:LASTEXITCODE = 0
  $changed=git -C $root diff --name-only HEAD 2>$null
  if($LASTEXITCODE -eq 0){
    $changedText=($changed -join "`n")
    if($changedText -match 'spec/SELECTED_SOLUTION.md'){ $related=@('spec/PRD.md','spec/APP_FLOW.md','spec/IMPLEMENTATION_PLAN.md','docs/traceability/TRACEABILITY_MATRIX.md'); $has=$false; foreach($r in $related){ if($changedText -match [regex]::Escape($r)){$has=$true} }; if(-not $has){$errors+='SELECTED_SOLUTION changed but PRD/APP_FLOW/IMPLEMENTATION_PLAN/Traceability were not changed.'} }
  }
}
$global:LASTEXITCODE = 0
Write-AgentDevKitJson -Json:$Json -Data @{ok=($errors.Count -eq 0);validator='validate-solution-doc-sync';errors=$errors;gitAware=$insideGit}
if($errors.Count -gt 0){exit 1}