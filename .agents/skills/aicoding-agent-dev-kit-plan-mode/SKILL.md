---
name: aicoding-agent-dev-kit-plan-mode
description: Use when AiCoding work is non-trivial, architecture-sensitive, or fuzzy. Enforces Plan Mode before implementation, adapts Spec Kit phases, and requires user selection before ambiguous architecture changes.
---

# AiCoding Agent Dev Kit Plan Mode

Use this Skill as a router overlay for AiCoding Agent Dev Kit.

## Required first step

Before implementation, declare:

```text
Mode:
Capability domain:
Current context loaded:
Unknowns:
Decision required:
Planned artifacts:
```

## Phase sequence

Use this sequence:

```text
Clarify -> Specify -> Plan -> User Decision -> Tasks -> Implement -> Verify -> Handoff
```

## Fuzzy architecture

If there are multiple plausible technical routes, do not implement.

Create or update:

```text
spec/PRD_OPTIONS.md
spec/NEEDS_USER_DECISION.md
```

Then ask the user to select one option.

Only continue after these exist:

```text
spec/SELECTED_SOLUTION.md
.agent-memory/DECISIONS.md
```

## Spec Kit adaptation

Use Spec Kit's flow as an operating model:

- constitution: AiCoding rules and AGENTS.md
- specify: user intent and constraints
- clarify: question/option gate
- plan: implementation plan
- tasks: execution tasks
- analyze/checklist: verify plan, traceability, and gate
- implement: only after plan and decision gates pass

## Superpower-style habits

Use explicit mode, minimal context loading, progress checkpoints, and stop conditions.

Do not depend on an external Superpower package being installed.

## Required commands

```powershell
pwsh scripts\new-agent-plan-mode-session.ps1 -Feature "<feature>" -Description "<description>" -NeedsDecision -Json
pwsh scripts\verify-agent-dev-kit-plan-mode.ps1 -Json
pwsh scripts\hooks\aef\plan-mode-gate.ps1 -Event manual -Mode warn -Json
```

## Handoff contract

Final answer must include:

```text
Mode:
Implemented:
Verified:
Not verified:
Decision records:
Rollback:
Next step:
```
