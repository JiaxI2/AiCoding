# Agent Patch Kit Developer Brief

This document is intentionally developer/agent-facing. It is not user documentation.

## Why this kit exists

Agent Patch Kit turns risky free-form repository edits into a short, auditable CLI workflow. It is designed for Codex/AiCoding agents working on Windows and Git repositories, where raw PowerShell regex replacements are easy to get wrong.

## Fast-read command

An agent should start by reading the machine-oriented brief:

```powershell
apatch brief --format md
```

For compact tool-orchestration context:

```powershell
apatch brief --format json
```

## Mandatory agent workflow

```powershell
apatch state status
apatch status
apatch scan "<target>" --fixed
apatch replace --old "<old>" --new "<new>" --fixed --preview
apatch replace --old "<old>" --new "<new>" --fixed --apply
apatch verify --old "<old>" --new "<new>" --fixed
apatch summary
```

Use `apatch ast` instead of `apatch replace` for structural code changes.

## State gates

The kit is enabled only if all scopes are enabled:

```text
system && user && project
```

A missing state file means enabled. A disabled system state blocks all users and projects. A disabled user state blocks that user. A disabled project state blocks only that repository.

Agents must not apply edits when `apatch state status` reports disabled, unless the user explicitly authorizes override.

## Minimal mental model

- `apatch scan`: find first, prefer fixed-string matching.
- `apatch replace --preview`: generate diff only.
- `apatch replace --apply`: write files and create transaction.
- `apatch ast`: wrap ast-grep for syntax-aware edits.
- `apatch links`: validate Markdown links through lychee.
- `apatch tx`: list/rollback transaction snapshots.
- `apatch state`: enable/disable the kit by scope.
- `apatch summary`: give user a concise Git diff summary.

## Never do this

Do not use broad PowerShell patterns like this for repository-wide edits:

```powershell
Get-ChildItem -Recurse | ForEach-Object { (Get-Content $_) -replace ... | Set-Content $_ }
```

Use `apatch` so preview, transaction, verify, and summary remain consistent.
