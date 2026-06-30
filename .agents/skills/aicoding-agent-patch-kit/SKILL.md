---
name: aicoding-agent-patch-kit
description: Use this skill whenever an agent modifies repository text, Markdown, scripts, or code. Enforce the Agent Patch Kit workflow: state gate, git status, rg/apatch scan, preview, apply with transaction snapshot, verify, Markdown link validation when docs changed, and diff summary. Avoid direct PowerShell regex edits.
---

# AiCoding Agent Patch Kit

Use this skill for safe repository edits, especially on Windows where raw PowerShell regular-expression replacements are error-prone.

## First-read rule for agents

Before using this kit in a repository, quickly load the developer-only brief and verify install mode:

```powershell
apatch install doctor
apatch brief --format md
apatch state status
```

`apatch install doctor` should report `install_mode: non-editable / user mode` and `bundle_assets: OK` for a normal user install. Editable/dev installs are allowed only when the user is actively developing this kit.

If `apatch state status` reports disabled, stop using this kit for apply/edit operations unless the user explicitly asks to re-enable it or authorizes override.

## Shell preference on Windows

When running repository verification, install, status, update, or documentation checks on Windows, prefer PowerShell 7 (`pwsh`) as the default shell. Use Windows PowerShell 5.1 (`powershell`) only for explicit compatibility gates or when a repository script requires it. This avoids Windows PowerShell 5.1 UTF-8/no-BOM parsing issues while still preserving compatibility coverage.

Preferred form:

```powershell
"C:\Program Files\PowerShell\7\pwsh.exe" -NoProfile -ExecutionPolicy Bypass -File scripts\verify-codex-kit.ps1
```

Compatibility form:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-codex-kit.ps1
```

## Documentation governance for AiCoding repositories

When editing AiCoding-governed README, CHANGELOG, release, or repository metadata:

- `README.md` defaults to Chinese-first bilingual content.
- The top of `README.md` must keep a visible bilingual switch linking `README_CN.md` and the English section.
- README entry pages should include a clickable environment preview with links to install, kit, verification, and governance sections.
- `CHANGELOG.md`, Tag messages, GitHub Release notes, and GitHub About descriptions use Chinese first, then concise English.
- GitHub About metadata should describe Agent Patch Kit and AI Debug Repair Kit in bilingual form.

## Non-negotiable workflow

Before modifying text or code, run:

```powershell
apatch status
apatch scan "<target>" --fixed
```

For simple literal text replacement, run preview before apply:

```powershell
apatch replace --fixed --old "<old>" --new "<new>" --preview
apatch replace --fixed --old "<old>" --new "<new>" --apply
```

For code-structure replacement, use ast-grep through `apatch ast`:

```powershell
apatch ast --lang c --pattern "<pattern>" --rewrite "<rewrite>" --preview
apatch ast --lang c --pattern "<pattern>" --rewrite "<rewrite>" --apply
```

After modifications, run:

```powershell
apatch verify --old "<old>" --new "<new>" --fixed
apatch summary
```

If Markdown files changed, also run:

```powershell
apatch links --mode offline --include-fragments full
```

## Scope controls

Agent Patch Kit can be enabled or disabled by system, user, or project scope:

```powershell
apatch state status
apatch state disable --scope project --reason "project opts out"
apatch state enable --scope project --reason "project opts in"
```

Effective state is enabled only when system, user, and project scopes are all enabled. Missing state files default to enabled.

## Rollback

`apatch replace --apply` and `apatch ast --apply` create an automatic transaction snapshot unless `--no-tx` is passed.

List and inspect transactions:

```powershell
apatch tx list
apatch tx rollback <transaction-id> --preview
```

Rollback only with explicit force:

```powershell
apatch tx rollback <transaction-id> --apply --force
```

Use `--clean-created` only when the user explicitly authorizes deleting files created after the transaction began.

## Rules for agents

- Prefer `--fixed` for literal text, paths, URLs, Markdown links, and Windows paths.
- Use regex only when a fixed string cannot express the target.
- Never do broad PowerShell `Get-Content | -replace | Set-Content` edits without preview.
- Do not apply a replacement until the preview has been checked.
- Do not apply edits when the kit is disabled, unless explicitly authorized.
- Do not hide `git diff --check` failures.
- If docs changed, run the Markdown link validator.
- End with a concise diff summary.


## Installation-mode rule

For normal use, Agent Patch Kit must be installed in non-editable mode. The original zip and extracted source directory may be deleted after `apatch install doctor` passes. Do not use `pip install -e` unless the user explicitly asks for development mode. If `apatch` fails with `ModuleNotFoundError: No module named 'agent_patch'`, reinstall v0.2.2 with `scripts\repair-agent-patch-kit.ps1` or `python -m pip install --force-reinstall .` from the v0.2.2 kit root.
