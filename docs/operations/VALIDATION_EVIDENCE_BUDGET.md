# Validation Evidence 性能预算

## 1. 基线身份

- 测量日期：2026-07-20
- Git SHA：`520e14b84805260ebca03b0eb438b08ffb243552`
- Git：`2.48.1.windows.1`
- Go：`go1.26.4 windows/amd64`
- 平台：Windows，PowerShell
- 仓库：`AiCoding-main` 独立 worktree，工作区无修改

## 2. 测量方法

在承诺 `validation check` 的 SLA 前，按同一 worktree 连续执行五次下列 Git/Go 调用；
每项取中位数。命令输出重定向为空，只计进程启动、Git/Go 执行和返回的墙钟时间。

```powershell
foreach ($i in 1..5) {
  Measure-Command { git rev-parse "HEAD^{tree}" } | % TotalMilliseconds
  Measure-Command { git status --porcelain --ignore-submodules=none } | % TotalMilliseconds
  Measure-Command { git status --porcelain } | % TotalMilliseconds
  Measure-Command { git write-tree } | % TotalMilliseconds
  Measure-Command { go version } | % TotalMilliseconds
}
```

`git write-tree` 会向 Git object database 写入 tree 对象，但不修改工作区、index 或 HEAD；
该调用不是纯查询。

## 3. 第 0 期实测

| 调用 | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | 中位数 |
|---|---:|---:|---:|---:|---:|---:|
| `git rev-parse HEAD^{tree}` | 115.017 ms | 67.559 ms | 57.142 ms | 149.873 ms | 69.480 ms | **69.480 ms** |
| `git status --porcelain --ignore-submodules=none` | 288.374 ms | 239.578 ms | 186.153 ms | 178.582 ms | 168.828 ms | **186.153 ms** |
| `git status --porcelain` | 170.187 ms | 181.531 ms | 176.504 ms | 344.951 ms | 211.314 ms | **181.531 ms** |
| `git write-tree` | 88.307 ms | 83.929 ms | 61.246 ms | 64.144 ms | 75.871 ms | **75.871 ms** |
| `go version` | 164.539 ms | 61.553 ms | 65.648 ms | 65.009 ms | 66.120 ms | **65.648 ms** |

带子模块脏检测的 `git status` 中位数为 `186.153 ms`，低于方案规定的 `400 ms` 停止线，
因此 Validation Evidence 第一期可以继续。HEAD 检查的两个主要 Git 调用中位数合计为
`255.633 ms`；独立执行 `git submodule status --recursive` 不进入检查路径。

## 4. SLA 与实现预算

第一期采用以下 warm-cache SLA：

```text
validation check --target HEAD --json 的 5 次墙钟中位数 <= 300 ms
```

预算只允许一次 `git status --porcelain --ignore-submodules=none`、一次
`git rev-parse <rev>^{tree}`、一次 toolchain cache 读取和一次内容寻址 Receipt `os.Stat`。
不得递归查询 submodule、不得扫描 Receipt 目录、不得逐文件哈希、不得哈希 CLI 二进制。

`--target INDEX` 额外执行一次 `git write-tree`，其 SLA 在第一期实现完成后单独实测回填，
不沿用 HEAD 目标的数字。

## 5. 回填规则

第一期完成后必须在同一环境重建 `bin/aicoding.exe`，分别对 Receipt miss/hit 执行五次
`validation check`，把真实样本和中位数回填本文件。若 HEAD hit 中位数超过 `300 ms`，
先用调用计数与 profile 定位超额来源；在新证据获得评审前，不提高 SLA，也不启用默认复用。

## 6. 第一期实现回填

- 实现测量 Git SHA：`c03076f`；
- 二进制：按该 SHA 重新执行 `go build -o bin\aicoding.exe ./cmd/aicoding`；
- 目标：干净 linked worktree 的 Release profile，warm toolchain cache；
- 计时边界：PowerShell 从启动 `aicoding.exe` 到进程退出的外部墙钟，命令输出重定向为空；
- HEAD 路径只执行一次 porcelain-v2 status 和一次 `HEAD^{tree}`，两者并发；常规 `.git`/
  `commondir` 直接解析，异常布局才回退 `git rev-parse --git-common-dir`。

| 场景 | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | 中位数 | 结论 |
|---|---:|---:|---:|---:|---:|---:|---|
| HEAD Receipt miss | 262.284 ms | 362.701 ms | 228.590 ms | 266.570 ms | 249.938 ms | **262.284 ms** | PASS |
| HEAD Receipt hit | 304.146 ms | 250.578 ms | 333.665 ms | 229.967 ms | 285.355 ms | **285.355 ms** | PASS |
| INDEX Receipt hit | 293.992 ms | 300.286 ms | 385.933 ms | 268.734 ms | 300.930 ms | **300.286 ms** | 参考值 |

