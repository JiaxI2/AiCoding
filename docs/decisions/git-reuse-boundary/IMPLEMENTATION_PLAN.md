# Implementation Plan: Git Reuse Boundary

Plan Status: Approved

目标：把 [GIT_REUSE_BOUNDARY.md](../../architecture/GIT_REUSE_BOUNDARY.md) 声明的边界
变成可执行门禁。不新增 CLI 命令，不修改 snapshot/runner/report 契约，不新增 adapter。

## 现状事实（实现前基线）

生产代码中 `internal/gitx` 之外直接 `exec.Command("git", ...)` 的调用点（共 7 处）：

| 文件 | 行为 | 读/写 |
|---|---|---|
| `internal/cstyle/cstyle.go:413` | `rev-parse --show-toplevel` | 读 |
| `internal/cstyle/cstyle.go:425` | diff 类参数组合 | 读 |
| `internal/platform/files.go:20` | `rev-parse --show-toplevel` | 读 |
| `internal/kit/plugin_runtime.go:219` | `config core.hooksPath .githooks` | 写（已登记 hook 安装流程） |
| `internal/kit/structure.go:586` | `status --short` | 读 |
| `internal/lifecycle/runtime_skill.go:132` | `rev-parse HEAD` | 读 |
| `internal/lifecycle/runtime_skill.go:185` | `rev-parse --path-format=absolute --git-common-dir` | 读 |

当前 import gitx 的生产包：`internal/cli`、`internal/docsync`、`internal/governance`、
`internal/kit`、`internal/pwshregex`、`internal/repohealth`、`internal/tagpolicy`。
迁移后新增：`internal/cstyle`、`internal/platform`、`internal/lifecycle`。

## Step 1：git 调用点收编到 gitx

把上表 7 处调用改为 `gitx.Run(repo, args...)`：

- `gitx.Run` 已支持指定 repo 目录与 stderr 并入错误，逐点核对输出解析等价性
  （原 `CombinedOutput` 调用点注意：gitx 返回 stdout 与含 stderr 的 error，
  解析逻辑若依赖 stderr 混排需调整为读 error）。
- 不给 gitx 增加语义函数；如确需公共解析 helper，只允许"输出→Go 值"形态
  （与现有 `StagedFiles` 同类）。
- 约束：`internal/platform` import gitx 后，gitx 永远不得 import platform
  （由 Step 2 的 forbiddenImports 固化）。若迁移 platform 产生实际问题，
  允许回退为"platform 保留原调用并加入豁免登记"，但必须在 PR 中说明原因。
- 测试文件（`*_test.go`）不迁移，门禁也不扫描。
- `CodingKit/tools/**` 是低层 capability（独立模块），不在本次范围。

## Step 2：dependency governance 扩展

### 2.1 `config/dependency-governance.json`

1. `goPackageBoundaries` 新增 entry：

```json
{
  "path": "internal/gitx",
  "forbiddenImports": [
    "internal/bootstrap", "internal/cache", "internal/cli", "internal/cstyle",
    "internal/docsync", "internal/governance", "internal/kit", "internal/lifecycle",
    "internal/mcpcontrol", "internal/platform", "internal/pwshregex",
    "internal/registry", "internal/releasegate", "internal/report",
    "internal/repohealth", "internal/reuse", "internal/runner",
    "internal/tagpolicy", "internal/testengine"
  ]
}
```

2. 新增顶层节 `gitProcessBoundary`：

```json
{
  "ownerPackage": "internal/gitx",
  "scanRoots": ["cmd", "internal"],
  "allowedImporters": [
    "internal/cli", "internal/cstyle", "internal/docsync", "internal/governance",
    "internal/kit", "internal/lifecycle", "internal/platform", "internal/pwshregex",
    "internal/repohealth", "internal/tagpolicy"
  ]
}
```

### 2.2 `config/schemas/dependency-governance.schema.json`

为 `gitProcessBoundary` 增加 schema 定义（required: ownerPackage、scanRoots、
allowedImporters；不允许附加属性，与现有 schema 风格一致）。

