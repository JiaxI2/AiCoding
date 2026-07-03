# Hook System

AiCoding v2.0 defines hooks as declared, auditable trigger points. Hooks must not silently modify install, update, uninstall, or package behavior.

## Hook Types

- `repo-hook`: repository hooks such as pre-commit, docsync, or governance checks.
- `kit-hook`: lifecycle-adjacent hooks owned by a Kit.
- `agent-hook`: agent workflow hooks such as quality gates and context loading.

## Registry

Hooks are declared in `config/hooks-registry.json` and may also be referenced from a Kit manifest `hooks` section. Every hook must declare an id, owner Kit, type, trigger, path, and default enabled state.

## Verification

```powershell
pwsh scripts/verify-hooks.ps1 -Json
```

The smoke gate parses the registry, checks unique hook ids, checks that the owner Kit exists, checks trigger text, validates hook paths, and runs PowerShell parser validation for `.ps1` hooks.

## Rules

- Hooks must declare their owner Kit and trigger.
- Hooks must support verification.
- Hook output should be machine-readable, preferably JSON.
- Hook failures must identify the Kit, hook id, and command path.
- Multiple Kits must not silently overwrite the same hook.
- State-based hook install and uninstall are reserved for a later phase; v2.0 freezes declaration and verification.
