# Common Code Management

Common code is reusable source owned by the AiCoding platform, not by an individual Kit. Current common modules live under `CodingKit/modules/common/**`, including controller, filter, math, protocol, platform, and tests assets.

## Registry

Common modules are declared in `config/common-registry.json`. A module record names the reusable unit, its repository path, owners, tests, README paths, and version.

Kit manifests declare dependencies through `commonDependencies`:

```json
{
  "commonDependencies": [
    {
      "id": "common-control",
      "path": "CodingKit/modules/common/controller",
      "version": "0.1.0",
      "usage": "reusable embedded motor-control modules"
    }
  ]
}
```

## Verification

```powershell
pwsh scripts/verify-common-code.ps1 -Json
```

The smoke gate parses `config/common-registry.json`, checks module paths, checks for README or module docs, checks for tests, and verifies that manifest `commonDependencies` reference registered common modules.

## Rules

- Common code must not be copied into `dist/` or Kit assets as a private fork.
- A Kit that depends on common code must declare the dependency in its manifest.
- Common API changes must update docs, examples, dependent manifests, and `CHANGELOG.md`.
- Common changes should trigger common self-tests, dependent Kit smoke verification, and docs sync.
- Export should include common code through manifest include rules or a future common package, not through manual copying.
