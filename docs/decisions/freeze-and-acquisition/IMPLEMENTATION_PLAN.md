# Implementation Plan: Contract Freeze And Acquisition Boundary

Plan Status: Approved

目标：落地 [FREEZE_AND_ACQUISITION_BOUNDARY.md](../../architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md)。
量级明显小于 Git 复用边界：一份新架构文档 + 两条 governance check + 契约测试，
零代码行为变化（冻结的是既成现实）。

## 现状事实（实现前基线，已核实于 v1.0.0 / origin/main）

- `config/kits/*.json` 与 `config/mcp/components/*.json` 当前不含任何 `://`；
- 含 git/github 字样 URL 的 config 域文件仅有：`.gitmodules`、
  `config/skill-sources.json`（获取登记面，合法）、`config/dependency-governance.json`
  （badge 权威 URL，非可克隆源）、`config/repository-navigation.json`（文档链接）、
  `config/schemas/cli-report.schema.json`（schema 标识）；
- 实现者必须先复核 `config/kit-registry.json`、`config/mcp-registry.json`、
  `config/codex-kit.json` 三个文件确认零 URL，并把审计结果写入交付说明。

## Step 1：架构文档

`docs/architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md` 已随本计划提交
（Status: Proposed，验收后翻转）。运行 docsync，按报错完成登记（如需要）。

## Step 2：governance 扩展

### 2.1 `config/dependency-governance.json`

新增顶层节（字段名可在实现中微调，语义不变）：

```json
"acquisitionBoundary": {
  "activationUrlFreeFiles": [
    "config/kit-registry.json",
    "config/kits",
    "config/mcp-registry.json",
    "config/mcp/components",
    "config/codex-kit.json"
  ],
  "cloneableSourcePattern": "(\\.git$)|(^https?://(www\\.)?(github\\.com|gitcode\\.[a-z]+)/[^/]+/[^/]+/?$)",
  "acquisitionRegistryFiles": [
    ".gitmodules",
    "config/skill-sources.json"
  ],
  "scanRoots": ["config"]
}
```

### 2.2 `config/schemas/dependency-governance.schema.json`

为 `acquisitionBoundary` 增加 schema 定义（required 四字段，
`additionalProperties: false`，风格与既有节一致）。

### 2.3 `internal/governance/dependencies.go`

新增两个 check，接入现有 `addDependencyCheck` 链：

1. **`activation manifests URL-free`**：遍历 `activationUrlFreeFiles`
   （目录项展开为 `*.json`），解码 JSON 后递归检查所有字符串值，
   含 `://` 即 error（报文件与 JSON path）。
2. **`cloneable sources registry`**：扫描 `scanRoots` 下全部 `.json` 与仓库根
   `.gitmodules`，对字符串值匹配 `cloneableSourcePattern`；命中且文件不在
   `acquisitionRegistryFiles` 内即 error。实现前先用该 pattern 对当前仓库
   跑一次语料审计：若 badge/文档 URL 意外命中，收紧 pattern 而不是扩大
   allowlist（allowlist 只留两个获取登记面）。

策略节缺失时报 policy missing error，不静默跳过（与 gitProcessBoundary 一致）。

### 2.4 测试

- `internal/governance/dependencies_test.go`：两个新 check 的正/负用例
  （fixture 中构造违规 manifest 与越界 clone URL）；
- `internal/cli/dependency_governance_fixture_test.go`：按既有模式补充 CLI 层
  回归，确认两个新 check 名出现在 JSON 报告。

## Step 3：冻结声明的交叉引用

在下列**非冻结**文档各加一行指向新边界文档（存在即可，不改契约内容）：

- `docs/COMMANDS.md`（JSON 契约段落）；
- 测试文档（三档语义处，具体文件以 docsync 语义绑定为准）；
- `docs/architecture/ARCHITECTURE_HANDBOOK.md` §8 文档地图新增一行。

已冻结架构文档不修改。

## Step 4：门禁执行顺序

```powershell
go build ./...
go test ./internal/governance/... ./internal/cli/...
bin\aicoding.exe governance dependencies --json
bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Smoke --json
bin\aicoding.exe test --profile Full --json
```

## 非目标（明确不做）

- 不实现 uv、wheel 缓存或任何下载加速代码（§3.4 出口，凭测量进入）；
- 不新增 CLI 命令，不改 Taskfile（§2.4 为声明性冻结，无门禁）；
- 不修改任何 schema 的必填项或字段语义（本计划只新增 governance 节）；
- 不做断网自动化测试（离线激活为验收手工抽查项）；
- 不触碰 internal/registry、internal/runner、internal/report、internal/lifecycle、
  internal/cli catalog 的公开契约。

## 分支说明

上一轮的 `codex/aicoding-architecture` 分支已并入 main。本计划的实现应从最新
`origin/main` 拉新分支执行；本决策目录与新架构文档以未跟踪文件形式存在于
worktree，切换分支后一并纳入首个提交。

## Rollback

删除新架构文档与本决策目录，还原 dependency-governance.json、schema 与
dependencies.go 的新增节与 check，还原交叉引用行；不触碰其他未提交改动。
