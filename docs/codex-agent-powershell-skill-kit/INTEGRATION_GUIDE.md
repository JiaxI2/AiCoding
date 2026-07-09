# AiCoding Integration Guide

## Correct Source-Ownership Model

AiCoding is a platform/integration repository. It should not own the canonical source for this kit.

This kit uses a non-canonical runtime mirror model:

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

The `.agents/skills/codex-agent-powershell-skill-kit` directory is required at runtime so Codex/AiCoding can discover the skill in repo scope. It is not the canonical source. It is a generated/materialized mirror created from the packaged sub-kit.

## Files to Add to AiCoding

Add these platform integration files to AiCoding when this kit is part of the current package set:

```text
config/codex-agent-powershell-skill-kit.json
dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/
docs/codex-agent-powershell-skill-kit/
bin/aicoding.exe lifecycle install --all --json
scripts/status-codex-agent-powershell-skill-kit.ps1
bin/aicoding.exe lifecycle uninstall --all --json
scripts/verify-codex-agent-powershell-skill-kit.ps1
scripts/test-codex-agent-powershell-skill-kit.ps1
.agents/plugins/marketplace.json
.codex-agent-powershell-skill-kit/install-state.json
```

Do not add `aicoding-overlay/.agents/skills/...` as a hand-maintained source folder.

## Runtime Materialized Files

The installer creates or refreshes this runtime mirror:

```text
.agents/skills/codex-agent-powershell-skill-kit/
```

The mirror includes `RUNTIME_MIRROR_NOTICE.md` and `.runtime-mirror.json` so future agents know it is generated, not canonical.

## Why This Model

- Keeps AiCoding consistent with sidecar kit lifecycle scripts.
- Avoids treating AiCoding as the canonical skill source repository.
- Keeps repo-scoped discovery available for Codex agents.
- Makes install, verify, status, uninstall, and re-sync deterministic.

## Recommended Commit Policy

Two valid modes exist:

1. Package-only commit:
   Commit `dist/`, `config/`, `scripts/`, `docs/`, and marketplace entry. Do not commit `.agents/skills/codex-agent-powershell-skill-kit/`. Run install after clone to materialize the runtime mirror.
2. Offline-ready commit:
   Commit the generated `.agents/skills/...` mirror too. Keep `RUNTIME_MIRROR_NOTICE.md` and `.runtime-mirror.json`. Never manually edit the mirror; regenerate from `dist/` when updating.

For AiCoding, prefer package-only unless clone-and-run without install is explicitly required.

## Do Not

- Do not expose the whole external source repo as AiCoding's runtime skill root.
- Do not edit generated plugin cache directly.
- Do not replace `.agents/plugins/marketplace.json` wholesale.
- Do not default to Windows PowerShell 5.1.
- Do not let the agent execute rewrite plans automatically.
- Do not manually edit `.agents/skills/codex-agent-powershell-skill-kit` and call it source-of-truth.