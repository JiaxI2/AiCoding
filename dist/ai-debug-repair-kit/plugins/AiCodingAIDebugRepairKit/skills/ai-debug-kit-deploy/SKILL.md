---
name: ai-debug-kit-deploy
description: >
  Use this skill when installing, uninstalling, validating, or diagnosing the
  AiCoding AI Debug Repair Kit, its CLI assets, Codex plugin packaging,
  profile files, and host toolchain readiness. Do not use it for business
  root-cause analysis.
---

# AI Debug Kit Deploy Skill

Use this skill to install or validate the AI Debug Repair Kit in AiCoding or standalone mode.

## First checks

Run from the AiCoding repository root when integrating into AiCoding:

```powershell
airepair doctor --output json
airepair version --output json
```

If `airepair` is missing, install the kit:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-ai-debug-repair-kit.ps1 -PackageRoot "<package-root>" -Json
```

## Responsibilities

- Verify Python and pip are available.
- Verify `airepair` CLI works.
- Verify `.codex-plugin/plugin.json` exists.
- Verify three skills exist.
- Verify marketplace entry exists.
- Verify example profiles are valid.
- Generate clear deployment status.

## Boundaries

This skill must not:

- Modify project source code.
- Start a repair loop.
- Run flash/reset/halt.
- Treat tool detection as test success.
- Claim hardware validation without an explicit HIL profile and user approval.

End with one of:

```text
ready
partial
failed
```
