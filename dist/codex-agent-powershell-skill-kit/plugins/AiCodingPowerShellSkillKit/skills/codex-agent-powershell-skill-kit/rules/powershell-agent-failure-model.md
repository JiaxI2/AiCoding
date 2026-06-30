# PowerShell Agent Failure Model

Classify before fixing:

| Class | Signal | Correct response |
|---|---|---|
| Syntax | Parser/AST error | Fix script structure, braces, quotes, interpolation. |
| Runtime | `pwsh`/module/cmdlet missing | Resolve runtime or install missing tool with approval. |
| CWD | Git/build command cannot find repo | Run `Get-Location` and `git rev-parse --show-toplevel`. |
| Path | `Cannot find path` | Validate path and use `Join-Path`. |
| Sandbox | deny-read/deny-write/helper ACL | Request escalation or use allowed workspace path. |
| Windows ACL | Access denied | Do not rewrite syntax; classify permission issue. |
| Safety | destructive command without `-WhatIf` | Block and generate safe rewrite plan. |
| Analyzer | PSScriptAnalyzer diagnostic | Fix script quality issue. |
