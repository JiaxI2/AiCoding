# Decision Memory Policy

## Goal

Keep Agent memory short, useful, and reviewable.

This Kit does not try to implement a full memory system. It only provides a lightweight decision anchor so future Agent sessions can quickly understand:

- What the current goal is.
- What important human decisions have been made.
- What important Agent proposals were accepted or rejected.
- Which decisions should be promoted to ADR.

## Files

```text
.agent-memory/
├── README.md
├── CURRENT.md
└── DECISIONS.md
```

## CURRENT.md

Use this file for the current working state only.

Recommended maximum: 40 lines.

Required sections:

```markdown
# Current State

## Current Goal

## Active Task

## Next Step

## Blockers
```

## DECISIONS.md

Use this file only for important decisions.

Decision template:

```markdown
## D-0000: <Decision title>

- Type: Human Decision | Agent Proposal, Human Accepted | Rejected
- Status: Proposed | Accepted | Superseded | Rejected
- Date: YYYY-MM-DD
- Context:
- Decision:
- Impact:
- Link:
```

## What counts as a decision

Record these:

- Platform architecture choice
- Hook / CI behavior change
- CLI command contract
- Install / uninstall policy
- Destructive-action safety policy
- TDD / SDD process rule
- Human rejection of an Agent proposal
- Agent proposal accepted by a human

Do not record these:

- Routine progress updates
- Test command output
- Full logs
- Long summaries
- Temporary implementation notes
- Repeated status reports

## Relationship with ADR

`DECISIONS.md` is a lightweight staging area.

Promote to ADR when the decision is:

- Architectural
- Hard to reverse
- Cross-cutting
- Security or safety related
- Likely to affect future contributors

Use:

```powershell
aicoding-agent-kit decision promote-adr --repo . --id D-0001
```

or:

```powershell
pwsh -File scripts/decision-promote-adr.ps1 -RepoRoot . -DecisionId D-0001 -Json
```

## Relationship with DocSync

Decision memory is not a replacement for official documentation.

Recommended Git policy:

```gitignore
.agent-dev-kit/cache/
.agent-dev-kit/context/
.agent-dev-kit/shards/
.agent-memory/CURRENT.md
```

Recommended committed file:

```text
.agent-memory/DECISIONS.md
```

If a decision changes user-visible behavior, architecture, CLI, Hook, CI, or release behavior, update the official docs too.

## Compatibility Note

Older Kit versions used `progress.txt`, `lessons.md`, and `session-summary.md`.
v0.11.1 replaces them with `CURRENT.md` and `DECISIONS.md`.
