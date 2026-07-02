param(
    [string]$RepoRoot = ".",
    [ValidateSet("human","agent-accepted","rejected")]
    [string]$Type = "human",
    [Parameter(Mandatory=$true)][string]$Title,
    [Parameter(Mandatory=$true)][string]$Decision,
    [string]$Context = "",
    [string]$Impact = "",
    [string]$Link = "",
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$dir = Join-Path $root ".agent-memory"
New-AgentDevKitDirectory $dir
$path = Join-Path $dir "DECISIONS.md"
if (-not (Test-Path -LiteralPath $path)) { "# Decision Memory`n" | Set-Content -Encoding UTF8 -LiteralPath $path }
$existing = Get-Content -Raw -LiteralPath $path
$ids = [regex]::Matches($existing, "D-(\d{4})") | ForEach-Object { [int]$_.Groups[1].Value }
$nextId = 1
if ($ids) { $nextId = (($ids | Measure-Object -Maximum).Maximum + 1) }
$id = "D-{0:0000}" -f $nextId
$typeLabel = switch ($Type) {
  "human" { "Human Decision" }
  "agent-accepted" { "Agent Proposal, Human Accepted" }
  "rejected" { "Rejected" }
}
$entry = @"

## ${id}: $Title

- Type: $typeLabel
- Status: Accepted
- Date: $(Get-Date -Format yyyy-MM-dd)
- Context: $Context
- Decision: $Decision
- Impact: $Impact
- Link: $Link
"@
Add-Content -Encoding UTF8 -LiteralPath $path -Value $entry
Write-AgentDevKitJson -Json:$Json -Data @{ ok=$true; id=$id; path=$path; title=$Title; type=$Type }
