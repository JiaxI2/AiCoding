# Common Code Management

Common CodingKit modules are repository assets governed by Go CLI checks.

## Current Verification

```powershell
bin\aicoding.exe kit verify --all --level lifecycle --json
bin\aicoding.exe verify --profile Smoke --json
bin\aicoding.exe test --profile Smoke --json
```

`common-control-kit` is declared in `config/kits/common-control-kit.json` and validated through the registry/manifest checks. No standalone PowerShell verifier is part of the current default standard.
