[CmdletBinding()]
param([string]$RepoRoot = '.', [switch]$Json)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath $RepoRoot).Path
$sourceChecks = @(
    'config/codex-agent-powershell-skill-kit.json',
    'dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/.codex-plugin/plugin.json',
    'dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/skills/codex-agent-powershell-skill-kit/SKILL.md',
    'scripts/verify-codex-agent-powershell-skill-kit.ps1',
    'scripts/test-codex-agent-powershell-skill-kit.ps1',
    '.agents/plugins/marketplace.json'
)
$runtimeChecks = @(
    '.codex-agent-powershell-skill-kit/install-state.json',
    '.agents/skills/codex-agent-powershell-skill-kit/SKILL.md',
    '.agents/skills/codex-agent-powershell-skill-kit/RUNTIME_MIRROR_NOTICE.md',
    '.agents/skills/codex-agent-powershell-skill-kit/.runtime-mirror.json'
)
$checks = @($sourceChecks | ForEach-Object { [pscustomobject]@{ Path = $_; Exists = (Test-Path -LiteralPath (Join-Path $repo $_)); Required = $true } }) + @($runtimeChecks | ForEach-Object { [pscustomobject]@{ Path = $_; Exists = (Test-Path -LiteralPath (Join-Path $repo $_)); Required = $false } })
$result = [pscustomobject]@{
    Name = 'codex-agent-powershell-skill-kit'
    Version = '1.3.0'
    RepoRoot = $repo
    SourceOwnership = [pscustomobject]@{
        AiCodingOwnsCanonicalSource = $false
        RepoScopedSkillRole = 'generated-runtime-mirror'
    }
    Checks = $checks
    RuntimeInstalled = (-not ($checks | Where-Object { -not $_.Required -and -not $_.Exists }))
}
$result | Add-Member -NotePropertyName Ok -NotePropertyValue (-not ($result.Checks | Where-Object { $_.Required -and -not $_.Exists }))
if ($Json) { $result | ConvertTo-Json -Depth 10 } else { $result.Checks | Format-Table -AutoSize; if (-not $result.Ok) { throw 'Status check failed.' } }