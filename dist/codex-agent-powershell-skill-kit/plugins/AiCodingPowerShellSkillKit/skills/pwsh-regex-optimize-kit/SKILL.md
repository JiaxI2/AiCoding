---
name: pwsh-regex-optimize-kit
description: PowerShell 7+ regex optimization subskill for safe capture replacement, bulk file replacement, dynamic callback transform, and Go fast-path lint integration.
version: 1.3.0
---

# PowerShell Regex Optimization Skill Kit v1.3.0

## Mission

Prevent AI-agent generated PowerShell regex edits from causing silent capture-group corruption or slow line-by-line replacement.

## Runtime

Use PowerShell 7+ by default.

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass
```

Dynamic callback replacement requires PowerShell 7+.

## Required rules

### PSRegex001.DoubleQuotedCaptureReplacement

Never write capture-group replacement tokens inside double quotes:

```powershell
# bad
$content -creplace '(\w+)', "$1"

# good
$content -creplace '(\w+)', '$1'
```

Reason: PowerShell expands `$1` before the regex engine receives the replacement string.

### PSRegex002.LinePipelineReplace

Never use line pipeline replacement for code files larger than trivial samples:

```powershell
# bad
Get-Content file.ps1 | ForEach-Object { $_ -replace 'old', 'new' } | Set-Content file.ps1

# good
$content = Get-Content -LiteralPath file.ps1 -Raw
$content = $content -creplace 'old', 'new'
Set-Content -LiteralPath file.ps1 -Value $content -NoNewline
```

### PSRegex003.DynamicCallbackRequiresPS7

When using scriptblock replacement, mark the script as PowerShell 7+:

```powershell
#requires -Version 7.0
$result = $source -creplace '(?:^|_)(\w)', { $_.Groups[1].Value.ToUpperInvariant() }
```

## Exported tools

- `tools/PowerShellRegexOptimizeKit.psm1`
- `tools/Invoke-PowerShellRegexOptimizationGate.ps1`
- Go fast-path package: `internal/pwshregex`

## Validation

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-PowerShellRegexOptimizationGate.ps1 -Path .\scripts -Recurse -Format Json
bin\aicoding.exe powershell regex-lint --staged --json
bin\aicoding.exe powershell regex-lint --path scripts --json
```