首版串行实现的 warm miss 中位数为 `386.390 ms`，超额主要来自额外的 common-dir Git
进程（独立中位数 `72.890 ms`）以及 status/TreeOID 串行等待。移除额外进程并并发两个必需
只读 probe 后，HEAD miss/hit 均通过原定 `300 ms` SLA；没有提高 SLA，也没有删除
untracked 或 submodule dirty 检查。INDEX 会执行 `git write-tree`，本期只记录参考值，不建立
HEAD SLA 的错误等价关系。

## 7. 第一期功能验收

2026-07-20 在同一干净 SHA 上执行：

| 验收路径 | 结果 | 墙钟/报告耗时 |
|---|---|---:|
| `test --profile Release --reuse off` | executed，58/58 PASS，生成可复用 Receipt（新 worktree 冷样本） | 171.026 s |
| `validation check --profile Release --target HEAD` | `VALIDATION_RECEIPT_HIT` | 报告 225 ms |
| `test --profile Release --reuse auto` | reused，58/58 PASS，同一 Receipt | 外部 392.763 ms |
| Smoke seed | executed，40/40 PASS，可复用 | 24.162 s |
| Smoke `--reuse auto --force` | executed，40/40 PASS，同一 Receipt | 23.089 s |
| Smoke `--reuse auto --verify-reuse` | executed，40/40 PASS，`VALIDATION_RECEIPT_HIT` | 22.893 s |

`171.026 s` 是新 worktree 冷构建缓存与当时主机负载下的验收种子，不是可比性能基线，也不
表示 Validation Evidence 导致 Release 回归；该次报告中 FRESH-001/GO-002/GO-001 分别为
`101.860 s`、`41.259 s`、`38.380 s`，均体现冷缓存/负载放大。诚实的可比头条应使用功能加入
前已记录的 Release 墙钟 `74.867 s`（约 76s）与复用墙钟 `392.763 ms`，即 **74.867 s →
392.763 ms，下降 99.5%**；`171.026 s → 392.763 ms` 只描述这次冷种子与命中的同批验收，
不得作为长期加速口径。

Release Receipt ID 为
`sha256:3e978d3eea94bcd083dfde8800f6505f03d3684a193be56ff28a90fd195c009b`；Smoke Receipt
ID 为 `sha256:10849976d0e6940fca527fa57fe51a54384bce5b49e28d36bab0c3323e9b603b`。
第一期验收时默认值仍是 `--reuse off`；该期不启用 Hook、不切换默认复用，也不进入 profile
继承。

## 8. 第二期 Context Gate 与默认复用回填

- 实现测量 Git SHA：`0de65d7d01e31c94cb4095ed055cc6a3ba89fe8b`；
- 场景：已有 `main` 的快进推送，pre-push stdin 中 `local_oid=HEAD`、`remote_oid=HEAD^`；
- 门禁：`refs/heads/main` 要求 Release Receipt，按输入的 `local_oid` 查 commit alias，并校验
  alias、Receipt、tree 和逐用例 `resultsDigest`；
- 计时边界：`gateMs` 是 Go 门禁核心，`commandMs` 是 CLI 报告耗时，`outerMs` 是 PowerShell
  从启动预编译 `bin/aicoding.exe` 到退出的墙钟；Hook 不运行构建或测试。

| pre-push 样本 | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | 中位数 | 最大值 |
|---|---:|---:|---:|---:|---:|---:|---:|
| Context Gate | 149 ms | 81 ms | 69 ms | 106 ms | 86 ms | **86 ms** | 149 ms |
| CLI 报告 | 217 ms | 173 ms | 131 ms | 213 ms | 158 ms | **173 ms** | 217 ms |
| 端到端墙钟 | 263.742 ms | 213.784 ms | 166.064 ms | 244.867 ms | 195.634 ms | **213.784 ms** | 263.742 ms |

第二期采用以下 warm-cache 预算：

```text
已有 main 的单 ref 快进 pre-push，5 次端到端墙钟中位数 <= 300 ms
```

该路径 5/5 允许推送，实测中位数通过预算。策略还会拒绝受保护 ref 的删除、非快进、缺失
commit alias 或 Receipt；未匹配的 feature ref 明确旁路。门禁以 stdin 中实际 `local_oid` 为
事实源，不用当前 `HEAD` 代替待推送对象。

