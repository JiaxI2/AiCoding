# TODO 0022: 命令延迟对齐 Git 级（查询类 <1s，聚合类 <3s，工作类靠复用）

Status: Planned
Verify: doctor --all / lifecycle status --scope all 中位数 <1200ms；查询类全部 <400ms；GO-007 并发包登记门禁绿；Full 中位数较改前下降 ≥60s

> 来源：owner 提问"如何达到 Git/Unix/Docker 级的响应" + 《aicoding-go-under-10s-kit》评审。
> **本项只做已实测、低风险的那一半；Kit 提议的守护进程与远端 attestation 全部拒绝。**

## 一、先厘清一个类比错误（这决定了方案的正确形状）

`git status` 快，`git clone` 不快（本仓 51s，0016 实测）。**Git 的快命令全是
"对预计算索引的查询"，慢命令全是"真干活"。** 拿 `aicoding test --profile Full`
（跑 go test + race）去比 `git status`，是拿测试运行器比查询——Git 从不跑你的测试套件。

正确的目标不是"所有命令 <10s"，而是**按命令性质分三档**：

```text
查询类  对既有事实/证据的读取        目标 <400ms   ← 对标 git status/log
聚合类  多域诊断汇总                 目标 <1.2s    ← 对标 git fsck --connectivity-only
工作类  真的跑测试/构建/clone        不设绝对上限，
                                     但"同一内容第二次"必须退化为查询类
```

**第三档正是 validationevidence 已经解决的**：Release 76s → 392ms（实测），
`test --profile Full` 同理。**"10 秒内"这个目标在工作类上已经达成了，
只是默认关着**（`--reuse off`，等 main 三次绿灯晋级）。

## 二、实测：慢在哪里（2026-07-21，warm）

```text
进程启动地板（version）                   107 ms   ← Go CLI 的底，无法再降
todolist                                  104 ms   ✅ 已是查询级
kit list                                  108 ms   ✅
plan check --staged                       168 ms   ✅
governance layout                         218 ms   ✅
validation check --profile Release        285 ms   ✅
governance dependencies                   683 ms   ⚠️ 三处 filepath.WalkDir 全仓扫描
lifecycle status --scope all             2108 ms   🔴
doctor --all                             2757 ms   🔴
verify --profile Smoke                   2991 ms   🔴
```

**根因已定位（doctor --all 的 JSON 分解）：**

```text
kit adapter              2 ms
runtime-skill adapter 1775 ms   ← 占 72%
```

`internal/lifecycle/runtime_skill.go:265` 走 `pwsh`/`powershell` 外部进程。
**实测 `pwsh -NoProfile` 空跑就要 541 ms** —— 这是 PowerShell 启动地板，
与 AiCoding 代码无关。runtime-skill 每次至少付 1–3 次这个地板。

> **一句话结论：AiCoding 的三个慢命令，慢在 PowerShell 进程启动和全仓扫描，
> 不慢在架构。** 这也解释了为什么 Git/Docker 感觉快——它们不 spawn PowerShell。

## 三、Kit 评审：采纳 1/4，拒绝 3/4

Kit 自己的结论是诚实的（"物理冷执行 10 秒不可落地，只有同步决策 10 秒可落地"），
但它推荐的 A+B+C 组合里只有 A 是本仓库该做的：

| Kit 方案 | 裁决 | 理由 |
|---|---|---|
| **A：内容寻址 Receipt + Impact Graph** | ✅ **已有 + 已规划** | Receipt 早已实现且更强；Impact Graph = 0021 的 `impact-policy.json` + 0017 节点级失效 |
| **B：常驻本地守护进程**（预热、监听、后台预计算） | 🔴 **拒绝** | 违反"无守护进程"原则；引入进程生命周期、端口/管道、僵尸进程、状态一致性四类新问题；且**解决不了真正的瓶颈**——瓶颈是 pwsh 启动，守护进程不会让 pwsh 变快 |
| **C：远端 Attestation** | 🔴 **拒绝** | 需要远端信任根 + 签名 + 传输；本地单人仓库无收益；ADR 0007 已定"CI Receipt 不等同本地 Receipt" |
| **D：真正冷启动 10 秒** | 🔴 **拒绝**（Kit 自己也只列为实验） | 要砍 race/集成/fresh-clone 才可能，即砍验证强度 |
| `cmd/perfprobe` 独立探针 | 🔴 拒绝 | `doctor perf` + 0016 的四段耗时已覆盖；第二套测量工具即平行事实源 |
| `config/perf-budgets.yaml` / `test-groups.yaml` / `impact-rules.yaml` | ⏸ 概念采纳，形态拒绝 | 预算归 0014（进 typed command catalog）、影响规则归 0021（`impact-policy.json`），不新增三个 YAML |

