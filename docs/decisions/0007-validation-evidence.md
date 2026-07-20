# ADR 0007: Validation Evidence 内容身份与 Receipt

PrimitiveReview: required

## Status

Accepted for phase 1。只落地 Git Tree 内容身份、语义指纹、不可变 Receipt、完整性检查与
显式复用判定；Hook、Plan Mode、Profile 继承均不属于本 ADR 的当前范围。

## 1. Decision

新增 `internal/validationevidence`，把一次测试结论绑定到 Git Tree OID 与验证语义，而不是
commit SHA、时间戳、mtime 或 CLI 二进制字节。它只依赖 `internal/gitx`；Test Registry、
Typed Command Catalog、profile 配置和选项由唯一 `testengine`/CLI 上层计算摘要后注入。

```text
Git subject + semantic digests
  -> validation identity
  -> exact Receipt path
  -> integrity-checked PASS evidence
```

Receipt 位于 `<git-common-dir>/aicoding/validation/receipts/<profile>/<identity>.json`；查询先按
identity 对唯一文件执行 `os.Stat`，不建 index、不扫描目录。报告位于
`reports/<identity>/`，Receipt 保存三个报告文件的 SHA-256。写入采用同目录临时文件、
`fsync`、`os.Rename`；相同 identity 的并发写入保留第一份完整报告，后续写入复用它，
Receipt 保持不可变。

## 2. Identity boundary

Identity 由 repositoryID、Tree OID、profile、validation plan、engine semantics、相关配置、
toolchain 和 options 的 digest 组成。repositoryID 是 Git common-dir 规范绝对路径的 digest，
Receipt 不泄露真实路径。engine semantic digest 由上层绑定 Catalog/Registry/实现版本，明确
不哈希带 buildvcs 的 CLI 二进制。

`HEAD` 只在工作区完全干净时可复用；`INDEX` 允许 index-only staged 状态。tracked 工作区
修改、非忽略 untracked、unmerged 或 dirty submodule 均使主体不可复用。被 gitignore 忽略的
文件不进入 Tree OID，Receipt 必须固定声明：

```json
{"scope":{"ignoredFilesOutOfScope":true}}
```

因此 Receipt 证明 Git 追踪内容的验证结论，不证明 ignored local state。

## 3. Composition and ownership

- `gitx` 是唯一 Git 进程与磁盘布局知识边界；本包既不直接启动 Git，也不解析 `.git`、
  `gitdir:` 或 `commondir` 格式。
- `validationevidence` 不执行 TestCase、不产生第二套 runner。
- `testengine` 决定 PASS/FAIL、Required/Warning/Skip policy，并决定是否生成 Receipt。
- JSON report 仍由现有 `testengine` 产生；Receipt 只保留完整性绑定的最小视图。
- CLI 只暴露 status/check/list/clean 薄入口，不拥有 evidence 规则。

生产代码固定为 `model.go`、`subject.go`、`fingerprint.go`、`store.go`、`checker.go` 五个文件。
公开操作只有 `Open`、`Capture`、`Fingerprint`、`Put`、`Check`、`List`、`Clean` 七个。

## 4. Correctness gates

- commit message amend 后 Tree 不变，identity 与 Receipt 继续命中；
- 两个 linked worktree 通过 common-dir 共享同一 Receipt；
- 不同仓库即使 Tree 相同也因 repositoryID 隔离；
- untracked、tracked 变化和 report/Receipt 篡改均 fail-closed；
- FAIL 无法调用 `Put` 生成 Receipt；
- 并发 `Put` 在 Windows `os.Rename` 语义下保持可读取；
- dependency governance 强制本包不 import 业务领域包，Git 只经 gitx。

## 5. Rollback

回滚时删除 `internal/validationevidence`、本 ADR 和 dependency-governance 节点即可；第一期
CLI 默认 `--reuse off`，因此移除该包不会改变原有 TestCase 执行语义。Git common-dir 下的
本地 Receipt 是可再生证据，可由 `validation clean` 删除，不属于版本化工作区。

## §12 Checklist 自评

**架构**

- 单一职责：只回答“给定 Git 内容与验证语义，是否存在完整 PASS Receipt”。
- 可继续拆分：生产代码已按 model/subject/fingerprint/store/checker 五个私有职责拆分；不再建 policy、inheritance 或 report authority。
- 可复用：HEAD/INDEX 捕获、内容寻址 store 和 checker 均不认识具体 TestCase。
- 无重复实现：Git 复用 `gitx`，执行复用唯一 `testengine`，JSON 继续复用既有报告。
- 新 Primitive 必要性：既有 testengine 只有当次执行报告，没有跨 commit 元数据变化仍稳定的内容身份证据。

**性能**

- Fast Path：exact path `os.Stat` miss；toolchain 以 executable path/size/mtime 命中本地小缓存。
- 无关扫描：check 不遍历仓库、不扫描 Receipt 目录、不递归查询 submodule。
- 重复 IO/计算：主体只调用一次 Git status；config 只读取显式路径；toolchain 命中不 spawn version。
- Agent/工具调用：零 Agent、零网络；Git 仅为 status/tree，且全部经 gitx。
- 最小 Context：指纹只保留 digest，不保留环境变量、命令输出或 report log 原文。
- 实测预算：第 0 期 `status` 中位数 186.153ms、`HEAD^{tree}` 69.480ms；HEAD check warm-cache SLA 300ms，见 `docs/operations/VALIDATION_EVIDENCE_BUDGET.md`。

**质量**

- 确定性：identity/Receipt 无时间戳、commit SHA、绝对路径或执行耗时；相同输入字节恒等。
- 接口稳定：首期仅七个操作；错误使用稳定 code + requiredAction。
- 最小输入输出：Fingerprint 只接收主体与上层语义摘要；Check 返回命中、原因和耗时。
- 独立测试：临时 Git repo、linked worktree、篡改、并发、path escape 和 toolchain-cache 测试均在包内。
- 自由组合：testengine、CLI 与未来外部消费者通过值对象组合；没有 Plan Mode 耦合。
