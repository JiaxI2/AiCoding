# Kit Lifecycle Architecture v2

AiCoding now treats a Kit as any capability unit that can be discovered, checked, verified, tested, updated, uninstalled, or exported by the platform lifecycle system. A Kit can be a Codex plugin package, Python CLI package, PowerShell skill kit, repository hook kit, documentation sync kit, CodingKit module, or repository asset package.
## AiCoding Kit System v2.0 Freeze

AiCoding Kit System v2.0 freezes the current platform boundary:

```text
Kit = lifecycle
Skill = invocation capability
Umbrella = task routing
Subskill = focused local capability
Manifest = declaration
Registry = discovery
Profile = validation cost
State = local runtime facts
Package = distribution and restore
Hook = automatic trigger point
Common = reusable source module
Trust = third-party and user-created skill boundary
```

The primary entrypoints are `scripts/aicoding-kit.ps1` for Kit lifecycle and `scripts/aicoding-skill.ps1` for third-party or user-created skill source management. Existing install, update, uninstall, status, verify, and test scripts remain legacy adapter entries behind manifests; v2.0 does not rewrite them and does not add any `*-all.ps1` lifecycle scripts.

Policy documents:

- [Kit Skill Routing](KIT_SKILL_ROUTING.md)
- [Common Code Management](COMMON_CODE_MANAGEMENT.md)
- [Hook System](HOOK_SYSTEM.md)
- [Third-Party Skill Policy](THIRD_PARTY_SKILL_POLICY.md)
- [User-Created Skill Policy](USER_CREATED_SKILL_POLICY.md)
- [Kit Export And Release](KIT_EXPORT_AND_RELEASE.md)
- [Kit Install State](KIT_INSTALL_STATE.md)
- [Kit Lifecycle Test Profiles](KIT_LIFECYCLE_TEST_PROFILES.md)

Default repository verification stays on Smoke. Full and Release profiles are explicit manual or release-only actions.

## Phase 1 Scope

Phase 1 implements the manifest adapter layer only:

```text
scripts/aicoding-kit.ps1
-> config/kit-registry.json
-> config/kits/<kit-id>.json
-> scripts/lib/AiCoding.KitRegistry.psm1
-> scripts/lib/AiCoding.KitRunner.psm1
-> scripts/lib/AiCoding.KitPackage.psm1
-> existing lifecycle scripts or builtin dry-run checks
```

The adapter does not rewrite existing lifecycle scripts and does not turn legacy scripts into wrappers. Existing scripts remain the executable truth for each Kit. The unified entrypoint only loads registry entries, resolves manifests, invokes the registered command, and normalizes the result.

Phase 1 supports these recommended commands:

```powershell
pwsh scripts/aicoding-kit.ps1 list
pwsh scripts/aicoding-kit.ps1 status -All -Json
pwsh scripts/aicoding-kit.ps1 verify -All
pwsh scripts/aicoding-kit.ps1 test -All
pwsh scripts/aicoding-kit.ps1 export -All -Zip -DryRun
pwsh scripts/aicoding-kit.ps1 export -All -Zip -Json
```

`export -DryRun` reports the intended output path and manifest include/exclude rules without writing zip, sha256, or BUILDINFO files. Phase 1.6 adds real `export -Zip` package generation while preserving the same manifest adapter path.

## Phase 1.5: Schema and Regression Gates

Phase 1.5 makes the adapter layer mechanically verifiable without changing install, uninstall, update, or export behavior.

The gate is `scripts/verify-kit-lifecycle.ps1`. It verifies that `config/kit-registry.json` parses, the registry matches `config/schemas/kit-registry.schema.json`, every enabled Kit manifest exists, each enabled manifest matches `config/schemas/kit-manifest.schema.json`, manifest ids match registry ids, PowerShell script command paths exist, manifest `mode` is either `script-adapter` or `declarative`, and `kind` is non-empty.

The same gate also runs the unified lifecycle entrypoint as a regression surface:

```powershell
pwsh scripts/aicoding-kit.ps1 list
pwsh scripts/aicoding-kit.ps1 status -All -Json
pwsh scripts/aicoding-kit.ps1 verify -All
```

`list` must include every enabled Kit, `status -All -Json` must return exactly the enabled Kit count, and default `verify -All -Json` must pass the Smoke verify-light path for every enabled Kit. The forbidden aggregate scripts `install-all.ps1`, `verify-all.ps1`, `test-all.ps1`, `update-all.ps1`, `export-all.ps1`, and `uninstall-all.ps1` must not exist.

`verify-codex-kit.ps1` runs `verify-kit-lifecycle.ps1` as part of the repository gate. The lifecycle gate invokes `aicoding-kit.ps1 verify -All -Profile Smoke -Json`, which is bounded verify-light and does not call legacy full verifier scripts. Explicit legacy full verification requires `-Profile Full`.

Markdown link checking for this phase is scoped to changed files. Full-repository `apatch links` still has historical baseline failures under existing submodule, plugin template, and external test asset content, so those legacy full-repo broken links are recorded but do not hard-block Phase 1.5.

## Phase 1.6: Real Export and Bundle Packaging

Phase 1.6 implements real package export without changing install, update, status, verify, test, or uninstall scripts.

`commands.export.include`, `commands.export.exclude`, and `commands.export.outputName` in each `config/kits/<kit-id>.json` manifest are the executable packaging contract. `scripts/lib/AiCoding.KitPackage.psm1` resolves those rules, stages matching repository files, writes `BUILDINFO.json`, creates the kit zip under `.aicoding/packages/`, computes a `.sha256` sidecar, and writes an adjacent `<package>.BUILDINFO.json` file.

