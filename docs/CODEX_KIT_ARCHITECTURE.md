# Codex Kit Architecture

AiCoding 是 Codex kit 的平台集成仓库。当前 main 的可观测标准是 Go-first 控制面、Taskfile 短路由、PowerShell 专项保留。

## Ownership

AiCoding owns:

- `config/kit-registry.json` and `config/kits/*.json`;
- Go CLI under `cmd/aicoding` and `internal/*`;
- Taskfile routing;
- `.github/workflows/aicoding-ci.yml`;
- repository hooks and governance docs;
- CodingKit external assets under `CodingKit/examples`, `CodingKit/modules`, `CodingKit/platforms`, `CodingKit/tests`, and `CodingKit/tools`.

AiCoding does not own canonical skill source. `CodingKit/agents/skills` is a read-only submodule dependency.

## Runtime Boundary

Plugin runtime state is managed through supported install/update/verify flows. Do not edit Codex plugin cache directly and do not copy CodingKit asset directories into plugin packages.

## Default Gates

```powershell
bin\aicoding.exe smoke --json
bin\aicoding.exe ci --profile Smoke --json
bin\aicoding.exe full --json
bin\aicoding.exe release gate --json
```

DocSync is enforced by `bin/aicoding.exe docsync`, `.githooks/pre-commit`, and `.github/workflows/aicoding-ci.yml`.

## PowerShell Boundary

PowerShell remains for specialty tooling only: tag planning / overlay compatibility, PowerShell quality, safety, Plan Mode, external skill workflows, and hardware/toolchain diagnostics.