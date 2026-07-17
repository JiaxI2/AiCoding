# Kit Lifecycle Test Profiles

Kit Lifecycle v2 uses three explicit validation profiles. Default repository gates must stay on Smoke unless a developer intentionally requests a heavier profile.

## Smoke

Smoke is the default gate for development, PR checks, and `verify-codex-kit.ps1`.

Smoke must:

- finish quickly enough for the default 20-second gate;
- validate registry and enabled manifest consistency;
- verify PowerShell script command paths exist;
- verify declared Kit skills, common registry declarations, and hook registry declarations;
- keep `bin/aicoding.exe fresh-clone --profile Smoke --json` on Smoke by default;
- keep `bin/aicoding.exe kit verify --all --profile Smoke --json` on Smoke by default.

Smoke must not:

- copy the full worktree;
- run full `test -All` kit tests;
- run real `export -All -Zip` package generation;
- write `.aicoding/packages`, `.aicoding/state`, or `.aicoding/tmp` artifacts;
- call Full or Release from default gates.

Recommended commands:

```powershell
pwsh tools/specialty/verify-codex-kit.ps1
bin\aicoding.exe kit verify --all --profile Smoke --json
bin/aicoding.exe fresh-clone --profile Smoke --json
```

## Full

Full is an explicit manual profile for complete kit tests during local development or pre-merge investigation.

Full may:

- run each enabled kit's manifest `test` command;
- copy a temporary source tree for manual source-only restore checks;
- take longer than the default Smoke gate.

Full must not:

- be called by default `verify-codex-kit.ps1`;
- run release bundle restore checks;
- write release package artifacts unless the invoked command explicitly does so.

Recommended command:

```powershell
bin/aicoding.exe test --profile Full --json
```

## Release

Release is an explicit release-only profile for package restore confidence before publishing artifacts.

Release may:

- run real `export -All -Zip -Json`;
- generate kit zip, sha256, and BUILDINFO artifacts under `.aicoding/packages/`;
- create and extract `aicoding-kit-bundle-*.zip`;
- validate `SHA256SUMS.txt`;
- scan extracted bundle contents and nested package zips for local absolute path leaks.

Release must not:

- be called by PR/default workflows;
- be called by default `verify-codex-kit.ps1`;
- commit generated package artifacts unless they are intentionally published release artifacts.

Recommended release commands:

```powershell
bin/aicoding.exe fresh-clone --profile Release --json
bin/aicoding.exe export --all --zip --json
```

## CI Policy

- PR and default branch workflows may run Smoke only.
- Full may be used in manual jobs or local manual validation.
- Release may be used only in manual dispatch, release jobs, or local release preparation.
- `bin/aicoding.exe test --profile Full --json` and `bin/aicoding.exe test --profile Release --json` guard this policy; Smoke-level checks remain Go-native and avoid package writes unless export/release explicitly requires them.
