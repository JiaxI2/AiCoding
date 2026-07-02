param(
    [string]$RepoRoot = ".",
    [ValidateSet("changed","staged","spec","memory","all")]
    [string]$Mode = "changed",
    [int]$MaxChars = 12000,
    [switch]$Json
)
# Compatibility wrapper for v0.11.1.
# Prefer load-agent-context.ps1 for staged sequential loading.
$stage = "L1"
if ($Mode -eq "memory") { $stage = "L0" }
if ($Mode -eq "spec" -or $Mode -eq "all") { $stage = "L3" }
& "$PSScriptRoot\load-agent-context.ps1" -RepoRoot $RepoRoot -Stage $stage -MaxChars $MaxChars -Json:$Json
