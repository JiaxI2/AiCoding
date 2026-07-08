# Kit Export And Release

Kit packages are generated artifacts under `.aicoding/packages/` and are not committed to `main`.

## Commands

```powershell
bin/aicoding.exe export -Kit aicoding-agent-dev-kit -Zip -Json
bin/aicoding.exe export -All -Zip -Json
bin/aicoding.exe export -All -Zip -DryRun -Json
```

Single-Kit export reads `commands.export.include`, `commands.export.exclude`, and `commands.export.outputName` from the Kit manifest. All-Kit export iterates enabled registry entries and calls the same `Export-AiCodingKit` path for every Kit before creating a bundle.

## Outputs

- `.aicoding/packages/<kit-id>-<version>.zip`
- `.aicoding/packages/<kit-id>-<version>.sha256`
- `.aicoding/packages/<kit-id>-<version>.BUILDINFO.json`
- `.aicoding/packages/aicoding-kit-bundle-<timestamp>.zip`
- `.aicoding/packages/aicoding-kit-bundle-<timestamp>.sha256`
- `.aicoding/packages/aicoding-kit-bundle-<timestamp>.BUILDINFO.json`

Bundle contents include `registry/kit-registry.json`, `manifests/config/kits/*.json`, `packages/*.zip`, `SHA256SUMS.txt`, and `BUILDINFO.json`.

## Profile Policy

Smoke must never write `.aicoding/packages/`. Full can run broader tests but does not perform release restore. Release can run real export, bundle extraction, SHA256 validation, and absolute path leak scanning.
