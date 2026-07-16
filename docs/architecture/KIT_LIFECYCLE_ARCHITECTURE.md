# Kit Lifecycle Architecture

当前产品使用 Go-native lifecycle control。`internal/lifecycle` 以静态 adapter 组合 Kit、MCP
和 runtime Skill；不引入动态插件系统，也不复制各领域已有实现。Lifecycle 的可观测入口是
`bin/aicoding.exe lifecycle ...`、`bin/aicoding.exe export ...` 和聚合门禁。

## Unified Static Adapters

统一命名空间支持：

- `kit`：复用 `internal/kit` 的 registry、plan、apply、status、doctor、verify 和 rollback；
- `mcp`：复用 `internal/mcpcontrol` 的 component selection、lifecycle、status、doctor 和 verify；
- `runtime-skill`：以显式 PowerShell specialty adapter 调用 runtime Skill profile 与 audit 脚本。

兼容期内，不带 `--scope` 的 `lifecycle ... --all` 继续保持原 Kit 语义，避免升级后意外修改
用户 Skill 根目录。跨域操作必须显式使用 `--scope all`；install/update 时还必须指定
`--runtime-profile runtime|full|skill-development`。所有 plan 都使用 dry-run，MCP 验证使用
显式或临时 `config.toml`，runtime Skill apply 只有在用户明确选择 profile 后才允许写入。

```powershell
bin\aicoding.exe lifecycle plan --action install --scope all --runtime-profile runtime --json
bin\aicoding.exe lifecycle status --scope all --json
bin\aicoding.exe lifecycle doctor --scope all --json
bin\aicoding.exe lifecycle verify --scope all --profile Smoke --json
```

`rollback --last` 当前只恢复 Kit lifecycle snapshot。MCP 在单次操作内负责配置/venv 失败回滚；
runtime Skill 对被迁移路径写入独立 rollback manifest。CLI 不把这两类局部恢复证据伪装成已完成
的跨域自动 rollback。

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

`specialty-pwsh` commands may exist only for explicit specialty workflows. Kit manifest 中的
specialty command 只验证 shape 与 path，不由默认 Kit adapter 执行。runtime Skill adapter
是唯一例外：仅在 `--scope runtime-skill|all` 且写操作显式指定 profile 时调用已登记脚本。
