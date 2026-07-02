# Agent Integration Prompt

## Domain-neutral Kit Rule

Do not write application-specific examples into the Kit package.

When the target repository has a concrete domain, generate examples only inside that target repository's PRD, option matrix, selected solution, ADR, and implementation plan.

## Fuzzy Requirement Rule

1. If the user requirement is fuzzy, do not implement immediately.
2. Generate `spec/PRD_OPTIONS.md` with 2-5 viable technical/implementation options.
3. Each option must include architecture, flow, pros, cons, risks, validation, effort, and recommended conditions.
4. The Agent may recommend an option but must ask for human selection.
5. The selected option must be written to `spec/SELECTED_SOLUTION.md`.
6. The decision must be recorded in `.agent-memory/DECISIONS.md`.
7. Architecture-level selections must create or update ADR.
8. Selected solution changes must force review/update of PRD, APP_FLOW, IMPLEMENTATION_PLAN, traceability, and tests.
9. MVP implementation must be split into small features and tracked with progress board.
10. The Agent should show current progress at handoff points.
