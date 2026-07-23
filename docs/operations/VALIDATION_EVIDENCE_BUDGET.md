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

## 9–11. 已完成验证阶段的结论与归档

- 第三期八场景基线确认：Receipt 命中是复用视图，冷执行与变更 miss 必须按同一 Go cache 口径比较。
- 第四期节点级 Receipt 确认：docs-only 可复用不受影响的 Go/lifecycle 节点，篡改审计保持 fail-closed。
- 第五期 Hermetic 物化确认：Release leaf 保留真 clone 对照能力，同时以显式 repo root 完成无 `.git` 源码树验证。
- 原 §9–11 的身份、逐场景数据、计时与失败现场均逐字保存在
  [过程性样本归档](evidence/validation-evidence-phases-3-5-archive.md)。

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
