# AiCoding Integration Guide

## Correct source-ownership model

AiCoding is a platform/integration repository. It should not own the canonical source for this kit.

This kit therefore uses a **non-canonical runtime mirror** model:

```text
Canonical kit source/package
        |
        v
AiCoding sub-kit package copy
  dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/
        | install/sync
        v
Repo-scoped runtime mirror
  .agents/skills/codex-agent-powershell-skill-kit/
```

The `.agents/skills/codex-agent-powershell-skill-kit` directory is required at runtime so Codex/AiCoding can discover the skill in repo scope. However, it is **not** the canonical source. It is a generated/materialized mirror created from the packaged sub-kit.

## Files to add to AiCoding

Add these platform integration files to AiCoding:

```text
config/codex-agent-powershell-skill-kit.json
dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/
docs/codex-agent-powershell-skill-kit/
scripts/install-codex-agent-powershell-skill-kit.ps1
scripts/status-codex-agent-powershell-skill-kit.ps1
scripts/uninstall-codex-agent-powershell-skill-kit.ps1
scripts/verify-codex-agent-powershell-skill-kit.ps1
scripts/test-codex-agent-powershell-skill-kit.ps1
.agents/plugins/marketplace.json
.codex-agent-powershell-skill-kit/install-state.json
```

Do **not** add `aicoding-overlay/.agents/skills/...` as a hand-maintained source folder. In v1.2.1 this overlay folder is intentionally absent.

## Runtime materialized files

The installer creates or refreshes this runtime mirror:

```text
.agents/skills/codex-agent-powershell-skill-kit/
```

The mirror includes `RUNTIME_MIRROR_NOTICE.md` and `.runtime-mirror.json` so future agents know it is generated, not canonical.

## Why this model

- Keeps AiCoding consistent with Agent Patch Kit / AI Debug Repair Kit style lifecycle scripts.
- Avoids treating AiCoding as the canonical skill source repository.
- Keeps repo-scoped discovery available for Codex agents.
- Makes install, verify, status, uninstall, and re-sync deterministic.

## Recommended commit policy

Two valid modes exist:

1. **Package-only commit**
   - Commit `dist/`, `config/`, `scripts/`, `docs/`, and marketplace entry.
   - Do not commit `.agents/skills/codex-agent-powershell-skill-kit/`.
   - Run install after clone to materialize the runtime mirror.

2. **Offline-ready commit**
   - Commit the generated `.agents/skills/...` mirror too.
   - Keep `RUNTIME_MIRROR_NOTICE.md` and `.runtime-mirror.json`.
   - Never manually edit the mirror; regenerate from `dist/` when updating.

For AiCoding, prefer package-only unless you specifically want clone-and-run with no install step.

## Do not

- Do not expose the whole external source repo as AiCoding's runtime skill root.
- Do not edit generated plugin cache directly.
- Do not replace `.agents/plugins/marketplace.json` wholesale.
- Do not default to Windows PowerShell 5.1.
- Do not let the agent execute rewrite plans automatically.
- Do not manually edit `.agents/skills/codex-agent-powershell-skill-kit` and call it source-of-truth.
