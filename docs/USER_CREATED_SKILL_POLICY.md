# User-Created Skill Policy

User-created skills start as drafts and only become runtime or Kit skills after verification.

## Scopes

- `Draft`: `.aicoding/user-skills/<skill-id>/`
- `RepoLocal`: `.agents/skills/<skill-id>/`
- `Kit`: canonical Kit path under `dist/<kit-id>/.../skills/<skill-id>/`

## Commands

```powershell
pwsh scripts/aicoding-skill.ps1 create -Skill my-skill -Scope Draft -Json
pwsh scripts/aicoding-skill.ps1 verify -Skill my-skill -Json
pwsh scripts/aicoding-skill.ps1 install -Skill my-skill -Json
pwsh scripts/aicoding-skill.ps1 adopt -Skill my-skill -Kit aicoding-agent-dev-kit -Json
pwsh scripts/aicoding-skill.ps1 list -Json
```

`create` scaffolds a draft. `verify` checks frontmatter, `name`, `description`, common secret patterns, and local absolute path leaks. `install` copies a verified draft into the repo-local runtime path and records install state. `adopt` returns the required Kit adoption plan; v2.0 does not silently move files into canonical Kit content.

## Required Content

Each user-created skill needs `SKILL.md`, frontmatter `name`, frontmatter `description`, when-to-use guidance, when-not-to-use guidance, verification command, examples or tests, and safety boundaries.

If a skill becomes part of a Kit, update the Kit manifest `skills.members`, run `verify-skills`, and update docs plus `CHANGELOG.md`.
