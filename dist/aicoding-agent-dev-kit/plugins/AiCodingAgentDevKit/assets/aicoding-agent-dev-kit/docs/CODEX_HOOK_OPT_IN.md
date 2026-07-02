# Codex Hook Opt-in Policy

Codex lifecycle hooks are optional.

They help load context at session start and show the context manifest at stop, but they are not enforcement.

## Default

Do not install project `.codex/hooks.json` unless the user opts in.

## Opt-in

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-native-adapter.ps1 `
  -RepoRoot . `
  -InstallCodexHooks `
  -Json
```

## Trust

After installation, restart Codex and review:

```text
/hooks
```

Only rely on lifecycle automation after hooks are trusted.
