# VALIDATION_EVIDENCE_BUDGET §9–11 过程性样本归档

本文件逐字保存 `docs/operations/VALIDATION_EVIDENCE_BUDGET.md` 原 §9–11 的过程性样本；源文档提交历史与本归档提交可共同用于追溯搬运前后的证据。

## 9. 第三期验证观测八场景基线

### 9.1 身份与口径

- 测量日期：2026-07-21；
- 原始八场景实现候选提交：`13eddf172e90c1e92ff181107b716cf20f026959`，Tree：
  `cda190f003db5c54236eaa750be72463fa24cda4`，父提交：`9cf1260`；
- 可比性修正的三行基准提交：`58f184f32d2892b8b3eaf4859ec7c8e12debc4a5`，Tree：
  `34c7e8b103d948d2751c585e399ef832f52f060e`；docs-only/one-go-file 派生 Tree 分别为
  `404ea154908e97d9f9ff6a7d772cea7fb412a7bb`、`3ae5e4b2a64830e811c61ffcc8bb350014a2326e`；
- 环境：Windows 11 10.0.26200，Intel Core i9-14900HX（24C/32T），
  `go1.26.5 windows/amd64`，Git `2.48.1.windows.1`；
- worktree：两批测量均使用干净 detached 候选树；测量前递归初始化三个 pinned submodule；
  预构建对应候选树的 `bin/aicoding.exe`；
- 计时：`outer_ms` 是 PowerShell 从启动预构建 CLI 到退出的墙钟；`report_ms` 是 test engine
  新鲜执行报告耗时。复用报告保留种子执行的 summary，所以 warm 场景的比较头条必须看
  `outer_ms`，不能把留存的 `report_ms` 当成本次命中耗时；
- “cold/warm”只描述 Receipt：cold/变更场景完整执行，warm 使用 `--reuse auto` 命中。
  warm-full/warm-release 是 `cache_hit_ratio=1` 的复用视图，本次没有执行任何用例；其中
  `duration_ms`、`slowest_cases` 和 PASS 数逐项继承种子报告，不能作为第二次独立测量；
- 原始批次没有把 Go build/test cache 状态固定为可比前提。修正批次固定
  `GOCACHE=C:\Users\24322\AppData\Local\go-build`，先执行一次 Full 预热（60 PASS，
  不入表），再不清缓存、不中插其他 Go 命令，按 cold-full → docs-only → one-go-file 串行
  测量。未重测行保留原始值，并明确标成未归一旧样本或 Receipt 复用视图；
- one-go/docs-only/lifecycle-change 分别使用只改一个 Go 文件注释、只改 `README.md`、只给
  `config/kit-registry.json` 增加合法 JSON 空白的独立提交。三者都不改变待测实现行为。

初始化 submodule 前的首次 Full 因 `fresh-clone skills submodule is empty` 出现 7 FAIL，属于
测量环境不完整，未进入基线；递归初始化后独立 `go test ./...` 通过，以下八项全部成功。
原始完整本机报告位于 ignored `test-results/0016-baseline-20260721/`；修正三行与预热报告位于
ignored `test-results/0016-baseline-corrected-20260721/`。

### 9.2 八场景实测

