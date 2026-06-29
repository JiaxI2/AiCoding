# Agent Patch Kit

Agent Patch Kit is an agent-safe patch workflow CLI and Codex/AiCoding skill package.

It standardizes:

```text
git status -> rg/apatch scan -> preview -> apply + transaction -> verify -> diff summary
```

## New in v0.2.1

- `apatch brief`: developer/agent-only fast-read entry point.
- `apatch state`: system, user, and project level enable/disable control.
- `apatch deploy --scope system|user|project`: CLI deployment for system-managed, personal, or project agents.

## Install on Windows

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1
```

Install missing dependencies when possible:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -InstallMissing
```

Required tools: `git`, `python >= 3.10`, `rg`.

Optional tools: `task`, `lychee`, `ast-grep`, `sd`.

## Agent fast-read

This is for developers and agents, not normal end-user docs:

```powershell
apatch brief --format md
apatch brief --format json
```

Agents should check state before applying edits:

```powershell
apatch state status
```

## Enable / disable by scope

```powershell
apatch state status
apatch state where
```

User scope:

```powershell
apatch state disable --scope user --reason "temporary opt out"
apatch state enable --scope user --reason "restore"
```

Project scope:

```powershell
apatch state disable --scope project --path C:\path\to\repo --reason "project opt out"
apatch state enable --scope project --path C:\path\to\repo --reason "project opt in"
```

System scope:

```powershell
apatch state disable --scope system --reason "machine policy"
apatch state enable --scope system --reason "machine policy"
```

Effective enablement requires system, user, and project scopes to all be enabled. Missing state files default to enabled.

## Deploy

Personal/user agent:

```powershell
apatch deploy --scope user --agent both
```

Project agent:

```powershell
apatch deploy --scope project --agent both --project C:\path\to\repo --write-agents-snippet
```

System-managed staging path:

```powershell
apatch deploy --scope system --agent both
```

## Safe replacement

```powershell
apatch status
apatch scan "old text" --fixed
apatch replace --old "old text" --new "new text" --fixed --preview
apatch replace --old "old text" --new "new text" --fixed --apply
apatch verify --old "old text" --new "new text" --fixed
apatch summary
```

## Structural code rewrite

```powershell
apatch ast --lang c --pattern "if ($A)" --rewrite "if ($A != NULL)" --preview
apatch ast --lang c --pattern "if ($A)" --rewrite "if ($A != NULL)" --apply
```

## Transactions

```powershell
apatch tx list
apatch tx rollback <transaction-id> --preview
apatch tx rollback <transaction-id> --apply --force
```

## Markdown links

```powershell
apatch links --mode offline --include-fragments full
```

## AiCoding plugin package

```powershell
apatch package aicoding-plugin --out dist/agent-patch-kit --zip
```
