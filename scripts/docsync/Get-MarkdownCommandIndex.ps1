param(
  [string]$RepoRoot = '.',
  [string[]]$Files = @()
)

$repo = Resolve-Path -LiteralPath $RepoRoot
if ($Files -and $Files.Count -gt 0) {
  $docs = @($Files |
    Where-Object { $_ -match '\.md$' } |
    ForEach-Object { Get-Item -LiteralPath (Join-Path $repo.Path $_) -ErrorAction SilentlyContinue } |
    Where-Object { $_ })
} else {
  $docs = @()
  foreach ($pattern in @('README*.md','CHANGELOG.md','AGENTS.md')) {
    $docs += @(Get-ChildItem -LiteralPath $repo.Path -File -Filter $pattern -ErrorAction SilentlyContinue)
  }
  foreach ($dir in @('docs','config')) {
    $full = Join-Path $repo.Path $dir
    if (Test-Path -LiteralPath $full) { $docs += @(Get-ChildItem -LiteralPath $full -Recurse -File -Include '*.md' -ErrorAction SilentlyContinue) }
  }
  foreach ($rel in @('CodingKit/README.md','CodingKit/AGENTS.md')) {
    $full = Join-Path $repo.Path $rel
    if (Test-Path -LiteralPath $full) { $docs += @(Get-Item -LiteralPath $full) }
  }
}

$items = New-Object System.Collections.Generic.List[object]
foreach ($doc in @($docs | Select-Object -Unique)) {
  $rel = $doc.FullName.Substring($repo.Path.Length).TrimStart('\','/') -replace '\\','/'
  if ($rel -match '^(\.git|\.agents|\.codex|dist|plugins|CodingKit/tests|CodingKit/agents/skills)/') { continue }
  $lines = Get-Content -LiteralPath $doc.FullName
  for ($i = 0; $i -lt $lines.Count; $i++) {
    $line = $lines[$i]
    $matches = @([regex]::Matches($line, '(?<![A-Za-z0-9_\.\-/\\])(?<script>(?:\.?[\\/])?scripts[\\/][A-Za-z0-9_\.\-\\/]+?\.ps1)'))
    for ($matchIndex = 0; $matchIndex -lt $matches.Count; $matchIndex++) {
      $m = $matches[$matchIndex]
      $script = ($m.Groups['script'].Value -replace '^\.[\\/]', '') -replace '\\','/'
      $argStart = $m.Index + $m.Length
      $argEnd = if ($matchIndex + 1 -lt $matches.Count) { $matches[$matchIndex + 1].Index } else { $line.Length }
      $argText = $line.Substring($argStart, [Math]::Max(0, $argEnd - $argStart))
      $args = @()
      foreach ($arg in [regex]::Matches($argText, '\s-(?<name>[A-Za-z][A-Za-z0-9_]*)')) { $args += $arg.Groups['name'].Value }
      $items.Add([pscustomobject]@{
        doc = $rel
        line = ($i + 1)
        script = $script
        args = @($args)
        raw = $line.Trim()
      }) | Out-Null
    }
  }
}
@($items.ToArray())