| 场景 | 命令/变化 | 模式 | Receipt ratio | Go cache 状态 | outer_ms | report_ms | 结论与最慢项 |
|---|---|---|---:|---|---:|---:|---|
| cold-full | `test Full --reuse off` | executed | 0 | 同一 GOCACHE；Full 预热后的第 1 个正式样本 | **150837** | 149675 | 60 PASS；GO-001 76685 / GO-002 37671 ms |
| warm-full | 同 Tree，`test Full --reuse auto` | reused | 1 | 不执行 Go；复用原始 cold-full 种子 | **424** | 留存 189733 | Receipt 视图；`duration_ms`/Top5 继承原种子，非第二次执行 |
| docs-only | 单 README 变更，`test Full --reuse auto` | executed miss | 0 | 同一 GOCACHE；连续第 2 个正式样本 | **192278** | 191053 | GO-002 80023 / GO-001 78200 ms |
| one-go-file | 单 Go 注释，`test Full --reuse auto` | executed miss | 0 | 同一 GOCACHE；连续第 3 个正式样本 | **200072** | 198870 | GO-002 85494 / GO-001 78988 ms |
| lifecycle-change | kit registry JSON 空白，`test Full --reuse auto` | executed miss | 0 | 原始旧样本；Go cache 未归一，不与前三行横比 | **428270** | 427039 | GO-002 199697 / GO-001 199165 ms |
| cold-release | `test Release --reuse off` | executed | 0 | 原始旧样本；Go cache 未归一，仅保留验收事实 | **490689** | 489468 | 63 PASS；GO-001 195696 / GO-002 195158 / FRESH-001 66956 ms |
| warm-release | 同 Tree，`test Release --reuse auto` | reused | 1 | 不执行 Go；复用 cold-release 种子 | **453** | 留存 489468 | Receipt 视图；`duration_ms`/Top5 继承种子，非第二次执行 |
| fresh-clone-release | `fresh-clone --profile Release` | executed | n/a | 原始独立 clone 样本；只用于步骤分解 | **56633** | CLI 56496 | 5/5 steps PASS，均有 `elapsed_ms` |

原始报告的继承关系已直接比对：warm-full 与其原 cold-full 种子的 `duration_ms` 均为
`189733`，Top5 数组逐项同为 GO-002 82974 / GO-001 77072 / GO-006 12265 /
GO-005 5506 / C99-007 2474 ms；warm-release 与 cold-release 的 `duration_ms` 均为
`489468`，Top5 逐项同为 GO-001 195696 / GO-002 195158 / FRESH-001 66956 /
GO-006 13096 / GO-005 4805 ms。这种完全一致来自 Receipt 视图复制，不是两次独立执行。

修正批次的 cold-full/docs-only/one-go-file 都是 60 PASS、0 FAIL；带 timing 的用例均满足
`queue_ms + setup_ms + execute_ms + persist_ms == duration_ms`，三行不一致数均为 **0**。
三个正式样本的 summary 均为 `cache_hit_ratio: 0`；两个变更 Tree 带
`VALIDATION_RECEIPT_MISS: no reusable Receipt exists`。原始 lifecycle-change/cold-release 报告的
同一守恒检查也为 0。warm 两行的 ratio 为 `1`，但仅证明 Receipt 命中开销，不能计作执行样本。

### 9.3 explain 精确性、确定性与 check 快路径

在已有 Full Receipt 后，README-only 提交执行两次
`validation explain --profile Full --target HEAD --json`。两次剔除顶层 `elapsedMs` 后的
`data` 字节一致，结果为：

```text
decision=miss
changed=subjectTreeOID
oldTree=3c50214e6737caa4293d7c152e7ec7be927ebb7f
newTree=a0ea8a41cedec902ab525302363d29f8b2f70832
unchanged=repositoryID,profile,validationPlanDigest,engineSemanticDigest,configDigest,toolchainDigest,optionsDigest
referenceSelection=latest same-profile Receipt by receipt-file mtime; diagnostic only
```

lifecycle-change 的同类 explain 精确给出
`changed=[subjectTreeOID,configDigest]`，其余 6 个字段不变。基准 Release Tree 的 hit explain
返回 exact identity `sha256:36cbddf4a76df271b6d3039ed4258d7ffcbba773eefae3185a22b63cd0ce9f2f`
且 `changed=[]`。

为验证 explain 不改变 `validation check` 的 O(1) 快路径，在 Release Receipt hit 上于 explain
前后各执行 5 次预构建 CLI 外部墙钟：

| 阶段 | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | 中位数 |
|---|---:|---:|---:|---:|---:|---:|
| explain 前 | 292 ms | 288 ms | 288 ms | 278 ms | 278 ms | **288 ms** |
| explain 后 | 281 ms | 290 ms | 282 ms | 275 ms | 285 ms | **282 ms** |

