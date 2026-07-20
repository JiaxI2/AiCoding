# ADR 0007: Validation Evidence 内容身份与 Receipt

PrimitiveReview: required

## Status

Accepted for phase 2。内容身份、不可变 Receipt 与逐用例审计已稳定；第二期增加 Context Gate、
pre-push exact-tree 门禁、commit alias，并在定期 seed/audit CI 接线后默认 `--reuse auto`。
Plan Mode 与 Profile 继承继续明确排除。

## 1. Decision

新增 `internal/validationevidence`，把一次测试结论绑定到 Git Tree OID 与验证语义，而不是
commit SHA、时间戳、mtime 或 CLI 二进制字节。它只依赖 `internal/gitx`；Test Registry、
Typed Command Catalog、profile 配置和选项由唯一 `testengine`/CLI 上层计算摘要后注入。

```text
Git subject + semantic digests
  -> validation identity
  -> exact Receipt path
  -> integrity-checked PASS evidence
  -> profile/commit alias
  -> remote-ref Context Gate
```

Receipt 位于 `<git-common-dir>/aicoding/validation/receipts/<profile>/<identity>.json`；查询先按
identity 对唯一文件执行 `os.Stat`，不建 index、不扫描目录。报告位于
`reports/<identity>/`，Receipt schema v2 保存三个报告文件的 SHA-256，以及当前 profile 选中
TestCase 排序 `(id,status)` 的 `resultsDigest`。写入采用同目录临时文件、
`fsync`、`os.Rename`；相同 identity 且 `resultsDigest` 相同的并发写入保留第一份完整报告，
状态摘要不同则拒绝冒用既有 Receipt，Receipt 保持不可变。

第二期 alias 位于 `aliases/<profile>/<commit-oid>`，文件内容仅一行 validation identity。
profile 分区避免同一 commit 的 Smoke/Full/Release 互相覆盖；alias 是可更新投影，Receipt 仍
不可变。pre-push 从 Git stdin 解析四字段 update，按真实 `local_oid` 解 tree，不使用当前
HEAD；policy 未匹配的 feature ref 明确旁路，`main` 与 release tag 要求 Release alias。

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
- CLI 暴露 status/check/list/clean 与 hook 薄入口，不拥有 evidence/push policy 规则。

生产代码为 `model.go`、`subject.go`、`fingerprint.go`、`store.go`、`checker.go`、`policy.go`
六个文件。公开操作为首期七个，加 `LoadPolicy`、`BindCommit`、`GatePush` 三个第二期操作；
push tree 继续复用既有通用 `gitx.TreeOID(repo, rev)`，不复制 `RefTreeOID`。

## 4. Correctness gates

- commit message amend 后 Tree 不变，identity 与 Receipt 继续命中；
- 两个 linked worktree 通过 common-dir 共享同一 Receipt；
- 不同仓库即使 Tree 相同也因 repositoryID 隔离；
- untracked、tracked 变化和 report/Receipt 篡改均 fail-closed；
- 留存报告复用与 `--verify-reuse` 均重新计算逐用例状态摘要，`PASS → WARN` 即使整体仍归一为
  PASS 也会 fail-closed；
- FAIL 无法调用 `Put` 生成 Receipt；
- 并发 `Put` 在 Windows `os.Rename` 语义下保持可读取；
- pre-push 以 stdin `local_oid` 而非 HEAD 判定；main 非快进/删除、tag 删除、缺 alias 均阻断；
- hook 薄壳不执行测试、构建或 Git 工作区写操作；
- dependency governance 强制本包不 import 业务领域包，Git 只经 gitx。

## 5. Rollback

行为回滚优先显式传入 `--reuse off` 并移除 `.githooks/pre-push`/policy 接线；这会恢复完整
TestCase 执行路径而不删除任何证明。Git common-dir 下的 Receipt/alias 是可再生证据，
Receipt 可由 `validation clean` 删除，不属于版本化工作区。Plan Mode 无需参与回滚。

## §12 Checklist 自评

**架构**

- 单一职责：只回答“给定 Git 内容、验证语义与 push context，是否存在 policy 要求的完整 PASS Receipt”。
- 可继续拆分：生产代码按 model/subject/fingerprint/store/checker/policy 六个私有职责拆分；不建 inheritance 或 report authority。
- 可复用：HEAD/INDEX 捕获、内容寻址 store 和 checker 均不认识具体 TestCase。
- 无重复实现：Git 复用 `gitx`，执行复用唯一 `testengine`，JSON 继续复用既有报告。
- 新 Primitive 必要性：既有 testengine 只有当次执行报告，没有跨 commit 元数据变化仍稳定的内容身份证据。

**性能**

- Fast Path：validation check 使用 exact path `os.Stat`；pre-push 使用 exact alias + Receipt 路径。
- 无关扫描：check 不遍历仓库、不扫描 Receipt 目录、不递归查询 submodule。
- 重复 IO/计算：主体只调用一次 Git status；config 只读取显式路径；toolchain 命中不 spawn version。
- Agent/工具调用：零 Agent；Context Gate 自身零网络，Git 仅为 status/tree/ancestry 且全部经 gitx。
- 最小 Context：指纹只保留 digest，不保留环境变量、命令输出或 report log 原文。
- 实测预算：第 0 期 `status` 中位数 186.153ms、`HEAD^{tree}` 69.480ms；HEAD check warm-cache SLA 300ms，见 `docs/operations/VALIDATION_EVIDENCE_BUDGET.md`。

**质量**

- 确定性：identity/Receipt 无时间戳、commit SHA、绝对路径或执行耗时；相同输入字节恒等。
- 接口稳定：首期七个、第二期只增加 policy/alias/gate 三个操作；错误使用稳定 code + requiredAction。
- 最小输入输出：Fingerprint 只接收主体与上层语义摘要；Check 返回命中、原因和耗时。
- 独立测试：临时 Git repo、linked worktree、篡改、并发、path escape、toolchain-cache 与非 HEAD push context 测试均在包内。
- 自由组合：testengine、CLI 与未来外部消费者通过值对象组合；没有 Plan Mode 耦合。
