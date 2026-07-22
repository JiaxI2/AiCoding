# Kit Lifecycle Architecture

Status: Accepted and Frozen

当前产品使用 Go-native lifecycle control。`internal/lifecycle` 以静态 Adapter Catalog
组合 Kit、MCP 和 runtime Skill，并将选择结果转换为 `ExecutionPlan`；不引入动态插件系统，
也不复制各领域已有实现。Lifecycle 的可观测入口是
`bin/aicoding.exe lifecycle ...`、`bin/aicoding.exe export ...` 和聚合门禁。

## Unified Static Adapters

统一命名空间支持：

- `kit`：复用 `internal/kit` 的 registry、plan、apply、status、doctor、verify 和 rollback；
- `mcp`：复用 `internal/mcpcontrol` 的 component selection、lifecycle、status、doctor 和 verify；
- `runtime-skill`：以显式 PowerShell specialty adapter 调用 runtime Skill profile 与 audit 脚本。

runtime Skill 的只读聚合路径先读取既有 config/state：本仓未配置 runtime skill，且调用方
没有显式给出 `--runtime-skill`、`--runtime-profile full` 或 source repository 时，adapter
返回 `skipped`，不启动 PowerShell 或 Git 进程。任何显式选择和所有写动作仍进入完整 probe，
因此该 Fast Path 只消除无配置成本，不减少显式覆盖。

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

每次 Kit 命令先把 registry 与 referenced manifests 组合为 `kit-catalog` 内容树。Plan、apply、
doctor 和 verify 使用同一批 detached manifest values；JSON adapter result 返回
`inputDigest`，lifecycle 返回静态 adapter `catalogDigest` 与 `planDigest`。

## Manifest Model

Kit registry entries live in `config/kit-registry.json`; manifests live in `config/kits/*.json`.
Schema v2 manifest 可选地声明一个互斥的 content-pinned source：

```json
{"kind":"git","url":"https://example.invalid/upstream.git","commit":"<40-hex>"}
{"kind":"content","digest":"sha256:<64-hex>"}
```

Git `commit` 必须是完整 40-hex，不接受 branch、tag 或缩写 SHA；content digest 必须是小写
`sha256:<64-hex>`。`source` 整体可选，因此已有 manifest 保持兼容；一旦提供，未知字段和混合
shape 都会被 `additionalProperties:false` 与严格解码拒绝。

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

## Pinned Registration and Local Materialization

外部 Kit 使用两个阶段，网络边界只存在于注册阶段：

```text
repository-local manifest
  -> kit register [--prefetch]     # 登记 metadata；可后台预取
  -> kit prefetch --id <id>        # fetch 完整 commit 并 rev-parse，写内容寻址 cache
  -> lifecycle install/update      # 只读本地 cache；networkCalls=0
  -> .aicoding/state/kits/<id>/source
```

`kit register` 原子更新已有 Kit registry 与 dependency binding，不 vendor source。
Git pin cache 位于 `<git-common-dir>/aicoding/pins/<source-identity>`；content pin 使用同一 scope，
由调用方预先放入内容后按 digest 本地验证。它是 cache 的第六个 scope：只有未被 registry 引用的
内容寻址条目可清理，被引用条目保留。

结构验证中的路径不变量是“仓库本地文件存在，或已解析 pin 中存在”。pin 未预取时 plan/apply
返回 `evidence-missing`，并给出 `requiredAction: aicoding kit prefetch --id <id> --json`；它不会
联网，也不会把缺失路径视为通过。安装 state 绑定 source identity 与 materialized content
identity；前移 pin 会改变 Kit catalog digest，所以旧 Validation Receipt 不能复用。卸载只删除
Kit-owned materialization，不删除共享 pin cache，也不修改外部 checkout。

## Go Control Plane

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe kit verify --all --profile Lifecycle --json
bin\aicoding.exe kit register --manifest config/kits/<id>.json --prefetch --json
bin\aicoding.exe kit prefetch --id <id> --json
bin\aicoding.exe lifecycle plan --action install --scope kit --all --json
bin\aicoding.exe lifecycle install --scope kit --all --json
bin\aicoding.exe export --all --zip --json
```

Kit 内部只读 planning/verification 使用 `internal/runner` 有界并发并保持稳定结果顺序；
lifecycle 跨 adapter 当前以单并发执行。State-writing actions 和 ZIP writing 保持串行，state
与 rollback 仍由 Kit 领域拥有。

产品级 `doctor --all` 与 `verify --profile` 的彼此独立只读检查也复用同一 `runner.Run`
有界并发，结果按登记索引写回；并行只改变 elapsed 信封，不改变有序结论字节。

统一 lifecycle 不推断默认领域；每次调用都必须显式选择
`--scope kit|mcp|runtime-skill|all`。

## PowerShell Specialty

`specialty-pwsh` commands may exist only for explicit specialty workflows. Kit manifest 中的
specialty command 只验证 shape 与 path，不由默认 Kit adapter 执行。runtime Skill adapter
是唯一例外：仅在 `--scope runtime-skill|all` 且写操作显式指定 profile 时调用已登记脚本。
