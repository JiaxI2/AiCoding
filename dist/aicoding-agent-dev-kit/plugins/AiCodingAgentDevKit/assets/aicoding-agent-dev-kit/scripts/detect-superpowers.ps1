param([switch]$Json)
$paths = @("$env:USERPROFILE\.agents\skills", "$env:USERPROFILE\.codex\skills")
$found = $false
foreach ($p in $paths) {
    if ($p -and (Test-Path -LiteralPath $p)) {
        if (Get-ChildItem -LiteralPath $p -Recurse -Filter "*superpower*" -ErrorAction SilentlyContinue | Select-Object -First 1) { $found = $true }
    }
}
$data = @{ ok = $true; superpowersDetected = $found; required = $false }
if ($Json) { $data | ConvertTo-Json -Depth 4 } else { $data }
