# Superpower Skill Adaptation for AiCoding

## Principle

AiCoding should borrow the useful habits from Superpower-style skills without tightly coupling the repository to a specific external skill package.

The useful habits are:

1. explicitly name the current working mode;
2. load only relevant context;
3. update progress after each meaningful step;
4. stop before irreversible or ambiguous decisions;
5. leave a durable handoff record.

## AiCoding equivalents

| Superpower-style habit | AiCoding implementation |
|---|---|
| Mode awareness | `spec/PLAN_MODE.md` and Agent response header |
| Context loading | `aicoding-agent-kit load --repo . --auto` |
| Stepwise progress | `spec/TASKS.md` and `.agent-dev-kit` progress |
| Stop condition | `spec/NEEDS_USER_DECISION.md` |
| Decision memory | `.agent-memory/DECISIONS.md` |
| Handoff | final response + `spec/TRACEABILITY.md` |

## Agent response contract

For non-trivial work, the Agent should include this before editing:

```text
Mode: Plan
Capability domain:
Known context:
Unknowns:
Decision required: yes/no
Next artifact:
```

For handoff:

```text
Mode: Handoff
Implemented:
Verified:
Not verified:
Decision records:
Rollback:
Next step:
```

## Do not overfit

Do not import another skill's private implementation details. Keep AiCoding's own lifecycle, registry, hooks, and scripts as the source of truth.
