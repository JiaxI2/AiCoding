# Spec Review Gates

## Gate 1: Requirement Review

Required files:

- `spec/PRD.md`
- `spec/APP_FLOW.md`

Must answer:

- What problem is solved?
- Who is the user?
- What is in scope?
- What is out of scope?
- What does completion mean?

## Gate 2: Design Review

Required files:

- `spec/TECH_STACK.md`
- `spec/CODING_GUIDELINES.md`
- `spec/PROJECT_STRUCTURE.md`
- `docs/adr/*.md` when architecture changes

Must answer:

- Which versions and tools are used?
- Where should files be created?
- What must not be touched?
- Which decisions are irreversible without ADR?

## Gate 3: Task and TDD Review

Required files:

- `spec/IMPLEMENTATION_PLAN.md`
- `spec/TEST_STRATEGY.md`

Each task must include:

1. Write failing test.
2. Run test and confirm failure.
3. Implement minimal code.
4. Run test and confirm pass.
5. Refactor.
6. Run test again.
7. Update progress / lessons / evidence.
8. Commit.
