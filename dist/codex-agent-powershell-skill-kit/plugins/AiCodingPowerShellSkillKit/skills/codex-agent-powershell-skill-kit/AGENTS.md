# Agent Operating Rules for PowerShell

## PS7-first

Always prefer `pwsh`. Do not emit `powershell.exe` commands unless fallback is required and documented.

## Pre-execution checklist

For every PowerShell script or command:

1. Resolve runtime.
2. Confirm current directory and repo root when repo context matters.
3. Run AST gate.
4. Run safety gate.
5. Run PSScriptAnalyzer gate for script files.
6. Generate safe rewrite plan for risky one-liners.
7. Execute only when all blocking gates pass.

## Recent failure prevention

- Do not use Linux aliases in PowerShell.
- Do not assume `python.exe`, `node.exe`, `git.exe`, or `rg.exe` exists. Use `Get-Command`.
- Do not edit user-level config files without backup.
- Do not run complex command chains when separate steps would be safer.
- Do not retry by randomly changing quoting. First classify the failure.
