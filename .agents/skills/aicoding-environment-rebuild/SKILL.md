---
name: aicoding-environment-rebuild
description: Rebuild an AiCoding-managed development environment from a valid repository checkout through the fixed sequence fresh-clone proof, bootstrap, baseline inspection, install plan, explicit confirmation, apply, and full verification. Use for new-machine setup, disaster recovery, or deliberate re-creation of all managed Kit, MCP, and runtime Skill state.
---

# AiCoding Environment Rebuild

## Skill Type

This skill is both consistent-workflow and organization-standard.

- consistent-workflow: every rebuild uses the same evidence-backed sequence and
  stops at the same write boundary.
- organization-standard: it preserves AiCoding ownership, JSON contracts,
  acquisition/activation separation, runtime Skill uniqueness, and bounded
  recovery semantics.

## When To Use

- A new machine already has an AiCoding repository checkout and needs all managed
  Kit, MCP, and runtime Skill state installed.
- The managed environment must be deliberately reconstructed after OS recovery,
  tool removal, or confirmed state loss.
- The team needs proof that the current source tree is reproducible before using
  it to rebuild the local runtime.

## When Not To Use

- Normal in-place updates or drift convergence; use `aicoding-upgrade-train`.
- Acquiring the repository or advancing source pins. Clone with recursive
  submodules or complete the acquisition workflow first.
- Repairing one Kit, one MCP component, or runtime Skill exposure only; use the
  corresponding scoped lifecycle plan instead of rebuilding every domain.
- Replacing, deleting, or taking ownership of unmanaged user configuration,
  unknown Skills, or unknown MCP entries.
- Editing capability source or plugin cache files.

## Meaning Of One-Click

"One-click" means one repeatable orchestration owned by this Skill. It does not
mean one unreviewed write command. The lifecycle plan and explicit user approval
remain mandatory because `--scope all` can write three independently owned domains
and there is no cross-domain atomic rollback.

`fresh-clone` is a reconstructability gate: it creates a temporary recursive clone,
overlays relevant current worktree changes, builds there, and runs the requested
profile. It does not install the current machine's managed environment.

## Workflow Contract

Trigger: a request to rebuild, recreate, bootstrap, or recover the complete
AiCoding-managed environment from an existing repository checkout.

Inputs:

- verification profile for the reconstructability gate: `Smoke`, `Full`, or
  `Release`;
- runtime profile: `runtime`, `full`, or `skill-development`;
- `--runtime-skill <name>` when the runtime profile is `skill-development`;
- optional `--source-repository`, `--standalone-root agents|codex`, and
  `--codex-config` selections;
- explicit approval before `--migrate-unmanaged` may be included.

Steps:

1. **Preflight the source and prerequisites without changing managed state.**
   Confirm the checkout contains `config/codex-kit.json`, Git and Go are available,
   and recursive submodules resolve to the commits recorded by the repository:

   ```powershell
   git status --short --branch
   git submodule status --recursive
   go version
   ```

   Stop on an uninitialized (`-`), mismatched (`+`), or conflicted (`U`) submodule.
   Do not advance pins inside this workflow. Record unrelated worktree changes;
   `fresh-clone` overlays them for validation but never authorizes modifying them.

2. **Prove fresh-clone reconstructability.** A clean checkout does not yet have
   `bin/aicoding.exe`, so invoke the control plane through Go:

   ```powershell
   go run ./cmd/aicoding fresh-clone --profile <Smoke|Full|Release> --json
   ```

   Require exit code 0, `schemaVersion=1`, and `ok=true`. On failure, report the
   retained `tempRoot`, failed step, and errors; do not delete the evidence or
   continue to installation.

3. **Bootstrap the local control plane.**

   ```powershell
   go run ./cmd/aicoding bootstrap --json
   bin\aicoding.exe version
   ```

   Bootstrap must return exit code 0 and JSON `ok=true`. The binary version check
   must also exit 0 before it becomes the authority for later steps.

4. **Record the uninstalled or damaged baseline.** Reuse one `<runtime-selection>`
   containing the chosen runtime profile and every applicable optional runtime
   flag in all remaining commands:

   ```powershell
   bin\aicoding.exe lifecycle status --scope all <runtime-selection> --json
   bin\aicoding.exe lifecycle doctor --scope all <runtime-selection> --json
   ```

   A non-converged baseline is expected on a rebuild, but ownership conflicts,
   invalid registry entries, source ambiguity, or duplicate active Skill names are
   blockers. Installation must not work around them.

5. **Generate the complete install plan.**

   ```powershell
   bin\aicoding.exe lifecycle plan --action install --scope all <runtime-selection> --json
   ```

   Review Kit, MCP, and runtime Skill adapter entries separately. Present
   `catalogDigest`, each `inputDigest`, `planDigest`, intended writes, warnings,
   ownership refusals, Marketplace/plugin actions, and any Hook review requirement.

6. **Confirm the write.** The user or named platform owner must explicitly approve
   the presented plan. After a long pause or concurrent source/config change,
   regenerate it and compare the digest evidence. A changed `inputDigest` voids the
   earlier approval.

7. **Apply exactly the approved selection.**

   ```powershell
   bin\aicoding.exe lifecycle install --scope all <runtime-selection> --json
   ```

   The apply flags must match the plan. Do not add `--kit`, `--component`, or the
   adapter-level `--all` flag to `--scope all`. Never add `--migrate-unmanaged`
   after approval; it must have appeared in the reviewed plan.

