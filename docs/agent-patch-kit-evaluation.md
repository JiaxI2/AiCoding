# Agent Patch Kit Deployment Evaluation

## Scope

This evaluation records the Agent Patch Kit v2.1 deployment into the AiCoding repository on 2026-06-29. It focuses on repository-agent workflow cost, not model quality.

## Capability Check

Agent Patch Kit v2.1 is not only an AiCoding component definition. The provided kit includes:

- standalone editable Python CLI installation through scripts/install-agent-patch-kit.ps1;
- apatch and agent-patch console commands;
- user, project, and system deployment scopes;
- repo-scoped Skill deployment to .agents/skills/aicoding-agent-patch-kit;
- uninstall support through scripts/uninstall-agent-patch-kit.ps1;
- AiCoding plugin packaging through apatch package aicoding-plugin;
- marketplace sidecar generation through integrations/aicoding/package-marketplace.ps1.

The AiCoding deployment used the project/repo-scoped path and did not modify CodingKit/agents/skills.

## Test Method

The comparison uses deterministic local measurements:

- token estimate: ceiling(character_count / 4);
- baseline context: existing AiCoding maintenance entry files an agent normally reads before platform changes;
- installed context: apatch brief --format md, apatch state status, and the generated repo snippet;
- timing: PowerShell 7 Measure-Command for apatch brief and apatch state status.

This measures context-entry and guardrail overhead. It does not claim a benchmark for reasoning quality, bug rate, or end-to-end coding speed.

## Results

Before Agent Patch Kit:

- Entry context: AiCoding architecture, maintenance, CodingKit docs, config, and marketplace.
- Lines measured: 505.
- Estimated tokens: 4479.
- Measured command time: manual read path.

After Agent Patch Kit:

- apatch brief --format md: 40 lines, 489 estimated tokens, 467.0 ms.
- Generated AGENTS snippet: 30 lines, 184 estimated tokens, static file.
- apatch state status: 4 effective scope rows, small output, 474.1 ms.

## Pain Point Comparison

First-read token cost:

- Before: agents rely on several repo governance docs, about 4479 estimated tokens for the measured entry set.
- After: apatch brief provides the patch workflow in about 489 estimated tokens.

Edit safety:

- Before: safety rules are distributed across AGENTS, maintenance docs, and agent behavior guidance.
- After: state gate, status, scan, preview, apply, verify, and summary are exposed as a single command workflow.

Disable control:

- Before: no dedicated patch-workflow enable/disable state.
- After: system, user, and project scopes are available; missing state defaults to enabled.

Marketplace exposure:

- Before: AiCoding marketplace only exposed the main AiCoding plugin.
- After: a sidecar and merged local plugin entry expose Agent Patch Kit without editing the Codex-Skills submodule.

Remaining cost:

- Before: repo governance docs are still required for platform boundary decisions.
- After: Agent Patch Kit reduces patch-operation overhead but does not replace AiCoding architecture and release governance.

## Deployment Artifacts

- .agents/skills/aicoding-agent-patch-kit/
- config/agent-patch-kit.json
- docs/agent-patch-kit-agents-snippet.md
- .agents/plugins/agent-patch-marketplace.json
- dist/agent-patch-kit/
- .agents/plugins/marketplace.json