两组 10/10 均为 `VALIDATION_RECEIPT_HIT`，中位数变化 `-6 ms`；Receipt 总数保持
`16 → 16`，Git porcelain 前后相同。两组都通过既有 300 ms HEAD hit SLA，未发现 explain
引入扫描或写入污染。

### 9.4 FRESH-001 与 standalone fresh-clone 分解

| 上下文 | clone | submodule | overlay | build | verify | steps 合计 | CLI/用例总耗时 |
|---|---:|---:|---:|---:|---:|---:|---:|
| cold-release 内 FRESH-001 | 51116 | 1635 | 643 | 8135 | 3300 | 64829 | 66956 ms |
| standalone fresh-clone-release | 40991 | 1611 | 648 | 7866 | 3321 | 54437 | 56496 ms |

standalone 样本中 clone 占 CLI 耗时约 **72.6%**；cold-release 内样本中 clone 占 FRESH CLI
耗时约 **76.4%**。这组数据只为 0018 的物化方案提供立项输入，本项不实现物化。

### 9.5 后续立项输入

- 0017：同一 Go cache 状态下，docs-only 只改变 `subjectTreeOID`，仍付出 `191053 ms` 整份
  Full 报告执行；可比 cold-full 为 `149675 ms`。原先 `425435 ms` 的 docs-only 数字来自
  未归一 Go cache 状态，不再作为立项依据。字段级 miss 仍说明无关领域被整份失效，因而值得
  继续设计分域 Receipt，但不预设节点边界；`424 ms` warm-full 只代表旧 Receipt 命中开销。
- 0018：fresh-clone 的最大成本是 clone/submodule 获取，且 standalone clone 单步为
  `40991 ms`；应以本表的 step 数据评估物化收益，不把整个 FRESH-001 总时长归因于 build。
- 本节不改变默认 `--reuse off`，也不实现 0017/0018；后续架构修改必须用同一八场景重测。

## 10. 第四期节点级 Receipt 回填（TODO 0017）

2026-07-21 在递归初始化 submodule 的 detached 探针 worktree 中，用同一预构建 CLI 和同一
Git common-dir Receipt store 串行测量。每个变更都形成独立 clean Tree；原始报告保存在本地
ignored `test-results/0017-node-receipts-20260721/`。`cache_hit_ratio` 分母为 Full 选中的
63 个用例，不含 3 个 profile SKIP。

| 场景 | 命令/变化 | 模式 | 节点命中 | engine_ms | outer_ms | 结论 |
|---|---|---|---:|---:|---:|---|
| seed-off | `test Full --reuse off` | executed | 0/63 | **243663** | **244897** | 63 PASS；发布整树与 PASS 节点 Receipt |
| docs-only-auto | 仅改根 README，`--reuse auto` | executed（部分复用） | **15/63** | **11654** | **12853** | `go` 6 + `lifecycle-readonly` 9 命中；其余真实执行，63 PASS |
| go-only-auto | 仅改一个 `.go` 注释，`--reuse auto` | executed（部分复用） | **3/63** | **438047** | 未留存 | 仅 `docsync` 3 命中；`go` 等域真实执行，63 PASS |
| verify-positive | 同 Go Tree，`--verify-reuse` | executed/audit | 0/63 | **346824** | **347688** | 63 PASS，`VALIDATION_RECEIPT_HIT` |
| verify-tamper | 把当前 Go 节点 `resultsDigest` 改为全零后审计 | executed/audit | 0/63 | **345023** | **345358** | 63 个用例通过，新增 `EVIDENCE-NODE-GO` FAIL；`VALIDATION_REUSE_AUDIT_MISMATCH` |

