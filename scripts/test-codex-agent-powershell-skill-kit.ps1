[CmdletBinding()]
param(
    [string]$RepoRoot = '.',
    [switch]$InstallMissingTools,
    [switch]$Json
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath $RepoRoot).Path
$skill = Join-Path $repo '.agents\skills\codex-agent-powershell-skill-kit'
$toolRoot = Join-Path $skill 'tools'
$caseRoot = Join-Path $skill 'tests\cases'
$results = New-Object System.Collections.Generic.List[object]

function Add-Result { param([string]$Name, [bool]$Ok, [string]$Message) $results.Add([pscustomobject]@{ Name=$Name; Ok=$Ok; Message=$Message }) | Out-Null }
function Expect-Pass { param([string]$Name, [scriptblock]$Action) try { & $Action; Add-Result $Name $true 'passed as expected' } catch { Add-Result $Name $false ("expected pass but failed: " + $_.Exception.Message) } }
function Expect-Fail { param([string]$Name, [scriptblock]$Action) try { & $Action; Add-Result $Name $false 'expected failure but passed' } catch { Add-Result $Name $true 'failed as expected' } }

Expect-Pass 'runtime-ps7' { & (Join-Path $toolRoot 'Test-PowerShellRuntime.ps1') | Out-Null }
Expect-Pass 'good-cases-full-gate' { & (Join-Path $toolRoot 'Invoke-PowerShellSkillKitGate.ps1') -Path (Join-Path $caseRoot 'good') -Recurse -InstallMissingTools:$InstallMissingTools | Out-Null }
Expect-Fail 'bad-syntax-ast-fails' { & (Join-Path $toolRoot 'Invoke-PowerShellAstGate.ps1') -Path (Join-Path $caseRoot 'bad\Syntax-MissingBrace.ps1') | Out-Null }
Expect-Fail 'bad-linux-alias-safety-fails' { & (Join-Path $toolRoot 'Test-PowerShellCommandSafety.ps1') -Path (Join-Path $caseRoot 'bad\Linux-Aliases.ps1') | Out-Null }
Expect-Fail 'bad-removeitem-safety-fails' { & (Join-Path $toolRoot 'Test-PowerShellCommandSafety.ps1') -Path (Join-Path $caseRoot 'bad\Unsafe-RemoveItem.ps1') | Out-Null }
Expect-Fail 'rewrite-blocks-bash-leakage' { & (Join-Path $toolRoot 'Invoke-SafeRewritePlan.ps1') -Path (Join-Path $caseRoot 'rewrite\BashLeakage.ps1') -Format Json | Out-Null }
Expect-Fail 'rewrite-interpolation-risk-blocks' { & (Join-Path $toolRoot 'Invoke-SafeRewritePlan.ps1') -Path (Join-Path $caseRoot 'rewrite\InterpolationRisk.ps1') -Format Json | Out-Null }

$ok = (@($results | Where-Object { -not $_.Ok }).Count -eq 0)
$summary = [pscustomobject]@{ Name='codex-agent-powershell-skill-kit-tests'; Ok=[bool]$ok; Results=$results.ToArray() }
if ($Json) { $summary | ConvertTo-Json -Depth 20 } else { $summary.Results | Format-Table -AutoSize }
if (-not $ok) { throw 'Gate failed.' }
