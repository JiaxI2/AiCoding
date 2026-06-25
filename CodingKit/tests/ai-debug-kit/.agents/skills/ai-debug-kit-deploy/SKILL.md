---
name: ai-debug-kit-deploy
description: Deploy and validate AI Debug Kit, including CLI, simulator smoke-test, J-Link dependency checks, active-profile, and deployment report.
---

# AI Debug Kit Deploy Skill

Use this skill when the user asks to install, initialize, validate, repair, or report AI Debug Kit capabilities in a workspace.

## Scope

This skill owns deployment and capability validation:

- Check Python, uv, workspace layout, and `ai-debug` CLI.
- Run `ai-debug doctor`.
- Run `ai-debug backend list --output json`.
- Run `ai-debug smoke-test --workspace <path> --output json`.
- Check optional J-Link support with `ai-debug backend discover --backend jlink --output json`.
- Create and verify `.ai-debug/deployment/active-profile.json`, `backends.json`, and `.ai-debug/targets/jlink-generic.json`.
- Distinguish `detected`, `configured`, `tested`, `validated`, `unsupported`, and `not_tested`.

This skill does not analyze firmware business behavior, tune control parameters, execute Flash, reset hardware, write target memory, or run motors.

## Required Flow

1. Identify the workspace and shell.
2. Check that `ai-debug` is importable or runnable.
3. Run `ai-debug doctor --output json`.
4. Run simulator smoke-test before any hardware probe action.
5. Read the active-profile and confirm `installation_status` is `ready`.
6. If J-Link is requested, verify `pylink-square` availability and run discovery only.
7. Report generated evidence paths and unvalidated hardware items.

## Commands

```powershell
uv run ai-debug doctor --output json
uv run ai-debug backend list --output json
uv run ai-debug smoke-test --workspace . --output json
uv run ai-debug backend discover --backend jlink --output json
```

If the console script is unavailable, use:

```powershell
uv run python -m ai_debug doctor --output json
uv run python -m ai_debug smoke-test --workspace . --output json
```

## References

- Read `references/installation.md` for setup and validation details.