同一 Receipt 的显式 `--reuse auto` 五次端到端样本为
`376.042/406.937/397.799/404.775/370.312 ms`，中位数 **397.799 ms**，与第一期可比命中
样本 `392.763 ms` 同量级。该数字只证明 opt-in 复用性能；由于当时只有 workflow 接线、没有
远端连续绿灯，评审后默认值已退回 `--reuse off`。只有 main 的远端 release-gate 连续 3 次
seed/audit 成功并在独立提交引用 run URL 后才允许晋级。第二期冷种子为 58/58 PASS、
`172.911 s`；其中
FRESH-001/GO-002/GO-001 分别为 `98.295 s`、`44.541 s`、`18.931 s`，仍属于冷缓存与主机
负载样本，不替代第 7 节的 `74.867 s` 历史可比基线。

第二期测量 Receipt ID 为
`sha256:8b394dfe1fba048f01853550d6644776ca0620d7bd22e0500290e246d1413400`，逐用例摘要为
`sha256:4adcd702591e337e6d21c25ff9f3d056ca5e863805e18f5e0b79b6b2aa0528f0`。Profile 继承和
Plan Mode 继续保持在范围外。

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

## 12. 已解决的历史限制：跨 shell 的 toolchain 身份偏严

`toolchainDigest.v1` 把 Go/Git 可执行文件的绝对路径与 mtime 一并计入身份。同一台机器上，
Git Bash 的 `/usr/bin/git` 与 PowerShell 解析到的 `cmd/git.exe` 会产生不同 toolchain 身份，
即使 Tree 等其余身份字段完全相同，两边的 Receipt 仍不能互相复用。

以上原限制描述保留为历史。TODO 0032 以 `toolchainDigest.v2` 解决：Receipt 身份只绑定显式
域/算法版本、规范化后的 `go version` / `git --version` 与平台/架构；解析后的绝对路径、
size、mtime 只作为本地 probe cache 键。任一键变化均重探，但版本语义未变时 digest 不变；
probe 失败或版本输出不可解析仍 fail-closed，损坏 cache 不提供身份、只能由真实 probe 重建。

该变化只提升 warm reuse 与跨 shell 的命中率，不降低 `--reuse off` 的冷运行成本。v1→v2
换域后第一次 Full/Release 必然全冷，这是预期身份迁移，不是性能或正确性回归。

本次换域后的最终 Full 与 Release 固定以 `--reuse off` 对同一 staged tree 真跑，summary
分别保存于 `test-results/0032-final-full/summary.json` 与
`test-results/0032-final-release/summary.json`；这两次只作为 v2 首轮冷验收，不计入 §13
要求的远端 main release-gate 绿灯。

## 13. main release-gate 复用晋级计数

v1 历史证据（不计入 v2 晋级）：1/3:
https://github.com/JiaxI2/AiCoding/actions/runs/29900035150 @ 9890b667bfdc54ef5fafe49d27c736210ad13732 PASS

该次正式 main 运行的 release-gate 依次完成 `--reuse off` 冷种子与 `--verify-reuse`
全量审计，并上传 `release-gate-evidence`。它证明 v1 身份方案，但 ADR 0007 §5 规定 fingerprint
算法契约换域必须重置计数，因此不进入 v2 的晋级轨道。

v2 当前计数：**3/3**:
https://github.com/JiaxI2/AiCoding/actions/runs/29916523297 @ 41eefac7a67ac1473a5b9cf7cfc6548ca7372027 PASS
https://github.com/JiaxI2/AiCoding/actions/runs/29921228586 @ 48a355c32941bca2a01eb1f95e3c78c6af3f8090 PASS
https://github.com/JiaxI2/AiCoding/actions/runs/29922476097 @ 44a99d13b9d9b84181318b7423a22595939438cd PASS

该次正式 main workflow dispatch 的 release-gate 先以 `--reuse off` 完成冷种子，再以
`--verify-reuse` 完成全量审计并上传 `release-gate-evidence`。两次均针对 Tree
`529ef271c491c717202a19b10fa7127a36d83c73`，复用审计返回
`VALIDATION_RECEIPT_HIT`，且与冷种子共用 Receipt
`sha256:f19b6a93718fad2fbc02f6ac1893bea5db0798779e41cc568b6033baa106d1be`。
两次 profile 均为 `71 total / 69 pass / 0 fail / 2 warn / 0 skip`；`ENV-004` 的 CI
环境未安装 Task 与 `FRESH-004` 的无 Release fresh-clone 基线均保持 advisory，workflow
及 release-gate job 的最终 conclusion 均为 `success`。

第二次正式 workflow dispatch 针对另一棵 main Tree
`7afa0605ec602f040408f942de43ad6fad013979`。其 `--reuse off` 冷种子与
`--verify-reuse` 审计同为 `71 total / 69 pass / 0 fail / 2 warn / 0 skip`，审计返回
`VALIDATION_RECEIPT_HIT`，两段共用 Receipt
`sha256:3da37e2e61244cc8fabccec306b4aab3e96d2e7e8b0bca4b00d2a4dfb7398e14`；workflow 与
release-gate job 均为 `success`。该 run 使用修复前的 workflow，因此仍真实记录
`ENV-004` 与 `FRESH-004` 两项 advisory。

