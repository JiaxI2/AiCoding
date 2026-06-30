# Source Ownership / Runtime Mirror Notice

This PowerShell skill payload is part of the `codex-agent-powershell-skill-kit` package.

For AiCoding integration, AiCoding does **not** own this directory as the canonical skill source.

Ownership model:

- Canonical source: external kit source/package maintained outside AiCoding.
- AiCoding package copy: `dist/codex-agent-powershell-skill-kit/plugins/AiCodingPowerShellSkillKit/skills/codex-agent-powershell-skill-kit/`.
- Repo-scoped runtime mirror: `.agents/skills/codex-agent-powershell-skill-kit/`, materialized by `scripts/install-codex-agent-powershell-skill-kit.ps1`.

Do not manually edit the generated repo-scoped mirror. Update the canonical kit source, rebuild the package, then reinstall/sync.
