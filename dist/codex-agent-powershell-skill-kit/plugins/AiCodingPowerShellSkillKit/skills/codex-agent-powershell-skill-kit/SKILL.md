---
name: codex-agent-powershell-skill-kit
description: PS7-first PowerShell guard for Codex agents. Use when executing Windows commands, writing or editing .ps1/.psm1/.psd1 files, translating Bash to PowerShell, troubleshooting PowerShell failures, or integrating AiCoding scripts.
version: 1.3.0
---

# Codex Agent PowerShell Skill Kit v1.3.0

## Mission

Prevent AI-agent PowerShell failures before execution.

The agent must treat PowerShell as a typed object pipeline, not as Bash. It must validate runtime, current directory, syntax, command safety, and analyzer diagnostics before executing or committing PowerShell scripts.

## Runtime policy

Default runtime is PowerShell 7+:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File <script.ps1>
```

Fallback to Windows PowerShell 5.1 only when `pwsh` is unavailable or the task explicitly requires a Windows PowerShell-only module.

## Mandatory gates

Before executing a generated PowerShell command or accepting a generated script, run the matching gate.

### Script file gate

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-PowerShellSkillKitGate.ps1 -Path .\scripts -Recurse
```

### One-line command gate

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-SafeRewritePlan.ps1 -Command '<candidate command>' -Format Markdown
```

If the rewrite plan contains `Block = true`, do not execute the original command.

## Failure model

Classify failures before rewriting commands:

1. syntax error;
2. missing cmdlet/runtime;
3. wrong current directory;
4. path not found;
5. sandbox deny-read/deny-write;
6. Windows ACL denial;
7. destructive operation blocked;
8. PSScriptAnalyzer quality failure.

Do not treat sandbox/ACL/path failures as syntax failures.

## Rules

### Never assume current directory

Before `git`, `rg`, `python`, `node`, `npm`, `gh`, or build commands:

```powershell
Get-Location
git rev-parse --show-toplevel
```

### Avoid complex one-line commands

Do not combine unrelated operations with excessive `;`, `&&`, pipes, redirection, or variable expansion. Prefer short steps or a reviewed `.ps1` file.

### Safe interpolation

When a variable is followed by `:`, `.`, `-`, letters, or numbers inside a string, use `${var}` or `-f` formatting.

```powershell
"${count}: files found"
"{0}: files found" -f $count
```

### No Bash leakage

Replace common Bash patterns:

| Never | Prefer |
|---|---|
| `ls -la` | `Get-ChildItem -Force` |
| `cat file` | `Get-Content -Path file` |
| `grep pattern` | `Select-String -Pattern pattern` |
| `rm -rf` | `Remove-Item -Recurse -Force` with safety review |
| `cp` | `Copy-Item` |
| `mv` | `Move-Item` |
| `curl -X` | `Invoke-RestMethod` or explicit native `curl.exe` when required |

### Destructive operations require safe mode

Any command that deletes files, changes registry/system policy, modifies ACLs, formats disks, stops services/processes, or changes network configuration must have one of:

- `-WhatIf`;
- `ShouldProcess`;
- explicit user approval;
- a dedicated backup/rollback plan.


### Regex optimization subskill

Use `pwsh-regex-optimize-kit` whenever generating or reviewing PowerShell regex replacement code.

- Never put regex capture replacement tokens in double quotes. Use `'$1'` and `'${Name}'`.
- Never perform code-file regex replacement through `Get-Content | ForEach-Object { $_ -replace ... }`.
- Use `Get-Content -Raw` or `[System.IO.File]::ReadAllText()` for bulk replacement.
- Dynamic callback replacement requires PowerShell 7+ and should use `#requires -Version 7.0`.

Fast path:

```powershell
bin\aicoding.exe powershell regex-lint --staged --json
```

Slow path:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File .\.agents\skills\codex-agent-powershell-skill-kit\tools\Invoke-PowerShellRegexOptimizationGate.ps1 -Path .\scripts -Recurse -Format Json
```

## Required tools

- `tools/Invoke-PowerShellAstGate.ps1`
- `tools/Invoke-SafeRewritePlan.ps1`
- `tools/Invoke-PSScriptAnalyzerGate.ps1`
- `tools/Invoke-PowerShellSkillKitGate.ps1`
- `tools/Test-PowerShellRuntime.ps1`
- `tools/Test-PowerShellCommandSafety.ps1`
- `tools/PowerShellRegexOptimizeKit.psm1`
- `tools/Invoke-PowerShellRegexOptimizationGate.ps1`

## Agent response after failure

When a gate fails, report:

1. which gate failed;
2. the exact file or command;
3. diagnostic summary;
4. safe rewrite plan;
5. whether execution is blocked.
