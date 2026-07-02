# Sequential Loading Model

v0.11.1 introduces a real sequential context loader.

The goal is to reduce token usage and improve execution speed by preventing the Agent from reading the whole repository by default.

## Loading Stages

```text
L0 Fast Start
  Reads only CURRENT.md, DECISIONS.md, and changed-file list.

L1 Minimal Task Context
  Adds IMPLEMENTATION_PLAN, TEST_STRATEGY, changed file snippets, and diff summary.

L2 Task and Interface Context
  Adds PRD, APP_FLOW, TECH_STACK, CODING_GUIDELINES, PROJECT_STRUCTURE, touched modules, and tests.

L3 Deep Trace Context
  Adds ADR, BDD, TDD, traceability, and DocSync-relevant files.
```

## Default Agent Rule

```text
Do not read the whole repository.

Start with:
  aicoding-agent-kit load --repo . --stage L0

Then escalate only when needed:
  aicoding-agent-kit load --repo . --auto
```

## Outputs

```text
.agent-dev-kit/context/context-pack.md
.agent-dev-kit/context/context-manifest.json
```

`context-pack.md` is the content the Agent should read.

`context-manifest.json` records what was included, skipped, truncated, and why.

## Escalation Rules

```text
Normal implementation:
  L0 -> L1

CLI / Hook / CI / config changes:
  L0 -> L1 -> L2

Spec / ADR / BDD / TDD changes:
  L0 -> L1 -> L2 -> L3

DocSync failure:
  L3

TDD failure:
  L2
```

## Why Manifest Matters

The manifest prevents hidden context assumptions.

It records:

- stage
- maxChars
- includedFiles
- skippedFiles
- truncatedFiles
- changedFiles
- roughTokens
- escalationReason
