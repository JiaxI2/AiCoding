# Agent Patch Kit Evaluation

## Scope

This document records the current Agent Patch Kit role inside AiCoding. It focuses on repository-agent patch workflow cost, not model quality.

## Capability Check

Agent Patch Kit provides:

- non-editable user-mode CLI installation;
- `apatch` and `agent-patch` console commands;
- user, project, and system enable/disable scopes;
- repo-scoped Skill deployment for `aicoding-agent-patch-kit`;
- uninstall support;
- AiCoding plugin packaging support;
- marketplace sidecar generation support.

The AiCoding deployment uses the project/repo-scoped path and does not modify `CodingKit/agents/skills`.

## Test Method

The comparison uses deterministic local measurements:

- token estimate: ceiling(character_count / 4);
- baseline context: existing AiCoding maintenance entry files an agent normally reads before platform changes;
- installed context: `apatch brief --format md`, `apatch state status`, and the generated repo snippet;
- timing: PowerShell 7 `Measure-Command` for `apatch brief` and `apatch state status`.

This measures context-entry and guardrail overhead. It does not claim a benchmark for reasoning quality, bug rate, or end-to-end coding speed.

## Current Workflow Value

First-read token cost:

- Without the kit: agents rely on several repo governance docs for patch-operation rules.
- With the kit: `apatch brief --format md` exposes the patch workflow in a compact command-oriented form.

Edit safety:

- State gate, status, scan, preview, apply, verify, and summary are exposed as a single command workflow.

Disable control:

- System, user, and project scopes are available; missing state defaults to enabled.

Marketplace exposure:

- A sidecar and merged local plugin entry expose Agent Patch Kit without editing the Codex-Skills submodule.

Remaining boundary:

- Agent Patch Kit reduces patch-operation overhead but does not replace AiCoding architecture, release governance, or submodule policy.

## Deployment Artifacts

- `.agents/skills/aicoding-agent-patch-kit/`
- `config/agent-patch-kit.json`
- `docs/agent-patch-kit-agents-snippet.md`
- `.agents/plugins/agent-patch-marketplace.json`
- `dist/agent-patch-kit/`
- `.agents/plugins/marketplace.json`