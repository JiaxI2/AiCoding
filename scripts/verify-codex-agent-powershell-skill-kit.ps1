[CmdletBinding()]
param(
    [string]$RepoRoot = '.',
    [switch]$InstallMissingTools,
    [switch]$FailOnWarning,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath $RepoRoot).Path
$skill = Join-Path $repo '.agents\skills\codex-agent-powershell-skill-kit'
$packagedSkill = Join-Path $repo 'dist\codex-agent-powershell-skill-kit\plugins\AiCodingPowerShellSkillKit\skills\codex-agent-powershell-skill-kit'
$tool = Join-Path $skill 'tools\Invoke-PowerShellSkillKitGate.ps1'
$statusScript = Join-Path $repo 'scripts\status-codex-agent-powershell-skill-kit.ps1'

$results = New-Object System.Collections.Generic.List[object]
function Add-Check { param([string]$Name, [bool]$Ok, [string]$Message) $results.Add([pscustomobject]@{ Name=$Name; Ok=$Ok; Message=$Message }) | Out-Null }
function Invoke-Check { param([string]$Name, [scriptblock]$Action) try { & $Action; Add-Check $Name $true 'passed' } catch { Add-Check $Name $false $_.Exception.Message } }

Invoke-Check 'status' { & $statusScript -RepoRoot $repo | Out-Null }
Invoke-Check 'source-ownership-marker' {
    if (-not (Test-Path -LiteralPath (Join-Path $skill '.runtime-mirror.json'))) { throw 'runtime mirror marker missing' }
    $meta = Get-Content -LiteralPath (Join-Path $skill '.runtime-mirror.json') -Raw | ConvertFrom-Json
    if ($meta.canonicalOwnedByAiCoding -ne $false) { throw 'runtime mirror metadata incorrectly claims AiCoding canonical ownership' }
}
Invoke-Check 'runtime' { & (Join-Path $skill 'tools\Test-PowerShellRuntime.ps1') | Out-Null }
Invoke-Check 'skill-tools-ast-safety-analyzer' { & $tool -Path @((Join-Path $skill 'tools'), (Join-Path $skill 'hooks'), (Join-Path $skill 'tests\cases\good')) -Recurse -InstallMissingTools:$InstallMissingTools -FailOnWarning:$FailOnWarning | Out-Null }
Invoke-Check 'packaged-skill-ast-safety-analyzer' { & $tool -Path @((Join-Path $packagedSkill 'tools'), (Join-Path $packagedSkill 'hooks'), (Join-Path $packagedSkill 'tests\cases\good')) -Recurse -InstallMissingTools:$InstallMissingTools -FailOnWarning:$FailOnWarning | Out-Null }
$kitScripts = @(
    (Join-Path $repo 'scripts\install-codex-agent-powershell-skill-kit.ps1'),
    (Join-Path $repo 'scripts\status-codex-agent-powershell-skill-kit.ps1'),
    (Join-Path $repo 'scripts\uninstall-codex-agent-powershell-skill-kit.ps1'),
    (Join-Path $repo 'scripts\verify-codex-agent-powershell-skill-kit.ps1'),
    (Join-Path $repo 'scripts\test-codex-agent-powershell-skill-kit.ps1')
)
Invoke-Check 'repo-kit-scripts-ast-safety-analyzer' { & $tool -Path $kitScripts -InstallMissingTools:$InstallMissingTools -FailOnWarning:$FailOnWarning | Out-Null }

$marketplace = Join-Path $repo '.agents\plugins\marketplace.json'
Invoke-Check 'marketplace-entry' {
    $data = Get-Content -LiteralPath $marketplace -Raw | ConvertFrom-Json
    $hit = @($data.plugins | Where-Object { $_.name -eq 'codex-agent-powershell-skill-kit' }).Count
    if ($hit -lt 1) { throw 'marketplace entry not found' }
}

$ok = (@($results | Where-Object { -not $_.Ok }).Count -eq 0)
$summary = [pscustomobject]@{ Name='codex-agent-powershell-skill-kit'; Version='1.2.1'; Ok=[bool]$ok; Results=$results.ToArray() }
if ($Json) { $summary | ConvertTo-Json -Depth 20 } else { $summary.Results | Format-Table -AutoSize }
if (-not $ok) { throw 'Gate failed.' }
