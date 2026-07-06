Get-Content -LiteralPath .\README.md | ForEach-Object { $_ -replace 'AiCoding', 'AiCodingKit' } | Set-Content -LiteralPath .\README.md
