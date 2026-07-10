# ADR 0002: Add Plan Mode Gate to AiCoding Agent Dev Kit

## Status

Accepted

## Context

AiCoding Agent Dev Kit already supports requirement clarification, option matrix planning, Spec/TDD, sequential context loading, decision memory, progress monitoring, and quality gates.

However, Skill triggering is not guaranteed. Some tasks may proceed directly to implementation even when architecture is ambiguous.

## Decision

Add a Plan Mode overlay that introduces:

- explicit Agent modes;
- required plan artifacts;
- one-hook bridge submodules for plan/spec gates;
- machine-checkable plan-mode registry;
- a gate that blocks implementation while user decision is pending.

## Consequences

Positive:

- fewer silent architecture assumptions;
- clearer user/Agent collaboration;
- better traceability between requirement, plan, tasks, and verification;
- aligned with the current AiCoding lifecycle.

Trade-offs:

- more upfront documents;
- local workflows need to learn the new gate;
- some small tasks may need explicit bypass or no-op plan.

## Rollback

Remove:

```text
docs/AGENT_DEV_KIT_PLAN_MODE.md
docs/SPEC_KIT_ADAPTATION.md
docs/SUPERPOWER_SKILL_ADAPTATION.md
config/agent-dev-kit-plan-mode.registry.json
tools/specialty/new-agent-plan-mode-session.ps1
tools/specialty/verify-agent-dev-kit-plan-mode.ps1
tools/specialty/hooks/aef/plan-mode-gate.ps1
tools/specialty/hooks/aef/spec-artifact-gate.ps1
.agents/skills/aicoding-agent-dev-kit-plan-mode/
```

Then remove module entries from `config/hooks-registry.json`.
