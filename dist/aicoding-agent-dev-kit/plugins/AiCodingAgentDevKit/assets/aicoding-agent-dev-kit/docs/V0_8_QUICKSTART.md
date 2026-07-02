# v0.11.1 Quickstart

## Install

```powershell
aicoding-agent-kit install --repo . --spec-pack --memory --hooks --workflow --thin-skill --subagents
```

## First Context Load

```powershell
aicoding-agent-kit load --repo . --stage L0
```

Read:

```text
.agent-dev-kit/context/context-pack.md
```

## Auto Load

```powershell
aicoding-agent-kit load --repo . --auto
```

Auto mode selects stage based on changed files:

- scripts/config/hooks/workflow changes -> L2
- spec/ADR/BDD/TDD changes -> L3
- otherwise -> L1

## Show Manifest

```powershell
aicoding-agent-kit manifest --repo .
```

## Fast Work Loop

```powershell
aicoding-agent-kit fast-start --repo .
aicoding-agent-kit load --repo . --auto
aicoding-agent-kit shard --repo .
aicoding-agent-kit gate --repo . --mode pre-commit
```
