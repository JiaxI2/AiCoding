---
name: ai-debug-operations
description: Generic AI Debug Kit operation lifecycle for safe, repeatable, auditable debug actions after deployment is complete.
---

# AI Debug Operations Skill

Use this skill when the kit is already deployed and the user asks for a generic debug operation such as reading memory, reading a register, capturing telemetry, exporting a session, or generating an operation report.

## Boundary

This skill executes generic debug actions through the CLI/Core/Policy path. It does not make business root cause claims, create firmware hypotheses, tune parameters, or decide whether a control loop is correct.

J-Link v0.2 operations are read-only: discovery, validate identity, capabilities, memory read, and register read. Treat J-Link as a probe module, not an ARM-only target model. C2000/C28x target behavior belongs in target profiles and future TI DSS/XDS backend support. Do not run halt, reset, write, or Flash through J-Link in v0.2.

Allowed statements:

- "The command returned OK."
- "The read returned N octets."
- "The register value was recorded."
- "The session bundle was exported."
- "No business acceptance rule was provided, so the result is recorded as data only."

Disallowed statements:

- "The root cause is ..."
- "Kp is too high ..."
- "This waveform proves the motor is unstable ..."
- "The product issue is fixed ..."

## Required Flow

1. Read `.ai-debug/deployment/active-profile.json` and `.ai-debug/targets/jlink-generic.json` when J-Link is requested.
2. Confirm workspace, backend, platform, and capability status.
3. Create or reuse a debug session.
4. Classify the requested operation risk level.
5. Check Policy and approval before control or write operations.
6. For J-Link memory read, confirm address and length are inside allowed target profile ranges.
7. Execute only supported CLI operations.
8. Check process exit code and JSON envelope `ok` and `code`.
9. Save raw evidence in the session bundle.
10. Use only user-provided thresholds or project scripts for validation.
11. Close or export the session and report evidence paths.

## Commands

```powershell
uv run ai-debug backend capabilities --backend jlink --output json
uv run ai-debug memory read 0x20000000 4 --backend jlink --output json
uv run ai-debug register read R0 --backend jlink --output json
```

Hardware operations beyond read-only require a later backend version.

## References

- Read `references/operation-lifecycle.md` for the detailed operation lifecycle and safety checks.
