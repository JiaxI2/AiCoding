# AiCoding DocSync Plus Validation Plan

## Scope

This plan validates the DocSync Plus kit after integration into the AiCoding repository.

## Required environment

- Git
- PowerShell 7 preferred
- Windows PowerShell 5.1 compatibility where required by existing gates
- AiCoding repository root

## Required commands

```powershell
bin/aicoding.exe docsync all --json
bin/aicoding.exe docsync all --json
bin/aicoding.exe docsync ci --json
bin/aicoding.exe docsync release --json
bin/aicoding.exe docsync all --json
bin/aicoding.exe docsync ci --json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
bin/aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe status --all --json
bin/aicoding.exe lifecycle plan --action update --all --json
bin\aicoding.exe governance lint --json
git diff --check
```

## Fixture scenarios

`test-docsync-plus.ps1` should cover:

1. Code/script change without docs -> fail.
2. Code/script change with docs -> pass or low score.
3. PowerShell command referenced in README but missing script -> fail.
4. README command uses unsupported `-Mode` value -> fail.
5. Policy file change without docs -> fail.
6. Empty `DOCSYNC-NO-DOC-CHANGE` marker -> fail.
7. Valid `DOCSYNC-NO-DOC-CHANGE` reason -> pass/warning depending on mode.
8. Docs-only change -> pass.

## Acceptance criteria

- Existing entrypoint remains compatible.
- Hook and CI still call `bin/aicoding.exe docsync`.
- JSON output is stable and parseable.
- No submodule files are modified.
- No plugin cache files are modified.
- README, CHANGELOG, and maintenance docs are updated.
- Required checks pass.

## Rollback

Rollback must be non-destructive:

```text
1. Revert only the DocSync Plus commit or changed files.
2. Restore the previous bin/aicoding.exe docsync content.
3. Remove internal/docsync/ and config/docs-sync.semantic.json only if introduced by the failed integration.
4. Do not clean/reset unrelated user changes.
5. Re-run verify-codex-kit and lint-git-governance.
```
