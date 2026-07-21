# Full 性能基线

## 1. 基线身份

- 测量日期：2026-07-20
- Git SHA：`afa4dc866ff6689faee975fc8190772af6671cd3`
- Task：`3.50.0`
- Go：`go1.26.4 windows/amd64`
- CPU：32 logical processors
- 存储：NVMe Samsung SSD 990 PRO 2TB（SSD）
- Microsoft Defender 实时扫描：开启

## 2. 测量口径

墙钟时间是在 PowerShell 中以 `Stopwatch` 包围 `task full` 得到；测试引擎时间来自同一次
`test-results/aicoding-global-test-*/results.json` 的 `summary.duration_ms`。

“冷”严格采用性能计划给定的产品口径：每次运行前执行
`bin\\aicoding.exe cache clean --json`，但不执行 `go clean -cache -testcache`，因此它表示
AiCoding cache 冷、Go 工具链和操作系统缓存可能已热。“热”不执行任何 cache clean。
所有六次运行均为 55 PASS / 0 FAIL / 0 WARN / 2 SKIP。

因此本文的“冷”只能解释为 **AiCoding-cache-cold**，不是 Go build cache、Go test cache 或
操作系统文件缓存全冷；表格中的“冷缓存 Full”均服从这个较窄定义。

性能计划输入中的“`task full ≈ 142s`”是改造前的先验观察，没有配套原始报告、cache 状态、
CLI rebuild 状态或系统负载记录，无法与本节口径做同条件复现。它比本次冷中位数 97.620s
高 44.380s；现有证据不能把差值归因到某一个因素，因此后续比例只使用本节六次可追溯实测，
不把 142s 合并进基线统计。

## 3. Full 总耗时

| 场景 | Run | 墙钟时间（s） | 引擎时间（s） | 报告目录 |
|---|---:|---:|---:|---|
| 冷 | 1 | 113.978 | 112.503 | `aicoding-global-test-20260720-120353` |
| 冷 | 2 | 97.620 | 90.126 | `aicoding-global-test-20260720-120601` |
| 冷 | 3 | 82.635 | 81.220 | `aicoding-global-test-20260720-120741` |
| **冷中位数** | | **97.620** | **90.126** | |
| 热 | 1 | 100.472 | 99.076 | `aicoding-global-test-20260720-120918` |
| 热 | 2 | 74.151 | 72.756 | `aicoding-global-test-20260720-121106` |
| 热 | 3 | 90.715 | 89.191 | `aicoding-global-test-20260720-121228` |
| **热中位数** | | **90.715** | **89.191** | |

## 4. 成本模型硬门禁

性能计划规定：若 `FRESH-001 + BOOT-001 + BOOT-002` 不到总耗时 40%，停止后续优化。
六次实测占测试引擎时间 66.3%–81.8%；冷中位样本为 73.7%，热中位样本为 80.0%。
成本模型成立，可以进入移除重复构建及 Full/Release 边界调整。

| 场景 | Run | FRESH+BOOT（s） | 引擎占比 |
|---|---:|---:|---:|
| 冷 | 1 | 82.944 | 73.7% |
| 冷 | 2 | 59.796 | 66.3% |
| 冷 | 3 | 63.060 | 77.6% |
| 热 | 1 | 80.995 | 81.8% |
| 热 | 2 | 54.724 | 75.2% |
| 热 | 3 | 71.360 | 80.0% |

## 5. 最慢 15 个用例

取热测第 3 次正式报告，按 `duration_ms` 降序：