**Kit 最大的价值是它的诚实分档**（同步决策 vs 物理冷执行）——这条采纳，
写进 0014 的延迟等级定义。

## 四、实现计划（六刀，全部低风险）

### 刀 1：runtime-skill 懒执行（收益最大，约 −1.7s）

`doctor --all` / `lifecycle status --scope all` 的 runtime-skill adapter 当前
**无条件 spawn pwsh**，即使本机根本没配置 runtime skill。

```go
// internal/lifecycle/runtime_skill.go
// 先做零成本的前置判定：未配置 runtime skill 时直接返回 skipped，不 spawn。
if !runtimeSkillConfigured(repo, opts) {
    return AdapterResult{ID: ScopeRuntimeSkill, OK: true, Status: "skipped",
        Warnings: []string{"no runtime skill configured; probe skipped"}}
}
```

判定依据只读既有配置/state 文件（`config/skill-sources.json`、
`.aicoding/state/`），**零进程、零 Git 调用**。
显式 `--runtime-skill NAME` 或 `--runtime-profile full` 时仍照常探测。

### 刀 2：`governance dependencies` 复用单次扫描（约 −400ms）

`internal/governance/dependencies.go` 有 **三处独立 `filepath.WalkDir`**
（第 269/367/570 行）。合并为一次遍历、多个访问者：

```go
// 一次 WalkDir，三个收集器
inv := walkOnce(root, collectImports, collectBadges, collectPaths)
```

宪法 §3 明确"禁止重复读取相同数据"——这是现成的违规点。

### 刀 3：聚合命令并行化（约 −30%，复用既有 runner）

`doctor --all` / `verify --profile Smoke` 内部的各域检查目前串行。
它们**互不依赖、全只读**，直接用既有 `runner.Run`（信号量 + 索引写回，
顺序确定性已保证）并行：

```go
runner.Run(ctx, checks, runner.Options{MaxParallel: 4})
```

**不新建调度器**（Full 优化阶段三的 DAG 调度仍按当时裁决挂起——
那是给测试引擎的，这里只是把已有 runner 用在诊断聚合上）。

### 刀 4：GO-002 race 降频不降级（Full 约 −70~90s）

**实测依据**：单个包的 race 冷编译 12.7 s（其中测试本身只跑 3.0 s，9.7 s 全是编译）——
`-race` 有**完全独立的构建缓存**，与 GO-001 零复用，等于每次 Full 付两次全量编译。

race 的价值是发现并发缺陷，而**并发代码不常改**。按修改频率分层：

```text
Full     只对声明了并发的包跑 race（并发包清单静态登记，见下）
Release  全仓 race（不变）
每周 CI  全仓 race（不变）
```

并发包清单落 `config/impact-policy.json` 的同一份文件（0021 已建，不新增配置）：

```json
"raceScope": {
  "packages": ["internal/runner", "internal/validationevidence",
               "internal/testengine", "internal/kit"],
  "reason": "packages with goroutines, shared state, or concurrent file writes"
}
```

**门禁保证不漏**：新增 static 用例 `GO-007`，扫描全仓 `.go` 文件中
`go func` / `sync.` / `chan ` 的出现位置，断言其所属包 ⊆ `raceScope.packages`。
**新写并发代码却没登记 → Full 直接红。** 这是"降频不降级"的机器保证，
不是靠人记得更新清单。

> 这不是砍验证：全仓 race 仍在 Release 与每周 CI 跑；Full 只是不再为
> 「没有并发的包」重复付一次全量 race 编译。

### 刀 5：PowerShell 面收尾（约 −1.8 s，与刀 1 同源）

**先说实测结论：PowerShell 面已经收敛得很好，剩下的问题只有一处。**

```text
PWSH-001 inventory        57 ms   ← 纯文本分析，不 spawn
PWSH-002 budget           85 ms   ← 纯文本分析，不 spawn
PWSH-003 默认入口          2 ms   ← 静态检查
doctor pwsh              111 ms   ← 文本扫描，不 spawn
repohealth.go:152        "pwsh" 只作为 hook 文件里的字符串 token 检测，不 spawn
──────────────────────────────────────────────────────
唯一真 spawn：internal/lifecycle/runtime_skill.go:265  → 1775 ms
```

