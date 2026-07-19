# ADR 0003: repo-context 领域（仓库上下文自动生成与保鲜）

## Status

Accepted。阶段 0–4 全部落地：扫描 → 生成 → commit 增量同步 → 聚合门禁。
实现在 `internal/repocontext`（领域）+ `internal/lifecycle` 的 `repo-context` adapter
+ `hook post-commit` + `doctor.repo-context`/`verify.repo-context` + 测试用例 `RC-001/RC-002`。

## TL;DR

把"Agent 理解本仓库需要的上下文"从**手工维护的目录级索引**升级为**从代码确定性生成、
随每次提交自动保鲜的受管资产**。它不是一套新框架，而是把三个已有基础能力
（内容快照 + 生命周期动词 + 报告契约）组合起来，只补了"扫描"和"生成"两段确定性纯函数。
内核六模块零改动，删除本领域不需要动任何内核代码。

## 1. 与设计哲学的对齐（Primitive First / Composition First）

本领域**复用**的基础能力，与它**新增**的最小逻辑，一一列清：

| 需要的能力 | 复用的已有 Primitive | 位置 | 本领域新增了什么 |
|---|---|---|---|
| 事实 → 稳定 digest | 内容快照原语 `registry.NewSnapshot` | `internal/registry` | 无（直接用） |
| 统一动词编排、有界执行 | lifecycle adapter + `runner.ExecutionPlan` | `internal/lifecycle` | 一行 descriptor + 一个 `run` 函数 |
| 可对账 JSON 证据 | `report.Result` / `AdapterResult` | `internal/report` | 无（直接用） |
| 提交触发 | hook 命令机制 | `internal/cli` `runHook` | 一个 `post-commit` 分支 + 薄壳 |
| 读取提交变更文件 | git 事实层 | `internal/gitx` | `CommitFiles` 一个薄函数 |
| 门禁登记 | 聚合器 + 唯一测试 Registry | `internal/repohealth`、`internal/testengine` | 两处 check + 两个 leaf 用例 |

**真正新增的领域逻辑只有四段确定性纯函数**，全部在 `internal/repocontext`，都不进内核：

1. `Scan` — 走仓库 → 归一化 `Facts`（`scan.go`）；
2. `render` — `Facts` → 每域一份 markdown（`generate.go`）；
3. `reconcile` — 期望产物 vs 现有 manifest → **只写内容变化的文件**（`lifecycle.go`）；
4. `affectedDomains` — 变更文件 → 受影响顶层域（`sync.go`）。

结论：新增能力 = 6 个复用 Primitive 的组合 + 4 段领域私有纯函数。没有新框架、没有新
控制面、没有新动词、没有新顶层命令——符合"优先组合、保持最小核心、扩展不改核心"。

## 2. 为什么是新领域而非现有领域的变体

对抗性追问：能不能用 kit / mcp / runtime-skill 表达？不能，两个字段本质不同：

| | kit / mcp / runtime-skill | repo-context |
|---|---|---|
| InputKind | 外部获取后登记的 registry/manifest | **从本仓库事实派生的扫描快照**（无 manifest 可写） |
| StateOwner | venv / plugin cache / junction | **自己生成的 markdown 文件 + 其 digest 清单** |

输入种类与状态所有权都无处安放，才新增第四个 adapter（扩展路径③，需 ADR）。
三条件（[架构手册](../architecture/ARCHITECTURE_HANDBOOK.md) §5.3）：现实问题 = 目录级索引
手工维护、随代码漂移、粒度粗；稳定变化点 = 代码演进本身；两个真实消费者 = AiCoding
自举 + 受管项目仓库（如 C99 kit 服务的 C 工程）。

## 3. 包与文件地图

