# TODO 0024: 标准化收尾（临时资源回收 + 预算/权限释放 + session 边界）

Status: Planned
Verify: bin/aicoding.exe cache status --json 含 temp scope 且能回收残留；失败 fresh-clone 的临时目录被登记而非无主泄漏

> 来源：owner 提问"测试架构有没有完善的收尾功能"。
> **实测答案：没有，而且已经漏了 79 个目录 / 27 MB。**

## 一、实测：漏在哪里（2026-07-21）

```text
%TEMP%\aicoding-*                79 个目录 / 27 MB
  ├─ fresh-clone-*               60 个   ← 主要泄漏源
  ├─ smoke-*                      3 个
  ├─ v090-pid / v090-foc          4 个
  └─ 其它（zip/exe/plan/mingw）    5 个
```

**根因（`internal/kit/freshclone.go:52-58`）：**

```go
defer func() {
    if report.OK && !keepTemp {
        _ = os.RemoveAll(tempRoot)   // 只有成功才清
    } else {
        report.KeptTemp = true       // 失败 → 永久保留，且无人回收
    }
}()
```

**失败保留现场用于诊断是对的；没有任何回收机制是错的。**
60 个 fresh-clone 目录 = 历次失败/中断的累积，每个含完整 clone（含 18 MB 子模块的部分内容）。

其余三处生成物已被 0013 治理（test-results 保留 5 份、receipts/reports 内容寻址），
**唯独 `%TEMP%` 这一片是无主地带**——因为它在仓库外，`cache clean` 看不见。

## 二、设计原则：收尾是 Primitive，不是新流程

> **不新增"清理流程"，把 temp 纳入既有 `cache` 域的第五个 scope。**

0013 已经确立了保留策略的三条红线，本项完全沿用：
失败证据不自动删、被引用的不删、清理由命令或写入侧触发（无守护进程）。

## 三、实现计划

### A. 临时资源登记（把无主变有主）

`%TEMP%` 里的目录之所以无主，是因为**没有清单**。加一个索引，
不是数据库，是一个追加式 JSONL（与 `attempts.jsonl` 同纪律）：

```text
<git-common-dir>/aicoding/temp-ledger.jsonl
{"path":"...\\aicoding-fresh-clone-20260721-...","kind":"fresh-clone",
 "createdAt":"...","repoRoot":"...","outcome":"failed","sizeBytes":453120}
```

- 创建临时目录时**追加一行**（`internal/platform` 加一个 `TempDir(kind)` 帮助函数，
  统一命名 + 登记；`freshclone.go` 与其它三处改用它）；
- 成功清理后追加 `{"path":...,"outcome":"released"}`；
- **登记是追加不可变的**，回收动作也是一条记录 —— 可审计。

### B. `cache` 增加 `temp` scope（第五个）

```text
aicoding cache status --json                  # 含 temp：数量、总大小、最老创建时间
aicoding cache clean --scope temp [--keep N] [--dry-run] --json
```

保留策略（与 0013 同构）：

```text
默认保留：最近 3 个失败现场（供诊断） + 任何 24h 内创建的
默认回收：其余全部
永不回收：ledger 中标记 outcome=investigating 的（人工标记）
孤儿处理：%TEMP% 中匹配 aicoding-* 但不在 ledger 里的 → 列出并可 --adopt 回收
          （这批就是本次发现的 79 个历史遗留）
```

**跨仓边界**：ledger 记 `repoRoot`，`cache clean --scope temp` 默认只回收
**本仓创建的**；`--all-repos` 才跨仓（显式，防误删他人 worktree 的诊断现场）。

### C. 预算与权限的"释放"—— 先厘清再动手

owner 提到"权限和预算回收"。实测盘点后结论是：

| 资源 | 现状 | 是否需要释放 |
|---|---|---|
| 临时目录 | **泄漏 79 个** | ✅ 本项 A/B |
| Receipt / reports | 内容寻址，0013 已治理 | ❌ 不需要 |
| work attempts / plan state | 追加不可变，是审计轨迹 | ❌ **刻意不回收** |
| token 预算 | AiCoding 不调模型，只记账 | ❌ 无可释放 |
| 文件权限 | 未修改任何 ACL | ❌ 无可释放 |
| Git 锁 / index.lock | Git 自己管理 | ❌ 不越界 |
| pwsh / go 子进程 | 均 `exec.CommandContext`，随 ctx 结束 | ⚠️ 需断言（见下） |

**所以"预算与权限回收"在本仓库不存在实际对象**——除了一条：
**子进程孤儿检查**。新增 `doctor` 一项：检测本机是否存在
父进程已退出的 `aicoding`/`pwsh` 遗留子进程（**只报告，不 kill** ——
杀别人的进程是危险操作，交给用户）。

### D. 迭代收尾的统一入口

高频迭代场景需要一条"收工"命令，但**不新增命令域**，用既有 cache：

```text
aicoding cache clean --json          # 无 --scope = 全部可清 scope 按各自策略
```

在 `doctor --all` 增加一条 Warn 级 bloat 检查（0013 已有 test-results 版本，
本项扩展到 temp）：temp 超过 20 个或 100 MB 时提示 `cache clean --scope temp`。
**只提示不自动清**（沿用 0013 裁决）。

## 四、明确不做

- **不做守护进程/定时清理**（0022 已拒绝方案 B）。
- **不回收 work/plan 审计轨迹**（它们是证据，不是垃圾）。
- **不 kill 任何进程**（只报告孤儿）。
- 不做跨机器/远端清理（远端归 CI 的 artifact retention，GitHub 自己管）。
- 不清 `%TEMP%` 中非 `aicoding-*` 前缀的任何东西（边界铁律）。
- 不做"权限回收"（无实际对象，见 C 表）。

## 五、自测（可信任方式）

```powershell
go test ./internal/cache/... ./internal/kit/... ./internal/platform/...

# A：登记
bin\aicoding.exe fresh-clone --profile Smoke --json
#   成功 → ledger 出现 created + released 两行，%TEMP% 无残留
#   人为制造失败（改坏 .gitmodules）→ ledger 出现 outcome=failed，目录保留

# B：回收 + 孤儿收养
bin\aicoding.exe cache status --json                      # temp scope 列出 79 个（含孤儿）
bin\aicoding.exe cache clean --scope temp --dry-run --json  # 清单 = 全部 - 最近3失败 - 24h内
bin\aicoding.exe cache clean --scope temp --adopt --json    # 回收历史孤儿
#   断言：最近 3 个失败现场仍在；非 aicoding-* 前缀的目录一个都没动（负例）
#   断言：其它 worktree 创建的目录未被回收（跨仓负例）

# C：孤儿子进程只报告不杀
bin\aicoding.exe doctor --all --json      # 含 orphanProcesses 计数，且不含任何 kill 动作

# D：bloat 提示
#   造 25 个 temp → doctor --all 出现 Warn → cache clean 后消失

bin\aicoding.exe test --profile Full --json ; git status --porcelain
```

通过判据：
1. 成功路径 ledger 有 created+released 且 `%TEMP%` 零残留。
2. 失败路径保留现场且**被登记**（不再是无主）。
3. 79 个历史孤儿可被 `--adopt` 回收，且最近 3 个失败现场保留。
4. **非 `aicoding-*` 前缀零触碰**（负例贴输出）。
5. **跨 worktree 不误删**（负例贴输出）。
6. 孤儿进程只报数不 kill。
7. work/plan 审计轨迹在任何 clean 后完好。

