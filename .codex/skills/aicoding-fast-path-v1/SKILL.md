---
name: aicoding-fast-path-v1
description: Use this skill when modifying or validating AiCoding Fast Path V1: Go native hook, governance lint, kit Smoke verify, and performance probes. Do not use it for Full/Release gate rewrites, MCP, repo-index, worktree orchestration, or hardware debug operations.
---

# AiCoding Fast Path V1 Skill

## Purpose

Keep AiCoding local hot-path checks fast and deterministic.

## Use when

- Editing `cmd/aicoding/main.go` or `cmd/aicoding/main_test.go`
- Editing `.githooks/pre-commit` or `.githooks/commit-msg`
- Editing `scripts/*fast-path-v1*.ps1` or `scripts/aicoding-fast.ps1`
- Debugging `aicoding hook`, `aicoding governance`, `aicoding kit verify --profile Smoke`, or `aicoding doctor perf`

## Do not use when

- User asks for repo-index, MCP, worktree orchestration, UI, memory.sqlite, or full platform redesign
- User asks to execute hardware actions: flash, reset, halt, run, write memory, write register
- User asks to replace Full/Release PowerShell/Python gates

## Required commands

```powershell
go test ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe doctor perf --json
```

## Boundary

Fast Path V1 validates structure and staged changes. Full semantic checks remain in PowerShell/Python and CI.
