# AiCoding DocSync Plus Specification

## Role

DocSync Plus is a repository-maintenance kit for AiCoding. It upgrades documentation synchronization from a path-only gate to a Git-diff-driven semantic drift gate.

It is not a Codex Skill, not a plugin, and not a replacement for `scripts/check-documentation-sync.ps1`.

## Architecture

```text
git diff changed files
  -> path gate
  -> semantic classifier
  -> PowerShell parameter/ValidateSet check
  -> JSON policy rule check
  -> Markdown command index check
  -> DOCSYNC-NO-DOC-CHANGE quality check
  -> doc drift score
  -> text/json/markdown report
```

## Compatibility

The following existing calls must remain valid:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode pre-commit -Staged
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all
```

DocSync Plus adds:

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode ci -Format json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode release -Format markdown -ReportPath .\docsync-report.md
```

## Scoring

Default score weights are stored in `config/docs-sync.semantic.json`:

```text
apiDrift      35
behaviorDrift 25
policyDrift   20
commandDrift  10
linkDrift     10
```

Modes:

| Mode | Behavior |
|---|---|
| `pre-commit` | Fast local gate. Warning above `preCommitWarn`; fail above `preCommitBlock`. |
| `all` | Full local gate. Fail above `allBlock`. |
| `ci` | Strict CI gate. Fail above `ciBlock`. |
| `release` | Release gate. Fail above `releaseBlock` unless a valid review note exists. |

## No-doc marker policy

`DOCSYNC-NO-DOC-CHANGE` is supported, but it must include a meaningful reason:

```text
DOCSYNC-NO-DOC-CHANGE: only renamed internal fixture directory; no user-facing command, policy, hook, or doc behavior changed.
```

Invalid examples:

```text
DOCSYNC-NO-DOC-CHANGE
DOCSYNC-NO-DOC-CHANGE: skip
DOCSYNC-NO-DOC-CHANGE: no
```

## MVP boundaries

Implemented first:

- PowerShell script surface checks
- JSON policy checks
- Markdown command index checks
- no-doc marker quality checks
- doc drift score
- status/install/verify/test scripts

Deferred:

- full C/C++ AST
- full Python CLI AST
- LLM-generated patches
- automatic PR creation
- release automation
```