8. **Verify convergence and runtime discovery.**

   ```powershell
   bin\aicoding.exe lifecycle status --scope all <runtime-selection> --json
   bin\aicoding.exe doctor --all <runtime-selection> --json
   bin\aicoding.exe verify --profile Smoke <runtime-selection> --configured --json
   ```

   If the plan did not touch Codex MCP registration, `--configured` may be omitted;
   record that decision. After plugin installation or refresh, instruct the user to
   review/trust changed Hooks through the supported Codex Hook review UI.

9. **Stop on partial failure; do not invent a global rollback.** Read adapter
   evidence before proposing recovery:

   | Domain | Recovery authority |
   |---|---|
   | Kit | `bin\aicoding.exe lifecycle rollback --scope kit --last --json` only when a last Kit snapshot is reported |
   | MCP | the failed operation's own config backup/staged-runtime restoration evidence |
   | runtime Skill | the migration rollback manifest reported by the runtime Skill domain |

   Report successful and failed domains separately. Do not loop, uninstall all, or
   delete caches/junctions to simulate rollback.

Exit criteria: fresh-clone proof and bootstrap succeeded; the approved install and
all verification commands have exit code 0 and JSON `ok=true`; status is converged;
runtime Skill names have one active source; the user received digest, state, Hook,
and rollback evidence locations.

Validation: formal AiCoding results are accepted only when `schemaVersion`, `ok`,
`errorKind`, and exit code agree. Human-readable text is not success evidence.
This CLI/hook gate blocks apply on a missing approval or changed digest, and blocks
completion on any required verification failure.

## Safety Boundaries

- Never use this workflow as permission to clone over, reset, clean, or delete an
  existing workspace.
- Never edit the Codex plugin cache, managed Codex configuration blocks, runtime
  junctions, venvs, install state, or rollback manifests directly.
- Never enable a disabled plugin silently or overwrite `AICODING_HOME`.
- Never expose the whole Codex-Skills repository, canonical source directories, or
  generated plugin Skills under a user Skill root.
- Preserve unmanaged user assets byte-for-byte unless the plan explicitly includes
  approved `--migrate-unmanaged` behavior and reports a rollback manifest.
- A Full or Release fresh-clone proof is stronger source evidence, but it does not
  replace hardware acceptance or an actual product release gate.

## Gate Rules

- **CLI checker:** validate this Draft with
  `pwsh tools/specialty/aicoding-skill.ps1 verify -Skill aicoding-environment-rebuild -RepoRoot . -Json`.
  Pass requires exit code 0 and `ok=true`; an example blocking failure is
  `local absolute path reference` or `missing frontmatter.description`. Also run
  the active `aicoding-user-skill-creator` Skill's `quick_validate.py` and
  `skill_gate.py validate`.
- **Hook gate:** `.aicoding/user-skills` is ignored Draft state, so repository
  pre-commit/CI intentionally does not claim to protect it. Until adoption, the
  three manual CLI/lint validations are the blocking substitute. Adoption must wire
  the authoritative Skill verification into the owned Kit and release gate.
- **MCP tool library:** no MCP orchestration tool is required. Git, Go, and the
  formal AiCoding CLI provide all deterministic checks; MCP servers remain lower
  capability tools and do not own the rebuild workflow.
- **Skip rationale:** no wrapper script or second test/lifecycle aggregator is
  added because `fresh-clone`, `bootstrap`, lifecycle, doctor, and verify already
  expose the required stable JSON contracts. Manual review remains for ownership,
  plan intent, Hook trust, and partial-recovery decisions.

## Verification

Run from the AiCoding repository root:

```powershell
pwsh tools/specialty/aicoding-skill.ps1 verify -Skill aicoding-environment-rebuild -RepoRoot . -Json
```

Then run `quick_validate.py` and `skill_gate.py validate` from the active
`aicoding-user-skill-creator` Skill. All three commands must exit 0.

Forward-test at least these prompts without applying writes:

- "新电脑已经递归克隆 AiCoding，按 full profile 重建全部环境。"
- "环境坏了，一键重装，但保留我自己配置的 Skill 和 MCP。"
- "只修复 visio-mcp。" The Skill must route this to a scoped MCP lifecycle plan,
  not to a full rebuild.

## Examples

- New machine, normal runtime: choose runtime profile `runtime`, run Smoke
  fresh-clone proof, review the three-domain install plan, then apply and verify.
- Full personal toolset: choose runtime profile `full` and canonical standalone root
  `agents`; any unmanaged collision blocks unless migration was separately approved.
- Skill development: choose `skill-development` plus exactly one
  `--runtime-skill`; the plugin must be disabled by the governed profile flow.

## Human Confirmation

- **Owner/确认人:** the user requesting rebuild or the named AiCoding platform owner.
- **Accepted gates:** the owner accepts the fresh-clone, bootstrap, CLI/lint, plan,
  runtime audit, verification, and Hook review gates before apply is approved.
- **Manual review scope:** unmanaged ownership, `--migrate-unmanaged`, per-domain
  plan intent, Hook trust, and any partial recovery remain human-only decisions.
- **Explicit decision:** record `approved`, `approved with risk`, or `rejected`
  after presenting the plan. A failure requires a new decision after evidence and
  bounded recovery options are reviewed.