| 文件 | 职责 | 关键导出 |
|---|---|---|
| `internal/repocontext/types.go` | 数据结构与 owned 根常量 | `Facts` `Manifest` `Report` |
| `internal/repocontext/scan.go` | 确定性扫描器（无 LLM、无外部进程） | `Scan(repo) (Facts, Snapshot, error)` |
| `internal/repocontext/generate.go` | `Facts` → markdown 渲染 + 内容 digest | `render`（包内） |
| `internal/repocontext/lifecycle.go` | 六动作 + `reconcile` + manifest 读写 | `Install/Update/Uninstall/Status/Doctor/Verify` |
| `internal/repocontext/sync.go` | 提交后增量同步（update 的内部步骤） | `Sync(repo, changedPaths, dryRun)` |
| `internal/lifecycle/repo_context.go` | lifecycle 请求 → 领域调用的翻译 | `runRepoContextAdapter` |
| `internal/lifecycle/catalog.go` | descriptor 一行登记 | — |
| `.githooks/post-commit` | 薄壳，触发 `hook post-commit` | — |

生成物 owned 根：`.aicoding/repo-context/`（已 gitignore，属受管可再生本地状态）。

## 4. 数据结构（确定性契约）

`Facts` —— 全部字段排序、相对路径、无时间戳，故 digest 跨机器跨时刻稳定：

```json
{
  "repo": "AiCoding",
  "languages": [{"language": "Go", "extension": ".go", "files": 128}],
  "toolchains": ["Git submodules", "Go modules", "Taskfile", "clang-format"],
  "domains": [{"path": "internal", "files": 96, "primaryLanguage": "Go"}]
}
```

`Manifest`（`.aicoding/repo-context/manifest.json`）—— 记录"从哪个事实生成、生成了哪些
文件、每个文件内容 digest 是什么"，是 owned-asset 纪律的唯一依据。**不含时间戳**，本身可对账：

```json
{
  "schemaVersion": 1,
  "factsDigest": "sha256:…",
  "files": [{"path": ".aicoding/repo-context/index.md", "digest": "sha256:…"}]
}
```

## 5. 动作语义表

`plan` 沿用契约：`lifecycle plan --action X --scope repo-context` = `X + dryRun`，与 apply 同路径。

每个动作**只做一件事**，因此只有真正需要"当前仓库事实"的动作才扫描：

| 动词 | 单一职责 | effect | 扫描次数 | 读/写什么 | 磁盘结果 |
|---|---|---|---|---|---|
| install | 生成 | write | 1 | 扫描 → 生成 | 写缺失/变化的文件 + manifest |
| update | 收敛 | write | 1 | 重扫描 → 收敛 | **只写内容变化的文件**，删不再需要的 |
| status | **新鲜度** | read | 1 | 扫描 + manifest | 不写；事实 digest vs manifest → fresh/drift/not-installed |
| uninstall | 移除 | write | **0** | 只读 manifest | 只删 digest 匹配的登记文件 + manifest |
| doctor | **完整性** | read | **0** | 读 manifest + owned 文件比对 | 不写；缺失/被篡改 → error |
| verify | **结构** | read | **0** | 读 manifest + stat owned 文件 | 不写；schema/文件缺失 → error |

**Primitive 性质（Do One Thing Well）**：

- **职责不重叠**：新鲜度是 `status` 唯一的职责；`doctor` 只查完整性、`verify` 只查结构，
  两者**不扫描仓库**——`doctor --all`/`verify --profile` 是高频门禁，不为一个由 post-commit
  hook 自愈的瞬时 drift 付全仓扫描代价（守卫 `TestRepoContextDoctorAndVerifyDoNotScanWhenNotInstalled`）。
- **每个动作至多扫描一次**：需要事实的动作（install/update/status）各扫一次并在 Report
  带回 `factsDigest`，adapter 复用它作 InputDigest；uninstall/doctor/verify **零扫描**，
  InputDigest 取自 manifest 记录（守卫 `TestRepoContextUninstallReadsManifestWithoutScanning`）。
