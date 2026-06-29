## Agent Patch Kit Safety Rule

Before any text/code modification, use Agent Patch Kit.

Developer-only fast-read:

```powershell
apatch brief --format md
apatch state status
```

If effective state is disabled, do not apply edits unless the user explicitly asks to enable the kit or authorizes override.

Mandatory flow:

1. `apatch status`
2. `apatch scan "<target>" --fixed` unless regex is required
3. `apatch replace ... --preview` or `apatch ast ... --preview`
4. `apatch replace ... --apply` or `apatch ast ... --apply`
5. `apatch verify`
6. `apatch summary`

For Markdown changes, run:

```powershell
apatch links --mode offline --include-fragments full
```

Do not use broad PowerShell regex replacement as the first choice.
