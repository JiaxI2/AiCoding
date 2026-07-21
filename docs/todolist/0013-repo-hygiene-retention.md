# TODO 0013: 仓库卫生（保留策略 + 统一清理面，扩展现有 cache 域）

Status: Planned
Verify: bin/aicoding.exe cache status --json 列出全部生成物类别及大小；cache clean --scope test-results 后保留最近 N 份且失败报告不被删

> 实测欠账（2026-07-20，主仓库 F:\Study\AI\AiCoding）：
> `test-results/` **14MB / 31 个目录**，每次 test 新建时间戳目录，**无任何保留策略**；
> `.git/` 81MB，其中 validation reports 已积累 9 份且只增不减；
> `cache clean` 目前只管 `.aicoding/cache/fast-path` 一个目录。
> 功能在长，垃圾也在长——这正是"每层都要贯彻架构思想"漏掉的运维层。

## 设计原则

**不新增 clean 命令域** —— `cache status|clean` 已存在且语义正确（Do One Thing：
本地生成物的观测与清理）。把它从"只管 fast-path"扩展为**全部本地生成物的唯一清理面**。
Convention over Configuration：保留策略有约定默认值，可配置但不必配置。

## 实现计划

1. **生成物注册表**（约定写死在 `internal/cache`，不做配置文件——类别是代码事实）：

   | scope | 路径 | 默认保留策略 | 永不自动删 |
   |---|---|---|---|
   | `fast-path` | `.aicoding/cache/fast-path` | 现状不变 | — |
   | `test-results` | `test-results/aicoding-global-test-*` | **保最近 5 份 + 全部含 FAIL 的** | 最新一份 |
   | `validation-reports` | `<common-dir>/aicoding/validation/reports/*` | 保有 Receipt 引用的；孤儿（Receipt 已被 clean）可删 | 被 alias 引用的 |
   | `work-state` | `.aicoding/state/work/*` | 只列出大小，**默认不清**（审计轨迹） | attempts.jsonl |

2. **`cache status --json` 扩展**：逐 scope 输出 `{path, entryCount, sizeBytes, policy}`
   + 总计。单次扫描每个根目录，不递归无关路径（宪法 §3）。
3. **`cache clean [--scope S] [--keep N] [--dry-run] --json`**：
   - 无 `--scope` = 全部可清 scope 按各自默认策略；
   - `--dry-run` 列出将删清单与释放字节数，零落盘；
   - **失败证据永不自动删**（含 FAIL 的 test-results、审计轨迹）——清理不得毁证；
   - validation-reports 的孤儿判定复用既有 store 读取路径，不重实现完整性逻辑。
4. **doctor 接入**：`doctor --all` 增加一条 bloat 检查（Warn 级）：任一 scope 超过
   阈值（test-results >20 份 或 >50MB）时提示 `cache clean`。**只提示，不自动清**。
5. **testengine 写入侧配合**：test 运行结束后（仅成功时）顺带执行本 scope 的
   保留策略（keep-last-5）——写入方负责自己的溢出，用户无感。失败运行不触发清理。
6. 文档：COMMANDS.md 更新 cache 两条；CHANGELOG。

## 明确不做

- 不做后台守护/定时清理（无守护进程原则；清理由命令或写入侧触发）。
- 不清 `.git` 对象库（`git gc` 归 git 自己，doctor 最多提示一行）。
- 不做跨仓库聚合清理（每仓自治）。
- 不把 bin/ dist/ 纳入（已是 generatedDirectories，clean 语义归 bootstrap/export 域）。

## 自测（可信任方式）

```powershell
go test ./internal/cache/... ./internal/testengine/...
# 造 8 份假 test-results（其中 1 份含 FAIL）：
bin\aicoding.exe cache status --json                     # 列出 8 份 + 各 scope 大小
bin\aicoding.exe cache clean --scope test-results --dry-run --json   # 清单=3 份（8-5），FAIL 那份不在清单
bin\aicoding.exe cache clean --scope test-results --json
# 断言：剩 5 份 + FAIL 份保留；最新一份必在
bin\aicoding.exe test --profile Smoke --json             # 跑真实一轮 → 写入侧自动保留生效
bin\aicoding.exe doctor --all --json                     # bloat 检查出现（低于阈值时为 pass）
bin\aicoding.exe validation status --json                # 清理后 Receipt 完整性不受影响
```

通过判据：dry-run 清单与实删一致；FAIL 证据与被引用 reports 永不被删（负例断言）；
`validation check` 在清理后仍能命中既有 Receipt；单次 `cache status` 不做全仓扫描
（只 stat 注册的 4 个根）。
