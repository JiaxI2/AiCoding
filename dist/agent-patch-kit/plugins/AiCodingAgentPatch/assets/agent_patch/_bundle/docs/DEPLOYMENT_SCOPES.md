# Deployment Scopes

Agent Patch Kit can be deployed through CLI to system, user, or project locations.

## User / personal agent deployment

```powershell
apatch deploy --scope user --agent both
```

Targets:

```text
%USERPROFILE%\.agents\skills\aicoding-agent-patch-kit
%USERPROFILE%\.codex\skills\aicoding-agent-patch-kit
```

## Project deployment

```powershell
apatch deploy --scope project --agent both --project C:\path\to\repo --write-agents-snippet
```

Targets:

```text
<repo>\.agents\skills\aicoding-agent-patch-kit
<repo>\.codex\skills\aicoding-agent-patch-kit
<repo>\docs\agent-patch-kit-agents-snippet.md
```

## System deployment

```powershell
apatch deploy --scope system --agent both
```

Targets are staged under `%ProgramData%\AgentPatchKit` on Windows. This is useful for enterprise or machine-managed agent wrappers. Codex user/project discovery remains the most portable default.

## Install script equivalents

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -DeployScope user -Agent both
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -DeployScope project -ProjectRoot C:\path\to\repo -Agent both -WriteAgentsSnippet
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\install-agent-patch-kit.ps1 -DeployScope system -Agent both
```