A kit package BUILDINFO contains at least:

- `schemaVersion`;
- `kitId`;
- `kitName`;
- `version`;
- `kind`;
- `manifestPath`;
- `registryPath`;
- `gitCommit`;
- `gitBranch`;
- `createdAt`;
- `includedFilesCount`;
- `packageFile`;
- `sha256File`.

Real single-kit export uses the same command path as dry-run:

```powershell
pwsh scripts/aicoding-kit.ps1 export -Kit aicoding-agent-dev-kit -Zip -Json
```

Real all-kit export still does not implement separate per-kit lifecycle logic. It loads enabled registry entries, calls the same `Export-AiCodingKit` function for each Kit, and only after all Kit exports pass creates a bundle:

```powershell
pwsh scripts/aicoding-kit.ps1 export -All -Zip -Json
```

The bundle is written to `.aicoding/packages/aicoding-kit-bundle-<timestamp>.zip` and contains:

- `registry/kit-registry.json`;
- `manifests/config/kits/*.json`;
- `packages/*.zip`;
- `SHA256SUMS.txt`;
- `BUILDINFO.json`.

`SHA256SUMS.txt` records hashes for the kit zip files included in the bundle. The bundle itself also receives a `.sha256` sidecar and adjacent BUILDINFO file. `-DryRun` remains side-effect free and only reports `wouldInclude`, `wouldExclude`, and `wouldWrite`.

## Phase 1.7: Fresh Clone Restore Test

Phase 1.7 adds `scripts/test-kit-fresh-clone.ps1` with explicit test profiles. The profile policy is defined in [Kit Lifecycle Test Profiles](KIT_LIFECYCLE_TEST_PROFILES.md). The default profile is a 20-second smoke gate:

```powershell
pwsh scripts/test-kit-fresh-clone.ps1 -Profile Smoke -Json
```

Smoke mode does not copy the full worktree, does not run `test -All`, and does not create real export/package artifacts. It validates the registry, enabled manifests, manifest ids, modes, non-empty kinds, PowerShell command paths, and the absence of forbidden aggregate lifecycle scripts.

Source-only restore and package restore remain available only as explicit manual or release-only checks:

```powershell
pwsh scripts/test-kit-fresh-clone.ps1 -Mode SourceOnly -Profile Full -Json
pwsh scripts/test-kit-fresh-clone.ps1 -Mode Package -Profile Release -Json
```

`aicoding-kit.ps1 verify -All` and `aicoding-kit.ps1 test -All` both default to `-Profile Smoke`, which performs manifest smoke checks for all enabled kits. Full and release test execution must be requested explicitly with `-Profile Full` or `-Profile Release`.

Package restore still runs `export -All -Zip -Json`, verifies kit zip `.sha256` sidecars, verifies the newest `aicoding-kit-bundle-*.zip` sidecar, extracts the bundle, checks for `registry/kit-registry.json`, `manifests/config/kits/*.json`, `packages/*.zip`, `SHA256SUMS.txt`, and `BUILDINFO.json`, validates bundle `SHA256SUMS.txt`, and scans extracted bundle content plus nested package zip entries for local absolute path leaks. `verify-codex-kit.ps1` runs only the Smoke profile by default.

`.aicoding/packages/`, `.aicoding/state/`, and `.aicoding/tmp/` are ignored generated/runtime paths. Package outputs are release artifacts only and are not committed by default.

## Registry Contract

`config/kit-registry.json` owns only:

- Kit id;
- enabled flag;
- execution order;
- manifest path.

The registry must not contain install logic. The manifest owns Kit metadata and command bindings.

## Manifest Contract

Each manifest in `config/kits/` uses schema version 2 and defines:

- `id`, `name`, `version`;
- one or more `kind` values;
- `mode`, currently `script-adapter`;
- useful paths for status and packaging;
- `commands` for the lifecycle actions available in Phase 1.

Supported adapter command types are:

- `powershell-script`: invokes an existing repository script with registered args;
- `external-command`: invokes an existing CLI such as `apatch`;
- `composed`: invokes other actions from the same manifest in order;
- `builtin-check`: checks required repository paths without side effects;
- `builtin-package`: emits an export plan in `-DryRun` or creates zip, sha256, and BUILDINFO artifacts in real export;
- `unsupported`: documents why an action is intentionally unavailable.

## All Mode

`-All` never implements separate lifecycle logic. It loads enabled registry entries in order and calls the same `Invoke-AiCodingKitAction` path used by `-Kit`.

The unified JSON shape is:

```json
{
  "schemaVersion": 2,
  "action": "verify",
  "mode": "all",
  "ok": true,
  "summary": {
    "total": 7,
    "ok": 7,
    "failed": 0,
    "skipped": 0
  },
  "kits": []
}
```

## Registered Kits

Phase 1 registers:

- `aicoding-platform`;
- `agent-patch-kit`;
- `ai-debug-repair-kit`;
- `codex-agent-powershell-skill-kit`;
- `docsync-plus`;
- `aicoding-agent-dev-kit`;
- `common-control-kit`.

`docsync-plus` keeps `uninstall` unsupported until state-based ownership exists. `common-control-kit` is a CodingKit module asset, so Phase 1 only checks and exports it.

## Later Phases

The next stages are intentionally not implemented in Phase 1:

- `.aicoding/state/kits/<kit-id>/install-state.json` writes;
- declarative lifecycle execution in `AiCoding.KitLifecycle.psm1`;
- state-based safe uninstall;
- converting old lifecycle scripts into wrappers around `aicoding-kit.ps1`.

Those changes should happen only after the adapter layer is stable and verified.
