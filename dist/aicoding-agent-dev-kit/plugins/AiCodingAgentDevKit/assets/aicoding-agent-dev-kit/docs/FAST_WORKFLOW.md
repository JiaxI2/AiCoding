# Fast Agent Workflow

## 1. 快速开始

```powershell
aicoding-agent-kit fast-start --repo .
aicoding-agent-kit context --repo . --mode changed --max-chars 12000
```

## 2. 快速执行

```powershell
aicoding-agent-kit changed --repo .
aicoding-agent-kit shard --repo .
aicoding-agent-kit gate --repo . --mode pre-commit
```

## 3. 快速收尾

```powershell
aicoding-agent-kit compact --repo .
aicoding-agent-kit token-audit --repo . --max-file-chars 20000
```

## Recommended Agent Loop

```text
fast-start
-> context-pack
-> implement one shard
-> run focused tests
-> update memory
-> gate pre-commit
-> commit
```
