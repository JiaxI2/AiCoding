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
bin\aicoding.exe verify hooks --json
```

The default smoke gate checks that repository hooks exist and use the prebuilt Go CLI fast path. Hooks never use `go run`; run `bootstrap` before enabling them. Use the PowerShell verifier only as an explicit specialty check.

## Validation Context Gate

`.githooks/pre-push` forwards Git's stdin protocol to `aicoding hook pre-push`. The Go gate reads
`local_ref local_oid remote_ref remote_oid`, loads `config/validation-policy.json`, and checks the
tree of each actual `local_oid`. It never substitutes current HEAD.

The default policy requires a Release Receipt for `refs/heads/main` and `refs/tags/*`; other refs
are explicitly outside the gate. Main must be fast-forward and cannot be deleted; release tags
cannot be deleted. A missing Receipt reports the exact ref and required profile. The remedy runs
validation outside the hook, then retries the push.

## Rules

- Hooks must declare their owner Kit and trigger.
- Hooks must support verification.
- Hook output should be machine-readable, preferably JSON.
- Hook failures must identify the Kit, hook id, and command path.
- Multiple Kits must not silently overwrite the same hook.
- Hooks must not run tests or builds, write the worktree, stash/reset/checkout, or push recursively.
- State-based hook install and uninstall are reserved for a later phase; v2.0 freezes declaration and verification.
