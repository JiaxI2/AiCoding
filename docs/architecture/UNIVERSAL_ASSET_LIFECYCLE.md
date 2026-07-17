# Universal Asset Lifecycle

## Contract

`internal/asset` is the reusable API. `cmd/assetkit` is a thin CLI. Existing Kit, Skill and MCP commands can migrate incrementally by calling this API without changing their current user-facing commands.

Extension interfaces:
- `Adapter`: type-specific validation and lifecycle hooks.
- `Source`: local, Git, GitHub Release or registry package resolution.
- `Executor`: controlled entrypoint execution.
- `Manager`: deterministic lifecycle orchestration.

## Guarantees

- safe relative paths and ZIP traversal protection;
- SHA-256 package and installed-tree integrity;
- staging, atomic replacement and one-step rollback;
- lockfile ownership and installed-file inventory;
- required/optional dependency model;
- managed and editable installation modes;
- user configuration remains separate from managed payload;
- deterministic deep merge for JSON objects;
- purge is explicit; normal uninstall preserves user overrides.

## Commands

```text
go run ./cmd/assetkit validate DIR
go run ./cmd/assetkit pack DIR --out FILE
go run ./cmd/assetkit install FILE --mode managed|editable
go run ./cmd/assetkit update FILE
go run ./cmd/assetkit uninstall ID [--purge]
go run ./cmd/assetkit rollback ID
go run ./cmd/assetkit verify ID
go run ./cmd/assetkit list
go run ./cmd/assetkit config-set ID dotted.key JSON_VALUE
```

Run `pwsh ./scripts/test-universal-asset-lifecycle.ps1` on Windows after cloning.
