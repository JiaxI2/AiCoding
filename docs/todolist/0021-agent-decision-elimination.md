# TODO 0021: 消除中间 LLM 决策（结构化结论 + change verify meta-tool）

Status: Planned
Verify: bin/aicoding.exe change verify --json 一条命令替代"git diff → 影响解析 → 选 profile → 跑测试"四步；所有 report.Result 含机器可判的 category/retryable，Agent 无需读日志即可决定下一步

> 来源：《AiCoding Workflow Automation Kit v1.0.0》评审采纳项。
> 核心原则（owner 认同、本项落地）：
> **不要让 LLM 驱动工作流前进；让工作流驱动工具执行，只在无法由程序确定时调用 LLM。
> 最大优化点不是换模型，而是消除中间 LLM 决策。**
> AWO 论文实证：把重复工具序列编译成确定性 meta-tool，LLM 调用下降 11.9%、
> 成功率提升 4.2pp。本项取其**结论**，不取其运行时。

## 评审结论：原则全采纳，运行时全拒绝

Kit 提出七条 CLI（`workflow lint/compile/run/resume/trace/explain/optimize`）+
`Engine`/`Scheduler`/`StateStore`/`Cache`/`EffectLedger` 五个接口 +
第二份 `node-result.schema.json`。逐项对照本仓库既有实现：

| Kit 提议 | 本仓已有 | 裁决 |
|---|---|---|
| `node-result.schema.json`（12 字段） | `report.Result`（13 字段，全仓唯一信封） | 🔴 **拒绝第二 schema**；缺的字段**加进 report.Result**（见下） |
| `Engine` + `Scheduler` + ready-set 调度 | `runner.ExecutionPlan` + 信号量 + fail-fast | 🔴 拒绝；DAG 调度早有裁决（Full 优化阶段三，实测 17.9s 低于 40s 阈值，不启动） |
| `Cache`（node_version+inputs+env+config 五元键） | `validationevidence` 内容寻址 Receipt（八元身份） | 🔴 拒绝；已有的更强（绑 Git Tree、不可变、完整性校验、resultsDigest 审计） |
| `StateStore` + Event Log | `.aicoding/state/work/<id>/attempts.jsonl`（追加不可变） | 🔴 拒绝；loopkit 已有 |
| `Authorizer` + commit-time 见证刷新 | `plan approve` 绑 Tree + pre-push Context Gate + fail-closed | 🔴 拒绝；已实现且更严（绑内容而非绑 epoch） |
| `TransitionResolver` 确定性转移 | `loopkit.Decide`（纯函数、零 IO、五具名终止态） | 🔴 拒绝；同一职责已有权威 |
| **结构化结论**（category/retryable，让运行时不必读日志） | `report.Result` **只有 ok + errorKind 三值** | ✅ **采纳** —— 真实缺口 |
| **meta-tool 晋升**（重复序列编译成确定性复合工具） | 无 | ✅ **采纳** —— 真实缺口，且收益最大 |
| **workflow YAML + Compiler** | 无 | ⏸ **推迟**：这是 Workflow DSL，06-plugin-sdk §7 明确拒绝；等出现两个真实消费者且经 ADR 证明 Markdown/Skill 不足才评估 |
| Trace/OpenTelemetry 父子 span | 0016 已有四段耗时 + slowest_cases | ⏸ 推迟；本地单机无分布式追踪需求 |

**一句话：Kit 描述的运行时，AiCoding 九成已经有了，而且更强——
因为它们绑内容身份而不是绑运行时状态。真正缺的是两件事，本项做这两件。**

## 实现计划

### A. `report.Result` 增补机器可判字段（不新建 schema）

Agent 现在要读 `errors[]` 自然语言才知道该重试还是该改代码——**这正是"中间 LLM 决策"的源头**。

```go
// internal/report/result.go —— 三个新字段，全部 omitempty，向后兼容
Category  string `json:"category,omitempty"`  // 有限枚举，见下
Retryable bool   `json:"retryable,omitempty"`
NextAction string `json:"nextAction,omitempty"` // 已有 requiredAction 的泛化；命令字符串
```

`Category` 枚举（**封闭集合，新增须改常量并过测试**，防自由文本化）：

```text
none              成功
usage             参数错误        → 改命令，不重试
validation        门禁不通过      → 改代码，不重试
transient         环境/网络抖动    → 可重试
toolchain         工具链不匹配     → 换环境，不重试
evidence-missing  缺 Receipt      → 跑对应 profile
conflict          并发/锁         → 可重试
internal          实现缺陷        → 上报，不重试
```

改造范围：`report.Fail` 与各 handler 的失败路径逐一归类。
**判据：Agent 只读 `{ok, category, retryable, nextAction}` 四字段即可决定下一步，
不需要读 `errors[]`。** 这是本项对"消除中间 LLM 决策"最直接的贡献。

### B. `aicoding change verify` meta-tool（AWO 原则的具体落地）

当前 Agent 改完代码要走的真实序列（Trace 已可观测，0016 的数据支撑）：