所以 PowerShell 优化 = **刀 1 的懒执行**，此外只剩两个小项：

1. **`repohealth.discoverTools` 的 `exec.LookPath` 探测**（`repohealth.go:589`）：
   实测每个工具 43–116 ms，六个工具 ≈ **360 ms**，且 `airepair` 单独 116 ms。
   改为**并发探测**（六个 goroutine，复用既有 `runner.Run`），
   360 ms → ≈ 120 ms（取最慢的一个）。
2. **ps1 退役进度纳入 `doctor pwsh` 输出**：`tools/specialty/*.ps1` 现有 22 个，
   0004/0006 已把 plan-mode 两个降级为薄壳。`doctor pwsh` 增加一行
   `remainingScripts / thinShells / deprecated` 计数，让退役进度可查
   （**只报数，不设门禁**——ps1 面已冻结，减少是自然过程，不必逼）。

**明确不做**：不为 pwsh 做进程池/预热/常驻（守护进程已在方案 B 拒绝）；
不重写任何 ps1 为 Go（ps1 面冻结，只减不增，等自然退役）。

### 刀 6：延迟等级登记 + 防回退门禁（0014 的执行部分）

把 0014 的等级表落到 `CommandDescriptor.LatencyClass`，
按本项实测值定档（**不用估算值**）：

```text
fast      <400ms   version/todolist/kit list/plan check/governance layout/validation check
standard  <1200ms  doctor --all/lifecycle status/governance dependencies/verify Smoke
                   （刀 1–3 完成后才够得着；未达标前先记实测值不设门禁）
work      不设上限 test --profile */release/fresh-clone —— 但必须支持 Receipt 复用
```

`doctor perf` 实测超预算 1.5× → Warn，3× → Fail（机器差异容忍）。

## 五、明确不做

- **不做守护进程 / 文件监听 / 后台预计算**（Kit 方案 B）。
- **不做远端 attestation**（Kit 方案 C）。
- **不追求物理冷执行 10 秒**（要砍验证强度，红线）。
- **不新增 perfprobe / perf-budgets.yaml / test-groups.yaml**（既有 `doctor perf`
  + typed catalog + `impact-policy.json` 已覆盖）。
- 不动 `go test` / `-race` 本身（那是工具链成本，0017 的节点级复用才是正解）。
- 不为省 100ms 去做 Go 编译参数调优（进程启动 107ms 已是地板）。

## 六、自测（可信任方式）

```powershell
go test ./internal/lifecycle/... ./internal/governance/... ./internal/repohealth/...

# 刀 1 负例：未配置 runtime skill 时不得 spawn pwsh
#   用 Process Monitor 或在 executor 注入计数器，断言 spawn 次数 = 0
bin\aicoding.exe doctor --all --json     # runtime-skill status=skipped
bin\aicoding.exe lifecycle status --scope all --runtime-profile full --json
#   显式要求时仍探测（status != skipped）——不能因为快就丢覆盖

# 刀 2：WalkDir 调用次数
#   注入计数器断言全仓遍历次数 = 1（改前为 3）

# 刀 3：并行后结论不变
#   同一输入串行/并行两次，剔除 elapsed 后 JSON 字节一致

# 前后对比（每条 5 次取中位）
foreach ($c in @('doctor --all','lifecycle status --scope all',
                 'governance dependencies','verify --profile Smoke')) {
  1..5 | % { (Measure-Command { & bin\aicoding.exe $c.Split(' ') --json }).TotalMilliseconds }
}

bin\aicoding.exe doctor perf --json      # 等级门禁全绿
bin\aicoding.exe test --profile Full --json
```

通过判据：
1. `doctor --all` / `lifecycle status --scope all` 中位数 **<1200ms**（改前 2757/2108）。
2. `governance dependencies` **<400ms**（改前 683）。
3. 未配置 runtime skill 时 **pwsh spawn 次数 = 0**（负例断言）；显式指定时仍探测。
4. 全仓 `WalkDir` 在 dependencies 路径上 **调用一次**。
5. 串行/并行结论字节一致。
6. **无新增守护进程、无新增测量工具、无新增 YAML 配置**。
7. **GO-007 并发登记门禁**：未登记的包里写并发代码 → Full 红（负例贴输出）。
8. **Release 的 GO-002 仍是全仓 race**（回归断言，防止降频误伤 Release）。
9. `doctor pwsh` 输出含 ps1 退役进度计数。
10. `test --profile Full` 全绿，**验证强度零削减**（全仓 race 仍在 Release + 每周 CI）。
