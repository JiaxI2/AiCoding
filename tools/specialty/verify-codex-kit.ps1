param([switch]$Json)
$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\..')).Path
$bin = Join-Path $repo 'bin\aicoding.exe'
if (Test-Path -LiteralPath $bin -PathType Leaf) {
    $argsList = @('full')
    if ($Json) { $argsList += '--json' }
    & $bin @argsList
    exit $LASTEXITCODE
}
$argsList = @('run', './cmd/aicoding', 'full')
if ($Json) { $argsList += '--json' }
& go @argsList
exit $LASTEXITCODE