| 排名 | ID | Title | 耗时（ms） | Severity |
|---:|---|---|---:|---|
| 1 | FRESH-001 | fresh-clone Full | 69,245 | WARN |
| 2 | GO-001 | 全仓 Go 单元测试 | 4,249 | REQUIRED |
| 3 | GO-002 | Go race 检查 | 4,159 | WARN |
| 4 | C99-007 | C Kit 快速验证 | 1,589 | REQUIRED |
| 5 | BOOT-001 | bootstrap 构建 Go CLI | 1,305 | REQUIRED |
| 6 | GO-003 | go vet 基础检查 | 1,085 | WARN |
| 7 | HEALTH-001 | doctor performance probes | 824 | REQUIRED |
| 8 | BOOT-002 | CLI bootstrap 基础可用 | 810 | REQUIRED |
| 9 | EXP-001 | export zip | 766 | REQUIRED |
| 10 | GO-004 | CLI 并发只读调用 | 704 | REQUIRED |
| 11 | GIT-005 | governance lint | 689 | REQUIRED |
| 12 | DOC-001 | DocSync CI | 452 | REQUIRED |
| 13 | GIT-008 | repository layout | 254 | REQUIRED |
| 14 | DOC-002 | DocSync all | 227 | WARN |
| 15 | LIFE-007 | kit lifecycle 结构验证 | 212 | REQUIRED |

## 6. 基线结论

- Full 的主导项是 FRESH-001，而不是 AiCoding cache 命中率。
- BOOT-001/002 单次成本不大，但与 Task `ensure-bin` 一起造成重复构建。
- 第一、二阶段完成后必须用相同口径重新测量；第三至第六阶段不得仅凭目标值启动。

## 7. 阶段一实测：移除重复构建

阶段一完成后连续两次运行公开 `task full` 依赖链；两次均为
55 PASS / 0 FAIL / 0 WARN / 2 SKIP，运行前后 `bin/aicoding.exe` mtime 均未变化，证明
`ensure-bin` checksum 命中且未重复构建。

| Run | 墙钟时间（s） | 引擎时间（s） | Binary mtime changed |
|---:|---:|---:|---|
| 1 | 80.904 | 80.646 | false |
| 2 | 97.420 | 97.160 | false |
| **两次中点** | **89.162** | **88.903** | |

与优化前热测中位数 90.715s 相比，墙钟改善 1.553s（约 1.7%）。首次执行曾因 Task 默认
生成的 `.task/` 未登记而在 GIT-008 失败；该次不计入性能结果。修复方式是把 `.task/`
登记为 Git 忽略的 transient/runtime-state，并保持根目录 layout 门禁通过。

## 8. 阶段二实测：Full / Release 边界

阶段二完成后冷/热各运行三次。冷测仍按第 2 节清理 AiCoding cache；首个冷测同时包含阶段二
源码触发的 CLI rebuild。六次 Full 全部为 55 PASS / 0 FAIL / 0 WARN / 3 SKIP。

| 场景 | Run | 墙钟时间（s） | 引擎时间（s） | Binary mtime changed |
|---|---:|---:|---:|---|
| 冷 | 1 | 59.055 | 50.632 | true |
| 冷 | 2 | 18.010 | 17.731 | false |
| 冷 | 3 | 18.151 | 17.917 | false |
| **冷中位数** | | **18.151** | **17.917** | |
| 热 | 1 | 17.924 | 17.688 | false |
| 热 | 2 | 18.378 | 18.116 | false |
| 热 | 3 | 17.858 | 17.600 | false |
| **热中位数** | | **17.924** | **17.688** | |

相对优化前冷中位数 97.620s，阶段二后 Full 墙钟点估计下降 81.4%；相对优化前热中位数
90.715s，点估计下降 80.2%。这两个数字描述的是 **Full 默认成本边界的墙钟变化**，不表示
底层工作普遍加速了 80%：阶段一真正删除重复 CLI build 的实测贡献只有约 1.7%；热态剩余
约 78.5 个百分点主要来自阶段二把真实 ZIP 和 FRESH-001 移出交互式 Full。

这次边界重划删除的仍是真冗余：旧 FRESH-001 在临时 clone 中再次执行 `go test ./...`，与
同一 Full 的 GO-001 重复。Full 中 EXP-001/FRESH-001 均为 profile SKIP，EXP-002/FRESH-003
静态替代均 PASS；真实 clean-clone Full 改由每周/手动 CI 执行，Release 继续保留真实 ZIP 与
hermetic clone。因此总重复工作下降，但不应表述为底层操作本身快了 81%。