- **单次遍历**：`Scan` 只做一次 `filepath.WalkDir` 并剪掉 `.git`/`node_modules`/`.aicoding`
  等目录；成本由 `BenchmarkScan` 独立度量。
- **确定性**：`Facts`/`Manifest`/生成文本全部排序、无时间戳、无绝对路径，同一仓库多次
  运行 digest 恒等。
- **最小写**：`reconcile` 只写内容 digest 变化的文件，未受影响的域字节与 mtime 不动
  （`BenchmarkReconcileNoOp` 度量收敛态成本）。

### 5.1 执行成本档案（Execution Cost First）

成本是先分析、后记录的设计输入，不是事后优化。已测量（本仓库，10 个域、1000+ 文件）：

| 操作 | 实测成本 | 说明 |
|---|---|---|
| `Scan`（全仓单次遍历） | **~2 ms/次** | 比 `git status` 还快；`BenchmarkScan` ~141µs/百文件 |
| `reconcile` 收敛态（无变化） | **~0.27 ms/次** | `BenchmarkReconcileNoOp` |
| doctor / verify / uninstall | ~0 扫描 | 只读 manifest + owned 文件 |

**三层成本可观测（Observable）**：命令级 `report.Result.elapsedMs` → 领域 Primitive 级
`lifecycle.adapters[].elapsedMs`（本轮新增，runner 已测量、adapter 现在带回）→ 热点级
`Benchmark*`（µs 精度）。任一层都不再是黑盒。

**基于证据的取舍**：`Scan` ~2ms 已足够快，`Sync`（post-commit）采用"全仓扫描 + 增量写"
而**不引入跨调用 facts 缓存**——缓存要付出持久化状态 + 失效逻辑的复杂度，为省 2ms 得不偿失，
违反"简单优于复杂"。增量只做在**写**这一层（`reconcile` 只写 diff、`Sync` 只涉受影响域）。
后续贡献者若要加缓存，必须先出示 `Scan` 成为真实瓶颈的测量证据，否则视为过度设计。

## 6. owned-asset 纪律（三条铁律）

`reconcile` 与 `removeOwned` 共同保证：

1. **只写变化**：期望内容 digest == manifest 记录时不重写该文件——改一个源文件，
   未受影响域的产物字节不变、mtime 不动。
2. **只删自己的**：uninstall / 清理只删 manifest 登记且**磁盘内容 digest 仍匹配**的文件；
   digest 不符（被手工改过）即拒删。
3. **永不碰未登记文件**：不在 manifest 里的文件（用户手写）一律不读不写不删。

回归保证：`TestUninstallRemovesOnlyOwnedArtifacts`、`TestSyncWritesOnlyChangedDomainAndReconvergesFresh`、
`TestDoctorDetectsTampering`。

## 7. 提交后增量同步（阶段 3）

`sync` 是 `update` 的内部增量步骤，**不是新动词**（动词表只增不改）。链路：

```text
git commit → .githooks/post-commit → aicoding hook post-commit
  → gitx.CommitFiles(HEAD) → repocontext.Sync(repo, changed, dryRun=false)
    → 未安装则静默空操作；否则重扫描 + reconcile（只写变化文件）
```

自愈：正常开发流每次提交都刷新，上下文长期保持 fresh。hook 保持薄壳、`|| true`
永不阻断提交。与 docsync 分工互补：docsync **拦**"人写文档没跟上代码"，
repo-context **让**"生成上下文自动跟上代码"。

## 8. 门禁接入与严重级（阶段 4）

聚合门禁只组合**零扫描**的完整性/结构检查，不为漂移付全仓扫描；漂移交给
`status`（显式）与 post-commit hook（自愈）。

