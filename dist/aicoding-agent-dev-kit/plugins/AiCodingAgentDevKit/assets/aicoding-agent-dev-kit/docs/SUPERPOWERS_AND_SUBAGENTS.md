# Superpowers and Subagents

## Problem

Skills can drift in long contexts because they are not always as stable as system-level instructions.

## Policy

- Use Thin Skill for routing.
- Use subagent templates for stable task roles.
- Use Superpowers only as an optional accelerator.
- Use Hook/CI scripts as executable truth.

## Mapping

| Need | Preferred Layer |
|---|---|
| Requirement questioning | Superpowers brainstorming or spec-reviewer |
| Plan writing | writing-plans or implementation-planner |
| TDD enforcement | tdd-enforcer + validate-implementation-plan |
| Debugging | systematic-debugger |
| CI enforcement | invoke-agent-quality-gate |