第三次正式 workflow dispatch 针对新 main Tree
`878cae97795ac7e62b21f4deee215d76d1ffb420`。release-gate 先通过官方
`go-task/setup-task` 的完整 commit SHA 安装 Task `3.52.0`，再执行 `--reuse off` 冷种子与
`--verify-reuse` 审计；两段均为 `71 total / 70 pass / 0 fail / 1 warn / 0 skip`，且
`ENV-004` 的原始 stdout 均为 `3.52.0`、状态均为 PASS。唯一 WARN 是无 Release
fresh-clone baseline 的设计内 `FRESH-004`。审计返回 `VALIDATION_RECEIPT_HIT`，两段共用
Receipt `sha256:6b40f63f0a70ce7d2b8cdaa2b8eb99c81b06e570a013627e7a615cab85e69047`；四个 workflow jobs
及总运行均为 `success`。

至此三次 v2 证据来自三棵不同的 main Tree，且每次均完成冷种子与全量审计，晋级前置计数
已满足。本节证据落账时 `--reuse` 默认值仍保持 `off`，并要求默认值翻转另开独立评审提交；
该后续评审现已由 ADR 0014 与本文件 §15 完成。

## 14. 配置与 Receipt 存储裁决：当前不引入数据库

配置权威保持 Git + JSON + checked-in schema；Validation Receipt store 保持 Git common-dir 下的
内容寻址文件存储。本轮只闭合配置/schema 清单，不建立数据库、索引服务或集中式配置加载器。

满足以下任一条件时才重新打开数据库评估，且评估必须先保存同机、warm filesystem cache 的
原始测量结果：

1. Receipt 数量不超过 10,000 时，连续两次测量中各真跑 5 次 `validation list --json`，
   其 p95 均超过 2 秒或超过最近已记录基线的 3 倍；
2. 连续三次 main `--verify-reuse` 审计中，Receipt 枚举/读取/校验的可分离 p95 开销超过
   30 秒，或超过对应 Release 总墙钟的 10%；
3. 出现一个已批准产品需求，需要在单次查询中聚合至少 3 个独立仓库的 Receipt 证据。

阈值未触发前，数据库议题关闭；单纯文件数量增长、冷 Release 时长或缺少跨仓查询设想均不
构成重开依据。

## 15. `--reuse auto` 默认值晋级

生效日期：2026-07-23。ADR 0014 依据 §13 的 `toolchainDigest.v2` 远端 3/3 证据，把
`test --profile ...` 的默认值从 `off` 晋级为 `auto`。`toolchainDigest.v2` 继续只把
规范化 Go/Git 版本与平台/架构纳入 Receipt 语义；可执行文件路径、size 与 mtime 只触发本机
probe，不因路径本身制造语义 miss。

`auto` 只在干净 HEAD 或 index-only 主体上，对 repository、Tree、profile、plan、engine、
config、toolchain、options 的完整 identity 以及 Receipt/报告/逐 leaf 状态摘要全部校验通过
后命中。普通 identity 变化是 miss 并真跑；精确路径存在但 store 损坏、fingerprint 非法或
读取失败则非零 fail-closed，不执行后覆盖损坏 Receipt。

首次本地晋级实测针对最终实现的 staged Tree
`68a2fc3a23bead9b3db0c9ff21e0abe7da943e23`：

| Release | 结论 | execution mode | cache hit ratio | 命令墙钟 |
|---|---|---|---:|---:|
| 默认 auto 冷跑 | 73 PASS / 0 FAIL / 0 WARN / 0 SKIP | executed | 0 | 217,369ms |
| 同 Tree 默认 auto 热跑 | 同上 | reused | 1 | 344ms |

复用报告保留冷证据自己的 `summary.duration_ms`；热跑墙钟使用 CLI `elapsedMs`。两次共用
Receipt `sha256:c4fc31279a83a7c2b9e82b63b4139e603020d96ab429a5ad3dbb0492b081a95a`。
原始 summary 位于 `test-results/0042-final-release-cold/summary.json` 与
`test-results/0042-final-release-warm/summary.json`，八项负例和 CI 命令逐字证明见 TODO 0042。

以下任一事实出现一次即立即回滚默认值为 `off`：Tree 已变化却命中旧 Receipt；任意真实失败
被 Receipt 掩盖为绿。不设观察窗口，不以“后续再看”延迟回滚。CLI release-gate 的显式
`--reuse off` seed 与 `--verify-reuse` audit 在晋级前后逐字不变。
