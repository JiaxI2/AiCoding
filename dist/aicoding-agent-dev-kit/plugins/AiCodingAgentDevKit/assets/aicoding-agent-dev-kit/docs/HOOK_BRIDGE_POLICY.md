# Hook Bridge Policy

v0.11.1 changes the Git Hook strategy.

## Core Rule

```text
One repository = one Git Hook entrypoint.
Agent Dev Kit must not overwrite an existing repository hook.
```

If the target repository already has:

```text
.githooks/pre-commit
.git/hooks/pre-commit
git config core.hooksPath
```

the Kit should install a bridge snippet into the existing hook, or only print instructions.

## Why

Many repositories already have governance, DocSync, lint, formatting, and release checks wired into their hook entrypoint.

Adding a second hook or resetting `core.hooksPath` can break existing governance.

## Correct Integration

```text
Existing repo hook
  -> existing checks
  -> DocSync
  -> Git governance
  -> Agent Dev Kit quality gate
```

The Agent Dev Kit only provides:

```text
scripts/invoke-agent-quality-gate.ps1
scripts/detect-existing-hooks.ps1
scripts/install-hook-bridge.ps1
scripts/uninstall-hook-bridge.ps1
```

## Git Hook vs Codex Hook

```text
Git Hook:
  deterministic commit-time enforcement
  should be the repository's single existing entrypoint

Codex Hook:
  optional lifecycle automation
  can load context on session start
  must not be treated as enforcement
```

## Default Behavior

v0.11.1 default install should not:

- create `.githooks/pre-commit`
- overwrite `.githooks/pre-commit`
- reset `git config core.hooksPath`
- install `.codex/hooks.json` without explicit opt-in

## Explicit Opt-in

Use these only when the user explicitly wants them:

```powershell
pwsh -File scripts/install-hook-bridge.ps1 -RepoRoot . -MergeExistingHook -Json
pwsh -File scripts/install-codex-native-adapter.ps1 -RepoRoot . -InstallCodexHooks -Json
```
