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
