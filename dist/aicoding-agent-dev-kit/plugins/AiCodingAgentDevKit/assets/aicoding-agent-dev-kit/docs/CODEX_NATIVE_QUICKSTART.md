# Codex Native Quickstart

## Install platform assets

```powershell
aicoding-agent-kit install --repo . --spec-pack --memory --hooks --workflow --thin-skill --subagents
```

## Install Codex native adapter

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-native-adapter.ps1 -RepoRoot . -Json
```

## Verify

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-native-adapter.ps1 -RepoRoot . -Json
```

## In Codex CLI

Restart Codex, then run:

```text
/plugins
/hooks
```

Review and trust hook definitions if prompted.

## Recommended first prompt

```text
Use AiCoding Agent Dev Kit. Load context with the sequential loader, inspect the manifest, then plan the next TDD task.
```

## Uninstall

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/uninstall-codex-native-adapter.ps1 -RepoRoot . -Json
```
