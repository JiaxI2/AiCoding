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

相对优化前冷中位数 97.620s，改善 81.4%；相对优化前热中位数 90.715s，改善 80.2%。
Full 中 EXP-001/FRESH-001 均为 profile SKIP，EXP-002/FRESH-003 静态替代均 PASS。

真实 `task release` 墙钟 74.867s、引擎 74.597s，58 PASS / 0 FAIL / 0 WARN / 0 SKIP。
EXP-001 真实 ZIP 耗时 758ms；FRESH-001 耗时 55,761ms，步骤包含成功的
`git.clone`、`git.submodule: submodules verified`、overlay、Go build 与 release verify。

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

阶段二已取得约 80% 总收益，继续引入 DAG、共享快照、handler 双路径或结果缓存缺乏与风险
相称的测量证据。阶段七同样暂不实施：本次热报告日志为 165 个文件、约 260KB，未表现为
关键路径；日志策略和确定性摘要应作为独立契约变更另行评估。
