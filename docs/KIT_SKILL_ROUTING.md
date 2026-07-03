# Kit Skill Routing

AiCoding v2.0 treats a Kit as the lifecycle unit and a Skill as the agent invocation unit. A Kit can own one umbrella skill and any number of member skills, but install, update, uninstall, and export remain Kit-level actions.

## Manifest Contract

Skill declarations live in `config/kits/<kit-id>.json`:

```json
{
  "skills": {
    "umbrella": {
      "id": "aicoding-agent-dev-kit",
      "path": "dist/aicoding-agent-dev-kit/plugins/AiCodingAgentDevKit/skills/aicoding-agent-dev-kit/SKILL.md",
      "role": "router"
    },
    "members": [
      {
        "id": "sdd-workflow",
        "path": "dist/aicoding-agent-dev-kit/plugins/AiCodingAgentDevKit/skills/sdd-workflow/SKILL.md",
        "role": "subskill"
      }
    ]
  }
}
```

The umbrella role must be `router` or `umbrella`. Member roles must be `subskill`. Skill ids must be unique inside the same Kit.

## Commands

```powershell
pwsh scripts/aicoding-kit.ps1 skills -All -Json
pwsh scripts/aicoding-kit.ps1 verify-skills -All -Json
pwsh scripts/aicoding-kit.ps1 skills -Kit aicoding-agent-dev-kit -Json
pwsh scripts/aicoding-kit.ps1 verify-skills -Kit aicoding-agent-dev-kit -Json
```

`skills` lists declared umbrella and member skills. `verify-skills` checks that every declared `SKILL.md` exists, has frontmatter, includes `name` and `description`, uses a valid role, and does not duplicate a skill id within its Kit.

## Boundary

Do not add `install-skill`, `uninstall-skill`, `update-skill`, or `export-skill` as Kit lifecycle replacements. If v2.1 needs partial runtime toggles, prefer `enable-skill` and `disable-skill` state entries while keeping the Kit as the lifecycle owner.
