# Decision Memory vs DocSync

## Decision Memory

Purpose:

- Short decision anchors
- Fast session recovery
- Human / Agent decision trace
- Low token usage

Files:

```text
.agent-memory/CURRENT.md
.agent-memory/DECISIONS.md
```

## DocSync

Purpose:

- Official documentation consistency
- Code / script / config / hook / CI drift checks
- Release and maintenance readiness

Files:

```text
README.md
CHANGELOG.md
AGENTS.md
docs/**/*.md
config/**/*.md
spec/**
specs/**
```

## No Conflict Rule

Decision memory is a lightweight input to the Agent. DocSync remains the official documentation gate.

The quality gate may read decision memory, but it must not treat `CURRENT.md` as official documentation.

## Recommended Behavior

```text
Routine work:
  update CURRENT.md locally if helpful

Important decision:
  update DECISIONS.md

Architecture decision:
  promote DECISIONS entry to ADR

Code/script/config change:
  run DocSync as before
```
