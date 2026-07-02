# Existing Hook Integration

Use this guide when the target repository already has repository-level hooks.

## Step 1: Detect

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/detect-existing-hooks.ps1 -RepoRoot . -Json
```

or:

```powershell
aicoding-agent-kit hook detect --repo .
```

## Step 2: Install Bridge

Non-destructive merge:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/install-hook-bridge.ps1 -RepoRoot . -MergeExistingHook -Json
```

This adds a marked block:

```text
BEGIN AICODING_AGENT_DEV_KIT_BRIDGE
...
END AICODING_AGENT_DEV_KIT_BRIDGE
```

## Step 3: Verify

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-hook-bridge.ps1 -RepoRoot . -Json
```

## Step 4: Remove Bridge

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/uninstall-hook-bridge.ps1 -RepoRoot . -Json
```

## What This Does Not Do

It does not:

- reset `core.hooksPath`
- replace existing hook content
- delete existing checks
- make Codex lifecycle hooks mandatory

## Recommended Existing Hook Shape

```text
pre-commit
  -> existing repo checks
  -> docs sync
  -> git governance
  -> Agent Dev Kit bridge
```
