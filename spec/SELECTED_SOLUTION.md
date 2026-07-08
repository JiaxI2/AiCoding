# Selected Solution: AiCoding Agent Dev Kit Plan Mode Overlay

Decision Status: Selected

Selected option: integrate the v0.4 Plan Mode overlay as a repo-scoped AiCoding Agent Dev Kit extension, using the existing AiCoding hook registry and lifecycle scripts.

Reasoning:

- Preserve `bin/aicoding.exe` as the lifecycle entrypoint.
- Preserve `config/hooks-registry.json` as the single hook registry.
- Do not modify `CodingKit/agents/skills`, generated plugin packages, or Codex plugin cache.
- Keep Plan Mode behavior auditable through docs, registry, hook scripts, and validation commands.
