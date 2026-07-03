# AiCoding Agent Dev Kit Plan Mode

## Purpose

This document upgrades AiCoding Agent Dev Kit from "clarify and implement" into a plan-first workflow.

The goal is not to copy Spec Kit or Superpower internals. The goal is to align AiCoding with the stable ideas behind them:

- make the plan explicit before code changes;
- keep user intent, architecture choice, tasks, and validation traceable;
- force user selection when architecture is ambiguous;
- keep the Agent in a visible mode: Clarify, Specify, Plan, Tasks, Implement, Verify, or Handoff.

## Plan Mode state machine

```text
Clarify -> Specify -> Plan -> User Decision -> Tasks -> Implement -> Verify -> Handoff
```

## Mode definitions

| Mode | Agent may do | Agent must not do |
|---|---|---|
| Clarify | ask questions, inspect repo read-only, create `spec/PRD_OPTIONS.md` | edit implementation files |
| Specify | write user intent, acceptance criteria, constraints | choose architecture silently |
| Plan | write `spec/IMPLEMENTATION_PLAN.md`, risks, validation, rollback | implement code |
| User Decision | show 2-5 options and wait | continue implementation |
| Tasks | write `spec/TASKS.md` and `spec/TRACEABILITY.md` | skip test/verify tasks |
| Implement | change files according to selected plan | change plan silently |
| Verify | run Smoke/golden/schema/lint checks | hide failures |
| Handoff | summarize evidence and rollback path | claim unverified success |

## Required artifacts

For fuzzy architecture:

```text
spec/PRD_OPTIONS.md
spec/NEEDS_USER_DECISION.md
```

For selected architecture:

```text
spec/SELECTED_SOLUTION.md
.agent-memory/DECISIONS.md
```

For implementation:

```text
spec/IMPLEMENTATION_PLAN.md
spec/TASKS.md
spec/TRACEABILITY.md
```

## Fuzzy architecture rule

If the requirement has more than one plausible architecture, integration pattern, data model, safety strategy, or lifecycle boundary, the Agent must stop and ask for user selection.

The Agent must show 2-5 options with:

- option name;
- when it fits;
- implementation impact;
- verification impact;
- rollback impact;
- risk;
- recommendation.

No implementation may continue while `spec/NEEDS_USER_DECISION.md` exists.

## Codex Plan Mode adaptation

Codex Plan Mode should be used as a behavioral mode even when the local Codex UI does not expose a dedicated Plan Mode switch.

In AiCoding, Plan Mode means:

1. read context and current registries;
2. map the request to capabilities;
3. write or update spec artifacts;
4. detect uncertainty;
5. ask the user to choose if needed;
6. only then implement.

## Enforcement

This workflow is enforced by:

```text
scripts/verify-agent-dev-kit-plan-mode.ps1
scripts/hooks/aef/plan-mode-gate.ps1
scripts/hooks/aef/spec-artifact-gate.ps1
config/agent-dev-kit-plan-mode.registry.json
config/agent-hook-modules.registry.json
```
