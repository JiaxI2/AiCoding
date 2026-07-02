# Option Decision to PRD Flow

## Flow

```text
1. User gives fuzzy requirement.
2. Agent creates PRD_OPTIONS.md.
3. Agent proposes 2-5 domain-specific options inside the target repo.
4. Agent recommends one if appropriate, but does not decide silently.
5. Human selects or rejects an option.
6. Agent writes SELECTED_SOLUTION.md.
7. Agent records DECISIONS.md.
8. Agent creates or updates ADR.
9. Agent updates PRD / APP_FLOW / IMPLEMENTATION_PLAN.
10. Agent splits MVP into small tasks.
11. Agent updates MVP_PROGRESS.md and progress-board.json.
```

## Decision Types

```text
Human Decision:
  User directly selects an option.

Agent Proposal, Human Accepted:
  Agent recommends, user confirms.

Rejected:
  User rejects an option, reason must be recorded if important.
```

## Forced Sync

When selected solution changes, these must be reviewed:

```text
spec/PRD.md
spec/APP_FLOW.md
spec/IMPLEMENTATION_PLAN.md
docs/adr/
docs/traceability/
spec/MVP_PROGRESS.md
```

The quality gate includes `validate-solution-doc-sync.ps1`.
