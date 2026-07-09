# Maintenance Method

This is the required operating method for maintaining Codex-Skills and AiCoding.

## Authority Chain

Agents must follow this order:

```text
AGENTS.md
-> maintenance Skill
-> docs
-> config and scripts
-> CI and Git hooks
```

Meaning:

1. `AGENTS.md` and nested `AGENTS.md` files define non-negotiable boundaries.
2. The maintenance Skill defines task workflow and mode gates.
3. Docs explain architecture and rationale.
4. `config/` and retained scripts define executable compatibility surfaces.
5. CI, Git hooks, and local verification decide whether work can be considered complete.

## Maintenance Skill

The canonical maintenance workflow is maintained in Codex-Skills:

```text
platform/aicoding-kit-maintenance/SKILL.md
```

When packaged, it is exposed by the AiCoding plugin as:

```text
aicoding-kit-maintenance
```

Use that Skill for architecture changes, plugin packaging, submodule updates, install/update scripts, CodingKit assets, hooks, CI gates, and repository maintenance.

## Standard Modes

### Codex-Skills Source Or Plugin Change

```text
read AGENTS
-> modify canonical source
-> build plugin when bundled
-> compare generated output
-> verify plugin
-> verify skills
-> update docs and CHANGELOG
-> commit Codex-Skills
```

### AiCoding Platform Change

```text
read AGENTS
-> inspect config, Go CLI, docs, and retained scripts
-> modify AiCoding-owned files only
-> verify through Go-native default gates
-> run explicit PowerShell compatibility checks only when the changed surface requires them
-> update docs and CHANGELOG
-> commit AiCoding
```

### Cross-Repository Upgrade

```text
verify and commit Codex-Skills
-> update AiCoding submodule to that commit
-> verify AiCoding
-> refresh installed plugin through Marketplace
-> review hooks when changed
```

AiCoding must not point to uncommitted Codex-Skills files.

## Required Checks

For AiCoding changes, use the Go CLI as the default control plane:

```powershell
go test ./...
go run ./cmd/aicoding bootstrap --json
go build -o bin/aicoding.exe ./cmd/aicoding
bin/aicoding.exe status --all --json
bin/aicoding.exe doctor pwsh --json
bin/aicoding.exe doctor pwsh-budget --json
bin/aicoding.exe cstyle status --json
bin/aicoding.exe docsync ci --json
bin/aicoding.exe skill verify --all --profile Smoke --json
bin/aicoding.exe lifecycle plan --action install --all --json
bin/aicoding.exe full --json
bin/aicoding.exe release gate --json
git diff --check
```

Run `bin/aicoding.exe docsync all --json` or `bin/aicoding.exe docsync release --json` when a change specifically touches DocSync policy or release documentation.

PowerShell checks are explicit compatibility or specialty gates. Keep them for tag planning, release overlay compatibility, PowerShell quality, Plan Mode, external skill workflows, safety, hardware/toolchain diagnostics, and Codex-Skills source/plugin work.

For Codex-Skills changes, run the source repository gates in the Codex-Skills repository:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/build-plugin.ps1 -Plugin AiCoding -Configuration Development -Clean -Verify
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/compare-generated.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-plugin.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-skills.ps1
git diff --check
```

## Skill Authoring Boundary

Keep the system `skill-creator` installed for the built-in Codex skill authoring guidance.

The user-maintained workflow is named `aicoding-user-skill-creator` and displayed as `User-Skill-Creator`. It belongs to the AiCoding kit and must not reuse the runtime name `skill-creator`.

When a task is about creating or validating AiCoding/user-maintained skills, use `aicoding-user-skill-creator`. When a task is about generic system guidance and no AiCoding-specific workflow is needed, the system `skill-creator` remains available.

## Runtime Skill Root Optimization

The source repository, user-level standalone skills, and installed plugin cache are separate runtime layers:

```text
Codex-Skills source repository
-> not a Skill Root
%USERPROFILE%\.agents\skills
-> selected standalone personal Skill links only
AiCoding Plugin cache
-> bundled aicoding-* runtime Skills
```

Use these compatibility entrypoints before changing runtime exposure:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/audit-runtime-skills.ps1 -Json
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/set-codex-skill-profile.ps1 -Profile runtime -DryRun -Json
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/migrate-skill-root.ps1 -Profile full -DryRun -Json
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/restore-legacy-skill-root.ps1 -DryRun -Json
```

Do not move or delete the legacy `%USERPROFILE%\.codex\skills` tree until a dry-run migration report, runtime audit, plugin verification, rollback plan, and user approval exist.

## Completion Rule

Do not claim completion if:

- a required gate failed;
- generated plugin output has unexplained drift;
- the submodule is dirty;
- AiCoding changed Skill source;
- docs or CHANGELOG no longer match behavior;
- DocSync modes, semantic policy, or the single `bin/aicoding.exe docsync` entrypoint are missing or fail status/verify/test checks;
- Hook changes have not been reported for `/hooks` review;
- destructive Git or cache actions were needed but not explicitly authorized.
