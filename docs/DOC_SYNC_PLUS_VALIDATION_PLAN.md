# AiCoding DocSync Plus Validation Plan

## Scope

This plan validates the current DocSync Plus capability in the AiCoding repository.

## Required Environment

- Git
- Go CLI built as `bin/aicoding.exe`
- PowerShell only when the changed surface is a PowerShell specialty gate
- AiCoding repository root

## Required Commands

```powershell
bin/aicoding.exe docsync staged --json
bin/aicoding.exe docsync all --json
bin/aicoding.exe docsync ci --json
bin/aicoding.exe docsync release --json
bin/aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe status --all --json
bin/aicoding.exe lifecycle plan --action update --all --json
bin\aicoding.exe governance lint --json
git diff --check
```

## Fixture Scenarios

`test-docsync-plus.ps1` should cover:

1. Code/script change without docs -> fail.
2. Code/script change with docs -> pass or low score.
3. PowerShell command referenced in README but missing script -> fail.
4. README command uses unsupported mode value -> fail.
5. Policy file change without docs -> fail.
6. Empty `DOCSYNC-NO-DOC-CHANGE` marker -> fail.
7. Valid `DOCSYNC-NO-DOC-CHANGE` reason -> pass/warning depending on mode.
8. Docs-only change -> pass.

## Acceptance Criteria

- Hook and CI call `bin/aicoding.exe docsync`.
- JSON output is stable and parseable.
- No submodule files are modified.
- No plugin cache files are modified.
- README, CHANGELOG, and maintenance docs match current behavior.
- Required checks pass.

## Rollback

Rollback must be non-destructive:

```text
1. Revert only the DocSync Plus commit or changed files.
2. Restore the current `bin/aicoding.exe docsync` behavior from Git history when needed.
3. Remove `internal/docsync/` and `config/docs-sync.semantic.json` only when they were introduced by the failed change.
4. Do not clean/reset unrelated user changes.
5. Re-run Go CLI verification and governance lint.
```