# Kit Lifecycle Architecture

当前 main 使用 Go-native kit lifecycle control。Lifecycle 的可观测入口是 `bin/aicoding.exe lifecycle ...`、`bin/aicoding.exe export ...` 和聚合门禁。

## Manifest Model

Kit registry entries live in `config/kit-registry.json`; manifests live in `config/kits/*.json`.

Allowed manifest modes:

- `go-builtin`
- `external-cli`
- `powershell-specialty`
- `declarative`

Allowed command types:

- `builtin-check`
- `builtin-lifecycle`
- `builtin-package`
- `external-command`
- `go-composed`
- `specialty-pwsh`
- `unsupported`

## Go Control Plane

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe kit verify --all --profile Lifecycle --json
bin\aicoding.exe lifecycle plan --action install --all --json
bin\aicoding.exe lifecycle install --all --json
bin\aicoding.exe export --all --zip --json
```

Read-only planning and verification paths use `internal/runner` for bounded parallelism and stable result ordering. State-writing actions and ZIP writing remain serialized.

## PowerShell Specialty

`specialty-pwsh` commands may exist only for explicit specialty workflows. They are validated for shape and path presence but are not default lifecycle execution routes.