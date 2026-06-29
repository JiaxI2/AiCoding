# Agent Patch Kit Architecture

## Goal

Provide a small CLI that agents can call instead of repeatedly writing long, fragile PowerShell snippets.

## Layers

```text
Codex Skill / AGENTS.md rule
        ↓
apatch CLI
        ↓
rg / Python literal replacement / ast-grep / lychee / git / task
        ↓
preview / apply / verify / rollback
```

## Why this architecture

- `rg` is used for fast search and fixed-string scans.
- Python writes simple replacements to avoid PowerShell encoding and string-escaping pitfalls.
- `ast-grep` handles structural code rewrites.
- `lychee` handles Markdown links and anchors.
- Git diff and transaction snapshots make changes reviewable and reversible.

## Deployment modes

- Personal agent: `%USERPROFILE%/.agents/skills` and/or `%USERPROFILE%/.codex/skills`.
- Project agent: `<repo>/.agents/skills` and/or `<repo>/.codex/skills`.
- AiCoding Marketplace package: `dist/agent-patch-kit/plugins/AiCodingAgentPatch` plus marketplace JSON.