| 门禁 | 职责 | 扫描 | 未安装 | 缺失/被篡改 | 漂移 |
|---|---|---|---|---|---|
| `doctor --all` → `doctor.repo-context` | 完整性 | 0 | 空操作 pass | **error** | 不检测（属 `status`） |
| `verify --profile` → `verify.repo-context` | 结构 | 0 | 空操作 pass | **error** | 不检测（属 `status`） |
| 测试 `RC-001`（结构）/ `RC-002`（计划） | 结构/计划 | 0/1 | pass | 结构破坏才 fail | — |
| `lifecycle status --scope repo-context` | 新鲜度 | 1 | not-installed | — | 报告 drift |

设计取舍：只有 owned 文件缺失或被外部篡改这种真实完整性破坏才 error（阻断）；漂移是
瞬时的、由 post-commit hook 自愈，不进聚合门禁的扫描路径。生成物已 gitignore，
fresh clone / CI 中一律"未安装 → 空操作"，绝不误伤。

## 9. 如何扩展（Convention over Configuration）

不需要改内核，也不需要加配置文件：

- **支持一门新语言**：在 `scan.go` 的 `languageByExt` 加一行 `".rs": "Rust"`。
- **识别一种新工具链**：在 `toolchainMarkers` 加一行 `"Cargo.toml": "Cargo"`。
- **产物加一段内容**：改 `generate.go` 的 `renderIndex`/`renderDomain`（纯函数）。
- **换更细的域切分**：改 `sortedDomains`（当前按顶层目录，约定优先）。

每一处都是领域私有纯函数的局部改动，digest 会因内容变化而变化，`status`/`doctor` 自动
对账，无需任何注册或配置。可选的 LLM 域发现若将来引入，只作显式可选步骤，产物仍走
同一 `reconcile` 与 digest 对账，可对账性不降级。

## 10. 如何删除（可删除性证明）

本领域可整体移除，内核零改动——删除以下、其余不动即可：

```text
internal/repocontext/                     整个包
internal/lifecycle/repo_context.go        adapter 翻译
internal/lifecycle/catalog.go             repo-context descriptor 一行
internal/lifecycle/types.go               ScopeRepoContext 常量
internal/cli/cli.go                       hook post-commit 分支
internal/cli/cli_ext.go                   --scope 的 repo-context 校验/守卫
internal/cli/catalog.go                   repo-context help 行
internal/gitx/git.go                      CommitFiles（若无其他消费者）
internal/repohealth/product.go            doctor/verify.repo-context 两处 check
internal/testengine/engine.go             RC-001 / RC-002 两个用例
.githooks/post-commit
```

`internal/registry`、`internal/runner`、`internal/report` 六模块无需任何修改——这是
runtime-skill 之后第二个"新领域进入不碰内核"的构造性先例。

## 11. 非目标（明确不做）

- 不并入 `aspenkit/aspens` 的 npm CLI（禁止第二控制面）；只做概念参照、Go 重实现。
- 默认零 LLM：扫描与生成全确定性，同一仓库两次扫描 digest 必须一致。
- 永不覆盖用户手写内容（§6 铁律 3）。
- 不加第四测试档、不改 JSON 报告契约、不新增知识进入点类型、不加新动词、不加顶层命令。

## Consequences

**收益**：仓库上下文随提交自动保鲜；每域小粒度 scoped context 按需加载、降 Token；
复用八动词与 JSON 契约，Agent 零新增学习成本；"已知的已知"上下文库持续变厚而调用成本
不升（四象限复利指标落地）。

**成本**：多一个领域模块要随语言/工具链场景演进；owned 纪律要求 manifest 与事实严格同步；
post-commit 给 hook 链加一个挂点（靠"薄壳、慢路径放 update"守住）。

**须警惕的重叠**：kit / mcp / repo-context 各自实现"期望态 → 写 owned → 收敛"的过程。
当前三者的 owned 形态不同（venv / junction / 生成文本），无实现重叠；**若将来再出现
第二个"生成文本文件"领域，应把 `reconcile`（digest-diff 写 + 按 owned digest 删）抽为
共享 Primitive**，而不是第三次复制——这是本 ADR 留给后续贡献者的显式路标。
