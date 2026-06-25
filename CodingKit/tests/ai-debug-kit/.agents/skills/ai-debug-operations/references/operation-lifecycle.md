# Generic Debug Operation Lifecycle

## Preflight

Read `.ai-debug/deployment/active-profile.json` and confirm:

- `schema_version` is supported.
- `installation_status` is `ready` or the requested operation can run with partial capabilities.
- The backend and platform match the user's target.
- The required capability is present and validated.

For J-Link, also read `.ai-debug/targets/jlink-generic.json` and use its `allowed_memory_ranges` for read checks.

## Risk Levels

- L0: local config and artifact inspection.
- L1: read-only observe operations.
- L2: halt, resume, step, breakpoint.
- L3: reset, RAM write, register write.
- L4: Flash, erase, powered hardware motion, OTA.
- L5: OTP, fuse, security lock.

Default policy allows L0/L1. v0.2 J-Link only implements L1 reads. L2+ needs a future explicit approval path and backend implementation.

## Result Checks

Every operation must check:

- Process exit code.
- JSON envelope `ok`.
- JSON envelope `code`.
- Warnings.
- Side effects.
- Read length or readback for data operations.

## Reporting

Reports must include:

- Target and backend identity.
- Actual capabilities used.
- Operation list.
- Evidence paths.
- Side effects.
- Unverified items.

Do not convert raw data into business conclusions unless a separate domain skill or user-provided rule is active.
