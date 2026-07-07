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
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/install-docsync-plus.ps1 -DryRun -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/status-docsync-plus.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-docsync-plus.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/test-docsync-plus.ps1 -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode all -Format json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/check-documentation-sync.ps1 -Mode ci -Format json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/verify-codex-kit.ps1
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/install-codex-kit.ps1 -DryRun
bin\aicoding.exe status --all --json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/update-codex-kit.ps1 -DryRun
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
- Hook and CI still call `scripts/check-documentation-sync.ps1`.
- JSON output is stable and parseable.
- No submodule files are modified.
- No plugin cache files are modified.
- README, CHANGELOG, and maintenance docs are updated.
- Required checks pass.

## Rollback

Rollback must be non-destructive:

```text
1. Revert only the DocSync Plus commit or changed files.
2. Restore the previous scripts/check-documentation-sync.ps1 content.
3. Remove scripts/docsync/ and config/docs-sync.semantic.json only if introduced by the failed integration.
4. Do not clean/reset unrelated user changes.
5. Re-run verify-codex-kit and lint-git-governance.
```
