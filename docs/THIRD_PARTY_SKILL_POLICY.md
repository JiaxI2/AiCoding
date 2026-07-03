# Third-Party Skill Policy

Third-party skills are not AiCoding canonical source. They must enter the repository through an isolated source, cache, verify, install, and state flow.

## Source Registry

Sources are tracked in `config/skill-sources.json`:

```json
{
  "schemaVersion": 1,
  "sources": [
    {
      "name": "example",
      "type": "git",
      "url": "https://github.com/example/skills.git",
      "trust": "third-party",
      "updatePolicy": "manual",
      "pin": "tag-or-commit"
    }
  ]
}
```

## Commands

```powershell
pwsh scripts/aicoding-skill.ps1 sources -Json
pwsh scripts/aicoding-skill.ps1 add-source -Name example -Url https://github.com/example/skills.git
pwsh scripts/aicoding-skill.ps1 download -Source example -Skill skill-id
pwsh scripts/aicoding-skill.ps1 verify -Skill skill-id -Json
pwsh scripts/aicoding-skill.ps1 install -Skill skill-id -Json
pwsh scripts/aicoding-skill.ps1 update -Skill skill-id -Pin tag-or-commit -Json
pwsh scripts/aicoding-skill.ps1 remove -Skill skill-id -Force -Json
```

Downloads go to `.aicoding/skill-cache/third-party/<source>/<skill-id>/`. Verified installs go to `.agents/skills/<skill-id>/` and write `.aicoding/state/skills/<skill-id>/install-state.json`.

## Rules

- Do not download directly into `.agents/skills`.
- Do not enable an unverified third-party skill.
- Do not auto-update third-party skills.
- Updates must be pinned to a commit or tag.
- Unknown or untrusted sources stay in cache or quarantine.
- A third-party skill can become AiCoding-owned only through fork or copy-as-new, new id if needed, origin record, manifest update, verification, docs, and changelog.
