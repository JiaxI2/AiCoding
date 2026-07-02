# Requirement Clarification Mode

v0.11.1 adds a requirement clarification mode for fuzzy requirements.

The goal is to make the Agent behave like a planning assistant before implementation:

```text
Fuzzy requirement
  -> Ask clarifying questions
  -> Generate multiple solution options
  -> Compare pros / cons / risks / validation paths
  -> Ask human to select
  -> Write selected solution into official specs
  -> Split MVP into small tasks
  -> Track progress
```

## Domain-neutral Kit Policy

This Kit must not contain application-specific examples.

The Kit provides only generic scaffolding:

```text
Option A
Option B
Option C
```

Concrete product, target artifact, hardware, business, UI, backend, or domain examples belong in the target repository's own PRD and design documents, not in the reusable Kit.

## Required Artifacts

```text
spec/PRD_OPTIONS.md
spec/SELECTED_SOLUTION.md
spec/APP_FLOW.md
spec/IMPLEMENTATION_PLAN.md
spec/MVP_PROGRESS.md
.agent-memory/DECISIONS.md
docs/adr/*.md
```

## Golden Rule

No implementation before option selection.

```text
Clarify -> Compare -> Select -> Sync Docs -> Split MVP -> Implement
```
