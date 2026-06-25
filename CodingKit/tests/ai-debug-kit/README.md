# AI Debug Kit

Reusable platform-oriented debug kit for AI agents and embedded projects.

The current version implements a host-side core with simulator and J-Link read-only backend support:

- `ai-debug` CLI with JSON envelopes.
- Pluggable backend registry for `simulator` and `jlink`.
- Simulator backend for deployment smoke tests.
- J-Link probe module through optional `pylink-square` integration.
- Architecture-neutral target profiles, including C2000/C28x profile metadata.
- Policy-gated simulator write/readback flow.
- Replay-ready session bundle and markdown report.
- Two Codex skills:
  - `.agents/skills/ai-debug-kit-deploy`
  - `.agents/skills/ai-debug-operations`
- Minimal C99 target shim template under `src/ai_debug/target_shim`.

## Quick Start

```powershell
uv venv
uv pip install -e .
uv run ai-debug doctor --output json
uv run ai-debug smoke-test --workspace . --output json
```

J-Link support is optional:

```powershell
uv sync --extra jlink
uv run ai-debug backend discover --backend jlink --output json
uv run ai-debug backend validate --backend jlink --output json
uv run ai-debug memory read 0x20000000 4 --backend jlink --output json
uv run ai-debug register read R0 --backend jlink --output json
```

For test-only fake J-Link behavior:

```powershell
$env:AI_DEBUG_JLINK_FAKE='1'
uv run ai-debug backend discover --backend jlink --output json
```

## Platform Boundary

The platform core performs deterministic debug actions: capability checks, session creation, simulator memory read/write, J-Link read-only memory/register access, evidence recording, and report generation. J-Link is treated as a probe/transport module, not as an ARM-only platform layer.

It does not perform business root cause analysis, FOC tuning, EtherCAT diagnosis, OTA strategy review, automatic Flash operations, reset, halt, or arbitrary target writes. Those must be separate domain skills, future approved backend capabilities, or project-specific verification rules.

## Reference Inputs

This implementation follows the two local requirement documents in the parent `tests/` directory, uses `Aladdin-Wang/Mklink-AI-Probe` as a structural template for Skill + references + Python CLI organization, and references `DigitalAllianceStudio/DSS_DataVisualizer` for TI DSS/XDS-style non-intrusive C2000/DSP data access concepts. See `docs/references/dss-datavisualizer.md` for the boundary between J-Link probe support and future TI DSS/XDS backend work.