### 2.3 `internal/governance/dependencies.go`

在 `RunDependencies`（现有 `addDependencyCheck` 链）新增两个 check：

1. **`git process ownership`**：遍历 `scanRoots` 下生产 `.go` 文件（跳过
   `_test.go`、跳过 ownerPackage 目录），用 `go/ast` 检查
   `exec.Command` / `exec.CommandContext` 调用中命令参数为字面量 `"git"` 的情形，
   命中即 error（报文件相对路径）。实现风格与 `checkGoPackageBoundaries` 一致
   （parser + WalkDir，此处需 full parse 而非 ImportsOnly）。
2. **`gitx importer allowlist`**：遍历 `scanRoots` 下生产 `.go` 文件
   （ImportsOnly 即可），import `github.com/JiaxI2/AiCoding/internal/gitx`
   而所在包不在 `allowedImporters` 内即 error。

策略节缺失时的行为：`gitProcessBoundary` 为空对象/缺失 → 该 check 报
policy missing error（与仓库"显式策略"风格一致），不静默跳过。

### 2.4 测试

- `internal/governance/dependencies_test.go`：为两个新 check 增加正/负用例
  （临时 fixture 目录中构造违规文件，断言 error 文案）。
- `internal/cli/dependency_governance_fixture_test.go`：按现有 fixture 模式补充
  CLI 层回归，确认 `governance dependencies --json` 输出包含两个新 check 名。

## Step 3：command catalog porcelain 禁用测试

`internal/cli/catalog_test.go` 新增 `TestCommandCatalogRejectsGitPorcelainVerbs`：

- 禁用集合（命令名与 alias 均检查）：
  `add, am, apply, bisect, blame, branch, checkout, cherry-pick, clone, commit,
  diff, fetch, init, log, merge, pull, push, rebase, remote, reset, restore,
  revert, show, stash, submodule, switch, worktree`；
- 断言 `Catalog()` 中每个 `CommandDescriptor.Name` 与 `Aliases` 不落入集合；
- 测试注释指向 GIT_REUSE_BOUNDARY.md §8/§9，说明 `status`/`tag`/`fresh-clone`
  为何不在集合内（同名不同义豁免）。

## Step 4：文档与登记

1. `docs/architecture/GIT_REUSE_BOUNDARY.md` 已随本计划提交（Status: Proposed）。
2. 运行 `bin/aicoding.exe docsync --json`；若 docs-sync policy / repository
   navigation 要求登记新文档，按门禁报错补登记（以门禁输出为准，不预猜）。
3. `CHANGELOG.md` 增加条目（版本只出现在 changelog/metadata，不进入任何 identity）。
4. 不修改已冻结架构文档的契约内容；如需交叉引用，只允许增加"详见
   GIT_REUSE_BOUNDARY.md"链接行。

## Step 5：门禁执行顺序

```powershell
go build ./...
go test ./internal/governance/... ./internal/cli/... ./internal/gitx/...
bin\aicoding.exe governance dependencies --json
bin\aicoding.exe docsync --json
bin\aicoding.exe test --profile Smoke --json
bin\aicoding.exe test --profile Full --json
```

全部 `ok=true`、退出码 0 后进入验收（见 [ACCEPTANCE_PLAN.md](ACCEPTANCE_PLAN.md)）。

## 非目标（明确不做）

- 不新增/改名任何 CLI 命令，不动 `internal/registry`、`internal/runner`、
  `internal/report` 的公开契约；
- 不在 run evidence 中加 HEAD OID 字段；
- 不实现跨进程 state CAS；
- 不迁移测试文件与 `CodingKit/tools/**` 中的 git 调用；
- 不为 gitx 增加 interface/mock 层。

## Rollback

删除 `docs/architecture/GIT_REUSE_BOUNDARY.md` 与 `docs/decisions/git-reuse-boundary/`，
还原 `config/dependency-governance.json`、schema、`internal/governance/dependencies.go`
的新增节与 check，还原 7 处调用点迁移与 catalog 测试；不触碰其他未提交改动。
