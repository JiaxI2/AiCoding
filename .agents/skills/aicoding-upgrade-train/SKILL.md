---
name: aicoding-upgrade-train
description: Run a safe, evidence-backed upgrade of AiCoding-managed components (Kit, MCP, runtime Skill) through the fixed sequence preflight, plan, confirm, apply, verify, and bounded rollback handling. Use when the user asks to upgrade, update, refresh, or converge managed components, or after a source pin has been advanced.
---

# AiCoding Upgrade Train

## Skill Type

This skill is both consistent-workflow and organization-standard.

- consistent-workflow: it defines the only approved multi-step sequence for applying
  write-effect `update` actions to managed domains.
- organization-standard: it encodes the frozen platform rules — plan before apply,
  JSON-contract reading, explicit scope, per-domain rollback semantics, and the
  acquisition/activation boundary.

## When To Use

- The user asks to upgrade, update, or refresh one or more managed components
  (Kit, MCP component, runtime Skill exposure).
- A source pin (submodule gitlink, skill source) was advanced and the runtime must
  now converge to it.
- `doctor` or `status` reported drift that an `update` action is expected to resolve.

## When Not To Use

- Advancing upstream source pins themselves: that is acquisition-plane maintenance
  (Codex-Skills gitlink chain, `config/skill-sources.json`), governed by
  `docs/architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md`. Run the acquisition flow
  first; this train then converges the runtime to the new pin.
- First-time installation of a component (use the install flow; same discipline,
  different action).