```text
git diff --name-only
  → 判断影响面（LLM 读文件列表决定跑什么）   ← 这一步的"思考"可以被编译掉
  → 选 profile（Smoke? Full?）              ← 同上
  → aicoding test --profile X
  → 读结果决定下一步                         ← 同上
```

编译为一条确定性命令：

```text
aicoding change verify [--staged|--since <rev>] [--json]
```

内部（**全部复用既有 Primitive，零新增能力**）：

1. `gitx.StatusSnapshot` / `gitx.CommitFiles` 取变更集（单次调用）；
2. 用 **`plan-policy.json` 的同构 pattern 机制**判定影响面 →
   新增 `config/impact-policy.json`：路径 → 建议 profile 的静态映射
   （`internal/**/*.go` → Full；`docs/**` → Smoke；`config/schemas/**` → Full 等）；
3. 查 `validationevidence`：目标 profile 是否已有当前树的有效 Receipt →
   命中则直接返回 `ok:true, category:none`（**零执行**）；
4. 未命中则调既有 `testengine.Run`，profile 由第 2 步确定；
5. 返回带 `category`/`retryable`/`nextAction` 的 `report.Result`。

**这条命令把四步 LLM 决策压成一次工具调用**——正是 AWO 的 meta-tool 晋升，
但晋升对象是**手工识别的一条真实高频序列**，不是自动挖掘（见"明确不做"）。

### C. meta-tool 晋升的准入规则（写进文档，防滥用）

Kit 的 `optimization.meta_tool_candidate` 条件全部采纳为**人工评审清单**，
写入 `docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md` 新增一节
（loop 架构已拥有"何时该停"，meta-tool 是"何时该编译掉一步"，同域）：

```text
序列可晋升为 meta-tool 的充要条件：
1. 在真实使用中重复出现（≥5 次，人工确认，不自动统计）
2. 顺序稳定，中间步骤不需要语义决策
3. 输入输出可形成稳定 schema
4. 副作用与权限边界清晰
5. 可独立测试
6. 合并后不隐藏必要的可观测信息（内部子步骤仍进报告）
7. 由人批准（禁止运行时自动晋升）
```

## 明确不做

- **不引入 workflow YAML / Compiler / DSL**（06-plugin-sdk §7 拒绝清单）。
- **不新建 `node-result.schema.json`**（`report.Result` 是唯一信封）。
- **不建第二 runner / scheduler / state store / cache / authorizer**（五个都已有权威）。
- **不做自动 trace 挖掘与自动 meta-tool 晋升**（Kit 的 `workflow optimize --window 200`）——
  自动改自身运行时违反"不隐式修改无关状态"；且当前是单人仓库，人工识别序列足够。
- 不做 LLM 预算/模型配置编译（AiCoding 不调用模型，无从预算）。
- 不做分布式 Trace / OpenTelemetry。

## 自测（可信任方式）

```powershell
go test ./internal/report/... ./internal/cli/... ./internal/testengine/...
# category 封闭集合：新增非法值必须编译失败或测试失败（负例验证后撤销）

# A：结构化结论
bin\aicoding.exe kit verify --kit does-not-exist --profile Lifecycle --json
#   期望 category:"usage"、retryable:false、nextAction 非空
bin\aicoding.exe validation check --profile Release --target HEAD --json
#   miss 时期望 category:"evidence-missing"、nextAction 给出跑 profile 的命令
# 断言：全部失败路径的 category 都在封闭枚举内（遍历测试）

# B：change verify meta-tool
echo x >> README.md ; git add README.md
bin\aicoding.exe change verify --staged --json
#   期望：影响面判定为 docs → 选 Smoke（贴 chosenProfile 字段）
git restore --staged README.md ; git checkout README.md
echo x >> internal/plan/plan.go ; git add internal/plan/plan.go
bin\aicoding.exe change verify --staged --json
#   期望：影响面判定为 go → 选 Full
git restore --staged internal/plan/plan.go ; git checkout internal/plan/plan.go

# Receipt 命中时零执行：
bin\aicoding.exe test --profile Smoke --reuse off --json      # 先种一个 Receipt
bin\aicoding.exe change verify --json                          # 期望 executedCases:0，毫秒级返回
1..5 | % { (Measure-Command { bin\aicoding.exe change verify --json }).TotalMilliseconds }
#   Receipt 命中路径中位数 < 500ms（0014 的 standard 等级）

bin\aicoding.exe governance dependencies --json ; bin\aicoding.exe test --profile Full --json
```

通过判据：
1. **全仓仍只有一个 report schema**（`grep -rn "node-result\|NodeResult" .` 为空）。
2. 所有失败路径 category 落在封闭枚举内（遍历断言，贴输出）。
3. `change verify` 对 docs-only 与 go 改动分别选出 Smoke/Full（正反两例贴输出）。
4. Receipt 命中时 `executedCases:0` 且中位数 <500ms。
5. 内部子步骤仍出现在报告中（meta-tool 不隐藏可观测信息）。
6. `internal/` 下无新增 workflow/engine/scheduler 包。
