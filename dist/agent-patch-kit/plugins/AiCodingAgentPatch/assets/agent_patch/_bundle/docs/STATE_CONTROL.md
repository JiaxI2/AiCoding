# Agent Patch Kit State Control

Agent Patch Kit supports system, user, and project level enable/disable controls.

## Inspect state

```powershell
apatch state status
apatch state where
```

## Enable / disable user scope

```powershell
apatch state disable --scope user --reason "temporarily disable agent patch workflow"
apatch state enable --scope user --reason "restore agent patch workflow"
```

## Enable / disable project scope

```powershell
apatch state disable --scope project --path C:\path\to\repo --reason "project opts out"
apatch state enable --scope project --path C:\path\to\repo --reason "project opts in"
```

## Enable / disable system scope

System scope writes to a machine-level state location. On Windows this is under `%ProgramData%\AgentPatchKit`. Administrator permissions may be required.

```powershell
apatch state disable --scope system --reason "machine policy disabled"
apatch state enable --scope system --reason "machine policy enabled"
```

## Effective rule

The effective result is enabled only when all scopes allow it:

```text
system enabled AND user enabled AND project enabled
```

Missing state files default to enabled.

## PowerShell wrappers

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\disable-agent-patch-kit.ps1 -Scope user
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\enable-agent-patch-kit.ps1 -Scope user
```

For project scope:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\disable-agent-patch-kit.ps1 -Scope project -ProjectRoot C:\path\to\repo
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\enable-agent-patch-kit.ps1 -Scope project -ProjectRoot C:\path\to\repo
```
