param(
    [string]$RepoRoot = ".",
    [int]$MaxFileChars = 20000,
    [switch]$Json
)
. "$PSScriptRoot\Common-AgentDevKit.ps1"
$root = Resolve-AgentDevKitRepoRoot $RepoRoot
$large = @()
$totalChars = 0
$files = Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue |
  Where-Object { $_.FullName -notmatch "\\.git\\" -and $_.FullName -notmatch "\\node_modules\\" -and $_.FullName -notmatch "\\.next\\" -and $_.Length -lt 5242880 }
foreach ($f in $files) {
  try {
    $text = Get-Content -Raw -LiteralPath $f.FullName -ErrorAction Stop
    $len = $text.Length
    $totalChars += $len
    if ($len -gt $MaxFileChars) {
      $large += [ordered]@{ path=$f.FullName.Substring($root.Length + 1).Replace("\","/"); chars=$len; roughTokens=[int]($len / 4) }
    }
  } catch {}
}
Write-AgentDevKitJson -Json:$Json -Data @{
  ok=$true
  totalChars=$totalChars
  roughTokens=[int]($totalChars / 4)
  largeFiles=$large
  advice="Read generated context-pack first; avoid loading large files unless explicitly needed."
}