- Read-only health questions (`status`/`doctor` alone answer those).
- Editing capability source code (that is a capability hot-zone change; run the
  component's verify profile instead).

## Workflow Contract

Trigger: an `update` intent against one or more managed domains.

Inputs: the target scope (`kit` | `mcp` | `runtime-skill` | `all`), component or kit
selection when applicable, the runtime profile when scope includes runtime Skills,
and the current repository state. Resolve one `<selection>` and reuse it unchanged
through preflight, plan, apply, and verify:

| Scope | Required `<selection>` |
|---|---|
| `kit` | `--kit <id>` or `--all` |
| `mcp` | `--component <id>` or `--all` |
| `runtime-skill` | `--runtime-profile runtime\|full\|skill-development`; add `--runtime-skill <name>` for `skill-development` |
| `all` | `--runtime-profile runtime\|full\|skill-development`; do not add `--kit`, `--component`, or the adapter-level `--all` flag |

If `--source-repository`, `--standalone-root`, `--codex-config`, or
`--migrate-unmanaged` is needed, treat it as part of the selection. A write plan
and its apply command must use the same selection byte-for-byte.

Steps:

1. **Preflight (read-only).** Record the baseline:

   ```powershell
   bin\aicoding.exe lifecycle status --scope <scope> <selection> --json
   bin\aicoding.exe lifecycle doctor --scope <scope> <selection> --json
   ```

   Abort and report if the baseline itself is broken in a way an update cannot fix
   (missing registry entries, ownership conflicts) — repair belongs to a separate
   decision, not to a blind update.

2. **Plan (dry-run).** Generate the intent and show it to the user before any write:

   ```powershell
   bin\aicoding.exe lifecycle plan --scope <scope> --action update <selection> --json
   ```

   Present: the per-adapter intents, `catalogDigest`, `inputDigest`, `planDigest`,
   and warnings. `--scope all` requires reviewing each domain's entry individually;
   never summarize a multi-domain plan as a single yes/no without listing domains.

3. **Confirm.** Proceed only after the user approves the presented plan. If the
   facts may have changed since planning (long pause, concurrent edits), re-run
   step 2 and compare `inputDigest` before applying.

4. **Apply.** Execute the same selection that was planned:

   ```powershell
   bin\aicoding.exe lifecycle update --scope <scope> <selection> --json
   ```

5. **Verify.** Prove convergence with read-only evidence:

   ```powershell
   bin\aicoding.exe lifecycle status --scope <scope> <selection> --json
   bin\aicoding.exe lifecycle verify --scope <scope> <selection> --profile Smoke --json
   ```

   For MCP components additionally run
   `bin\aicoding.exe mcp verify <component> --profile Smoke --json` when the
   component's behavior changed, and the protocol probe
   (`mcp verify --configured`) when Codex registration changed.

6. **On failure, apply bounded rollback semantics — never invent a global one:**

   | Domain | Recovery authority |
   |---|---|
   | Kit | `bin\aicoding.exe lifecycle rollback --scope kit --last --json` restores the last Kit state snapshot |
   | MCP | the domain already restores config backup/staged runtime within the failed operation; read its result evidence, do not re-run destructive steps |
   | runtime Skill | consult the migration rollback manifest reported by the domain |

   Report what was restored, what was not, and stop. There is no cross-domain
   atomic rollback; do not claim one.

Exit criteria: `ok=true` on apply and verify, `status` shows the converged state,
and the user has received the digest evidence (`planDigest`, per-domain
`inputDigest`) plus the location of any rollback evidence.

Validation: judge every command by `schemaVersion`/`ok`/`errorKind` in its JSON
result; never infer success from human-readable text. Exit code 0 is required in
addition to `ok=true`. This is the blocking CLI/hook gate: any invalid selection,
changed digest, non-zero exit, or `ok=false` stops the train before the next write.

## Safety Boundaries

- Never run `update` without a presented and approved plan from the same selection.
- Never use `--scope all` as a convenience default; it is a reviewed multi-domain
  decision.
- Never advance source pins, edit plugin caches, junctions, venvs, or Codex managed
  blocks directly; the lifecycle owns activation, the acquisition registry owns pins.
- Never retry a failed write action in a loop; read the domain's error and evidence
  first.
- Runtime Skill writes require the explicit `--runtime-profile` the user chose.
- User-owned assets (unmanaged Codex config entries, user config files, user
  Skills) must remain byte-identical through the train; an ownership refusal from
  the CLI is a protection, not an obstacle to work around.

## Gate Rules

- **CLI checker:** validate this draft with
  `pwsh tools/specialty/aicoding-skill.ps1 verify -Skill aicoding-upgrade-train -RepoRoot . -Json`.
  Pass requires exit code 0 and `ok=true`; an example blocking failure is
  `local absolute path reference` or `missing frontmatter.name`. Also run the
  active `aicoding-user-skill-creator` Skill's `quick_validate.py` and
  `skill_gate.py validate` before installation or adoption.
- **Hook gate:** this Draft is intentionally not wired into repository
  pre-commit/CI because `.aicoding/user-skills` is ignored local state. Until it
  is adopted by an owned Kit, the manual CLI checks above are the blocking review.
  Adoption must add the authoritative Skill verifier to the Kit/release gate.
- **MCP tool library:** no MCP workflow tool is needed. The formal AiCoding CLI
  owns lifecycle actions and JSON evidence; MCP remains a managed capability and
  must not become a second lifecycle controller.
- **Human confirmation:** the Owner/确认人 must accept the per-domain plan and
  digest evidence before apply; the validator cannot replace that decision.
- **Skip rationale:** no helper script or second orchestration CLI is added because
  the existing lifecycle commands already provide deterministic plan/apply/verify
  checks. The Skill only fixes their order and human decision points.

## Verification

Run the draft verifier from the AiCoding repository root:

```powershell
pwsh tools/specialty/aicoding-skill.ps1 verify -Skill aicoding-upgrade-train -RepoRoot . -Json
```

Then run `quick_validate.py` and `skill_gate.py validate` from the active
`aicoding-user-skill-creator` Skill. All three commands must exit 0.

## Examples

- "把 visio-mcp 升到登记的最新状态" → scope `mcp`, component `visio-mcp`:
  preflight → plan → confirm → update → `mcp verify visio-mcp --profile Smoke`.
- "Codex-Skills 的 gitlink 刚推进了，同步一下运行时" → acquisition already done;
  run the train with scope `runtime-skill` and the user's profile.
- "全部升级" → scope `all` with per-domain plan review and explicit
  `--runtime-profile`; expect Kit, MCP, runtime Skill entries listed separately.

## Human Confirmation

- **Owner/确认人:** the user who requested the upgrade, or the named platform owner.
- **Accepted gates:** that owner reviews the CLI plan, per-domain digests, warnings,
  and the manual Draft validation gate before approving apply.
- **Manual review scope:** ownership conflicts, `--migrate-unmanaged`, Hook changes,
  and rollback evidence remain human review decisions.
- **Explicit decision:** record `approved`, `approved with risk`, or `rejected`
  before step 4. A failed apply requires a new explicit confirmation after the
  rollback evidence is reviewed; never treat the first approval as permission to retry.