docs-only 外部墙钟相对 seed 从 `244897 ms` 降到 `12853 ms`，下降 **94.8%**；它证明无关的
Go/lifecycle 用例没有重跑。Go-only 样本处于不同系统负载，耗时不作为性能回归基线，只用于
验证反向失效：GO-001…006 均真实执行，而 DOC-001/002/004 保持节点命中。正向审计后，手工
篡改前后 Receipt 文件 SHA-256 分别为 `C6D392…7344` / `B13861…053D`；审计按预期失败后从
逐字节备份恢复为 `C6D392…7344`，随后 `validation check --profile Full --target HEAD` 在
`230 ms` 内以 `VALIDATION_RECEIPT_HIT` 命中整树 Receipt。

单元/集成测试另外断言：可复用主体只调用一次 Tree listing；dirty 主体调用零次；未标注用例
归入 `repo`；失败节点不发布 Receipt；节点 Receipt/报告篡改 fail-closed。默认值仍为
`--reuse off`，第 5 节的三次 main 远端 seed/audit 晋级纪律不变。

## 11. 第五期 Hermetic 物化回填（TODO 0018）

### 11.1 真 clone 对照样本

2026-07-22 在本分支未提交实现 overlay 上显式执行
`fresh-clone --profile Release --json`，证明公共真 clone 能力未删除。结果
`sourceMode=cloned`、总耗时 **69871 ms**，全部步骤通过并成功释放临时目录：

| source tree | clone | submodule verify | overlay | build | release verify | baseline | temp release |
|---|---:|---:|---:|---:|---:|---:|---:|
| `cca56633041147fe31a2a32dcc3ccf582954cdf3` | **52429** | 1755 | 688 | 8689 | 3826 | 84 | 2321 ms |

该样本需要递归获取 3 个 gitlink 工作树；clone 单步占命令总耗时约 **75.0%**。成功后 Git
common-dir 只写一行当前真 clone Tree 作为 FRESH-004 的可变提示基线；它不是 Receipt、alias
或授权凭证。每周/手动 `clean-clone-full` workflow 与显式三 profile 命令保持不变。

### 11.2 Release 物化验收样本

修复两条真实集成负例后，对 staged Tree
`35fcc02fd045557aaf889b51d902021fd96f7fa0` 执行
`test --profile Release --reuse off`。报告 **67/67 PASS、0 WARN、0 SKIP**，引擎总耗时
`323100 ms`；整体数字受本轮 GO-001/GO-002 主机负载影响，不拿来宣称物化收益。FRESH-001
单 leaf 为 **16184 ms**，报告 `sourceMode=materialized`、复合身份
`sha256:afb0b97761fc8c8e30bcd2baf6df617f7da369eeffda69355f2847c5e36fa2f5`：

| source tree | files | recursive gitlinks | materialize | build | release verify | temp release | FRESH-001 |
|---|---:|---:|---:|---:|---:|---:|---:|
| `35fcc02fd045557aaf889b51d902021fd96f7fa0` | 1809 | 3 | **4610** | 6697 | 3831 | 953 | **16184 ms** |

命令与步骤名均无 clone，`keptTemp=false`；manifest 位于源码树外，临时目录由统一 ledger
成功释放。相对第 9.4 节同为 Release 内 leaf 的 `66956 ms`，节省 `50772 ms`、下降
**75.8%**；相对本节当前真 clone 对照 `69871 ms`，下降 **76.8%**。这两个百分比只比较
FRESH leaf，不把本轮完整 Release 的 Go 负载混入结论。

### 11.3 fail-closed 负例

- 第一轮真实仓库集成暴露 Windows 系统 bsdtar 3.8.4 对中文目录返回
  `Invalid empty pathname`；实现改为 Go 标准库读取 `git archive` tar 流，并以中文路径夹具锁定
  回归，同时拒绝 `../` 越界 entry。
- 第二轮物化已成功，但无 `.git` 的源码树使未显式定根的 `release verify` 失败；失败现场保留后
  直接重放 `release verify --repo-root <materialized-root>` 通过，最终 leaf 固化该显式参数。
- 原始失败与最终报告保存在本地 ignored
  `test-results/0018-materialization-20260722/`；失败不会发布成功 Receipt。

<!-- VERBATIM ARCHIVE END -->