纠正提交前在本机实际执行一次 `fresh-clone --profile Full --json`，总耗时 123.657s；
`git.clone`、递归 submodule status、overlay、CLI build 与临时 clone 内 `go test ./...` 全部
通过。该数字代表独立 hermetic CI 成本，不计入日常 Full 中位数。

基线冷三次 113.978/97.620/82.635s 的离散度约为中位数的 ±16%，明显大于阶段二稳态
17.858–18.378s。用基线极值和阶段二稳态极值交叉计算，观测到的保守下降范围约为
78%–84%；81.4%/80.2% 只保留为中位数点估计。样本数仅三次，这个范围不是统计置信区间。

真实 `task release` 墙钟 74.867s、引擎 74.597s，58 PASS / 0 FAIL / 0 WARN / 0 SKIP。
EXP-001 真实 ZIP 耗时 758ms；FRESH-001 耗时 55,761ms，步骤包含成功的
`git.clone`、`git.submodule: submodules verified`、overlay、Go build 与 release verify。

> 上述 FRESH-001 是 TODO 0018 之前的历史基线。当前 Release 中同一 ID 已改为 REQUIRED 的
> Git-object 物化验证，不再 clone；真实 clone 只由显式 `fresh-clone` 与每周/手动
> `clean-clone-full` 保留。新旧证据用 `sourceMode=materialized|cloned` 区分，历史数字不改写。

### 8.1 阶段二后最慢 15 个 Full 用例

取热测第 3 次报告：

| 排名 | ID | Title | 耗时（ms） |
|---:|---|---|---:|
| 1 | GO-001 | 全仓 Go 单元测试 | 4,434 |
| 2 | GO-002 | Go race 检查 | 4,227 |
| 3 | C99-007 | C Kit 快速验证 | 1,477 |
| 4 | GO-003 | go vet 基础检查 | 1,035 |
| 5 | HEALTH-001 | doctor performance probes | 813 |
| 6 | GO-004 | CLI 并发只读调用 | 693 |
| 7 | GIT-005 | governance lint | 653 |
| 8 | DOC-001 | DocSync CI | 399 |
| 9 | EXP-002 | export manifest 静态验证 | 329 |
| 10 | BOOT-002 | CLI bootstrap 基础可用 | 263 |
| 11 | LIFE-007 | kit lifecycle 结构验证 | 229 |
| 12 | DOC-002 | DocSync all | 228 |
| 13 | GIT-008 | repository layout | 219 |
| 14 | BOOT-003 | bootstrap 前置条件完整 | 200 |
| 15 | GIT-001 | 工作区状态 | 172 |

### 8.2 阶段三至六裁决

| 阶段 | 实测证据 | 决定 |
|---|---|---|
| 三：资源感知 DAG | Full 热中位数 17.924s，低于 40s 启动阈值 | 不实施 |
| 四：RepoSnapshot | repo/governance/DocSync 相关用例合计约 2.779s，非关键路径 | 不新增包 |
| 五：进程内 handler | 27 个 AiCoding 子进程合计 6.203s，但包含必须执行的领域工作，纯启动收益未分离 | 不改变 CLI 契约覆盖 |
| 六：精确缓存 | GO-001/002/003 合计 9.696s；缓存可减少时间但带来静默假 PASS 风险 | 不实施 |

阶段二将交互式 Full 的实测墙钟降低约 78%–84%，其中主要收益来自成本边界重划；继续引入
DAG、共享快照、handler 双路径或结果缓存缺乏与风险相称的测量证据。阶段七同样暂不实施：
本次热报告日志为 165 个文件、约 260KB，未表现为关键路径；日志策略和确定性摘要应作为
独立契约变更另行评估。
