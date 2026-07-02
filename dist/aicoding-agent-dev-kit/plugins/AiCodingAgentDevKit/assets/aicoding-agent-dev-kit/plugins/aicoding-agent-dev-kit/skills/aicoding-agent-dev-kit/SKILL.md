---
name: aicoding-agent-dev-kit
description: Thin Agent entrypoint for requirement clarification, option matrix planning, Spec Pack, TDD, sequential context loading, decision memory, progress monitor, and quality gates.
---

# AiCoding Agent Dev Kit Thin Skill

Use this Skill only as an Agent routing layer.

## Domain-neutral rule

Do not write application-specific examples into the reusable Kit.

Concrete examples belong in the target repository after reading that repository's context.

## If the requirement is fuzzy

1. Do not implement yet.
2. Create or update `spec/PRD_OPTIONS.md`.
3. Show 2-5 technical options with pros, cons, risks, validation, effort, and recommended conditions.
4. Ask the human to select or reject an option.
5. Write the selected option to `spec/SELECTED_SOLUTION.md`.
6. Record the decision in `.agent-memory/DECISIONS.md`.
7. Update PRD / APP_FLOW / IMPLEMENTATION_PLAN / ADR / traceability.

## Before implementation

1. Load context with `aicoding-agent-kit load --repo . --auto`.
2. Read `.agent-dev-kit/context/context-pack.md` and manifest.
3. Check `spec/IMPLEMENTATION_PLAN.md` for Red-Green-Refactor.
4. Initialize or update the progress board.
5. Run the quality gate before commit or handoff.
