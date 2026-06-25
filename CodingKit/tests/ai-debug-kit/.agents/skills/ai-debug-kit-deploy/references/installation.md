# AI Debug Kit Installation and Validation

## Local Setup

From the kit root:

```powershell
uv venv
uv pip install -e .
uv run ai-debug doctor --output json
```

The always-validated backend is `simulator`. J-Link is optional, implemented as a probe module, and uses the `jlink` extra:

```powershell
uv sync --extra jlink
uv run ai-debug backend discover --backend jlink --output json
```

When `pylink-square` is missing, J-Link commands must return `DEPENDENCY_MISSING` instead of crashing.

## Smoke Test

Run:

```powershell
uv run ai-debug smoke-test --workspace . --output json
```

Expected evidence:

- `.ai-debug/deployment/active-profile.json`
- `.ai-debug/deployment/smoke-test.json`
- `.ai-debug/deployment/backends.json`
- `.ai-debug/targets/jlink-generic.json` with generic and C2000/C28x profile metadata
- `.ai-debug/sessions/<session-id>/manifest.json`
- `.ai-debug/sessions/<session-id>/actions.jsonl`
- `.ai-debug/sessions/<session-id>/final-report.md`

## Safety Defaults

- Simulator validation is automatic.
- J-Link v0.2 validation is discovery, connect identity, capabilities, read-only memory, and register read.
- Halt, reset, RAM write, register write, and Flash require future explicit approval paths.
- Level 4 operations are not part of v0.2 automated deployment.
