# DocSync Bridge

Agent Dev Kit must not replace DocSync.

## Responsibility Split

```text
Agent Dev Kit:
  fast-start
  sequential loading
  context-pack
  decision memory
  TDD / spec / traceability gate orchestration

DocSync:
  official documentation consistency
  script/config/hook/CI to docs drift checks
  semantic documentation drift checks
```

## Integration Rule

`invoke-agent-quality-gate.ps1` should call DocSync only if the target repository provides:

```text
scripts/check-documentation-sync.ps1
```

If the script does not exist, Agent Dev Kit continues without failing.

## Recommended Modes

```text
pre-commit:
  run fast Agent Dev Kit checks
  optionally run DocSync pre-commit/staged mode if supported

ci/all/release:
  run full Agent Dev Kit checks
  optionally run DocSync all/ci mode if supported
```

## No Conflict Rule

Runtime context files must not be treated as official documentation.

Ignore:

```text
.agent-dev-kit/cache/
.agent-dev-kit/context/
.agent-dev-kit/shards/
.agent-memory/CURRENT.md
```

Keep or review:

```text
.agent-memory/DECISIONS.md
docs/adr/
spec/
specs/
docs/traceability/
```
