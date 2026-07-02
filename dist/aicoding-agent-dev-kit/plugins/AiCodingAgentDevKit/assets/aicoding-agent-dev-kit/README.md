# AiCoding Agent Dev Kit v0.11.1

A reusable, domain-neutral Agent Development Kit for:

- requirement clarification
- option matrix planning
- selected solution sync
- SDD / TDD workflow
- sequential context loading
- lightweight decision memory
- hook bridge
- Codex-native adapter
- progress monitoring

## Domain-neutral rule

This Kit must not include application-specific examples.

Application examples belong in the target repository's own PRD, selected solution, ADR, and implementation plan.

## Clarification mode

```powershell
aicoding-agent-kit clarify init --repo . --requirement "Describe the unclear requirement"

aicoding-agent-kit clarify choose --repo . `
  --option-id OPT-002 `
  --name "Selected generic option name" `
  --reason "Why this option was selected"
```

## Progress mode

```powershell
aicoding-agent-kit progress init --repo . --from-plan
aicoding-agent-kit progress status --repo .
aicoding-agent-kit progress update --repo . `
  --id F-001 `
  --status doing `
  --current "Writing failing test"
```

## Sequential loading

```powershell
aicoding-agent-kit load --repo . --auto
aicoding-agent-kit manifest --repo .
```

## Hook bridge

```powershell
aicoding-agent-kit hook detect --repo .
aicoding-agent-kit hook install-bridge --repo . --merge-existing-hook
```
