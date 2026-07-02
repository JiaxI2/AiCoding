# MVP Task Progress Monitor

v0.11.1 adds a lightweight progress monitor for multiple small features.

## Runtime Progress

```text
.agent-dev-kit/progress/progress-board.json
.agent-dev-kit/progress/PROGRESS_BOARD.md
```

These are generated runtime files and do not need to be committed.

## Human-readable Official Progress

```text
spec/MVP_PROGRESS.md
```

This file can be committed when the MVP plan/status needs to be shared.

## Commands

```powershell
aicoding-agent-kit progress init --repo .
aicoding-agent-kit progress status --repo .
aicoding-agent-kit progress update --repo . --id F-001 --status doing --current "Writing failing test"
aicoding-agent-kit progress board --repo .
```

## Agent Rule

The Agent should show progress at handoff points:

```text
Current MVP
Active task
Blocked task
Next task
Gate status
```
