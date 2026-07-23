# Codex Kit Architecture

Status: Accepted and Frozen

AiCoding 是 Codex kit 的平台集成仓库。当前 main 的可观测标准是 Go-first 控制面、Taskfile 短路由、PowerShell 专项保留。

## Ownership

AiCoding owns:

- `config/kit-registry.json` and `config/kits/*.json`;
- Go CLI under `cmd/aicoding` and `internal/*`;
- Taskfile routing;
- `.github/workflows/aicoding-ci.yml`;
- repository hooks and governance docs;
- CodingKit external assets under `CodingKit/examples`, `CodingKit/modules`, `CodingKit/platforms`, `CodingKit/tests`, and `CodingKit/tools`.

AiCoding does not own canonical skill source. `CodingKit/agents/skills` is a read-only submodule dependency.

## Runtime Boundary

Plugin runtime state is managed through supported install/update/verify flows. Install and update compare the released source package `BUILDINFO.json` with the installed cache, refresh drift through the Codex plugin CLI, and write AiCoding install state only after the cache matches. A disabled plugin is never silently re-enabled. Do not edit Codex plugin cache directly and do not copy CodingKit asset directories into plugin packages.

Standalone runtime exposure uses `%USERPROFILE%\.agents\skills` as the canonical root. `%USERPROFILE%\.codex\skills` is a compatibility root that must not retain a second registered copy after profile convergence. Profile migration may back up an unmanaged registered path only when explicitly requested, and records a rollback manifest under the user Codex backup area.

## Dependency Direction Boundary

AiCoding is the composition root. It may register and operate lower-level Kits, Skills and MCP components; those capabilities must not depend on or observe AiCoding.

```text
platform -> integration -> capability -> runtime
```

Product namespaces are reserved for the upper layers. A generic capability keeps domain names, receives runtime configuration through its manifest, and exposes no platform workflow prompt. Stable asset identities do not include versions; versions remain in metadata, documentation, changelog and release authority surfaces.

The executable contract is `config/dependency-governance.json`; `bin/aicoding.exe governance dependencies --json` validates registry coverage, dependency edges, namespace leakage, Skill/MCP responsibility, version opacity and README badge authority.

## External Standalone Skill Chain

GitHub-sourced standalone Skills use nested Git submodules rather than copied source:

```text
AiCoding
-> CodingKit/agents/skills (Codex-Skills submodule)
-> external/<repository> (upstream repository submodule)
-> sourcePaths mapping to the directory containing SKILL.md
-> selected user-level junction
```

Codex-Skills owns the upstream URL, stable-tag policy, pinned gitlink, and external binding manifest. AiCoding owns the runtime name-to-source-path mapping in `config/codex-kit.json`. Clone/update verification must initialize submodules recursively, and the runtime audit must continue rejecting duplicate active Skill names.
Codex-Skills resolves updates from the highest non-prerelease semantic-version tag and pins that release commit. Switching to the runtime profile removes only registered junctions with exact target matches. Removing an external Skill from the kit must delete both the AiCoding runtime mapping and the Codex-Skills URL binding/gitlink in the coordinated cross-repository change.

## Default Gates

```powershell
bin\aicoding.exe doctor --all --json
bin\aicoding.exe verify --profile Smoke --json
bin\aicoding.exe lifecycle plan --action install --scope kit --all --json
bin\aicoding.exe test --profile Smoke --json
```

Smoke and Full are independent development-feedback profiles; choose the radius required by the
change. Publication runs only `bin\aicoding.exe test --profile Release --json`, because Release
is a strict superset of Full on the current 73-leaf registry. The direct profile, Severity, and
executed-Command comparison is recorded in
[TODO 0041](../todolist/done/0041-release-only-publication.md).

DocSync is enforced by `bin/aicoding.exe docsync`, `.githooks/pre-commit`, and `.github/workflows/aicoding-ci.yml`.

## PowerShell Boundary

PowerShell remains for specialty tooling only: tag planning / overlay compatibility, PowerShell quality, safety, Plan Mode, external skill workflows, and hardware/toolchain diagnostics.
