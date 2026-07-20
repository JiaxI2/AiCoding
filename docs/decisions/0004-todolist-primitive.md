# ADR 0004: todolist Primitive（待实现工作清单）

PrimitiveReview: required

## Status

Accepted。实现于 `internal/todolist` + CLI `aicoding todolist` + `docs/todolist/`。

## TL;DR

需要一个地方记录"已规划、尚未实现"的工作项，先落完整计划、后续实现并绿灯。做法不是造流程，
而是一个最小 Primitive：读取 `docs/todolist/*.md` 的头部、汇报每项 status——单一职责、零仓库
扫描、确定性、可独立测试/benchmark。

## Context

- 复杂改动常常需要"先规划、分步实现"；此前只有 `docs/decisions/`（已定决策）与 `docs/spec/`
  （计划模式工件），缺一个轻量的**待办队列**：明确 Planned → In-Progress → Done 的生命周期。
- 约束：不新增流程/大模块（Primitive First）；放置位置须过 `governance layout`
  （markdown 必须在 `docs/` 下）→ 选 `docs/todolist/`，与既有 planning 目录同域。

## Decision

- **约定优于配置**：`docs/todolist/NNNN-slug.md`，头部 `Status:` + `Verify:` + 首个 `# ` 标题；
  `README.md` 不计入。`Verify` 是"证明已完成"的可执行命令——绿灯由命令证明，不靠口头。
- **Primitive**：`internal/todolist.List(repo) (Report, error)`，只读 `docs/todolist/` 一个目录，
  解析头部（遇正文即停，不读整篇），按文件名排序返回 items + summary。
- **CLI**：`aicoding todolist [--json]`（只读 domain 命令，`report.Result` 信封，带 `elapsedMs`）。
- 不做：不跑 `Verify` 命令（跑验证是 Workflow/门禁的职责，Composition First）；不扫描仓库；
  不加写操作（改 status 是编辑文件，人/agent 直接改）。

## §12 Checklist 自评（Primitive 宪法）

**架构**
- 只有一个职责？是——枚举 todo 头部并汇报 status，不跑命令、不改状态。
- 可继续拆分？否——`List` + `parseItem` + `normalizeStatus` 已最小。
- 能被直接复用？是——纯读函数，任何 Workflow/gate 可调用。
- 存在重复实现？否——`docs/decisions`/`docs/spec` 是"已决策/规格"，本项是"待办队列"，语义不同。
- 真的需要新 Primitive？是——现有面无"带状态的待办队列"能力。

**性能**
- 有 Fast Path？是——只读单目录、只读文件头（遇正文即停）。
- 无关扫描？无——绝不扫描仓库树（`BenchmarkList` 度量，成本与仓库大小无关）。
- 重复 IO / 计算？无——每文件读一次头部。
- Agent/工具调用？无——纯 Go、无子进程、无网络。
- 最小 Context？是——输出只含 file/title/status/verify，不含正文。

**质量**
- 确定性？是——按文件名排序；相同文件相同输出（`TestListParsesHeaderAndSummarizesDeterministically`）。
- 接口稳定？是——新增命令与包，不改任何既有接口。
- 最小输入/输出？是——输入 `repo`，输出标准 `Report`。
- 独立测试/Benchmark？是——`internal/todolist` 全套单测 + `BenchmarkList`。
- 自由组合？是——后续 `Verify` 门禁（TODO 0001）可组合本 Primitive 读取待办再逐项验证。

## Rollback

删除 `internal/todolist`、CLI catalog 中一行 + `runTodolist`、`docs/todolist/`；内核零改动。
