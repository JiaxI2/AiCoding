# ADR 0006: adrreview —— "新 Primitive ADR 必含 §12 自评"门禁

PrimitiveReview: required

## Status

Accepted。实现于 `internal/adrreview` + 测试 Registry 静态用例 `ADR-001`（三档全跑）。
本 ADR 落地 todolist 0001。

## TL;DR

[Primitive 宪法](../architecture/PRIMITIVE_CONSTITUTION.md) 约定"每个新 Primitive/新领域的
ADR 必须含 §12 Checklist 自评"。这条约定里**可机器化的部分只有存在性**（写没写自评节），
质量判断留给人。`adrreview` 就只做这一件事：枚举 `docs/decisions/*.md`，对声明
`PrimitiveReview: required` 的 ADR 检查 `## §12 Checklist 自评` 节是否存在，缺失即报缺口；
测试引擎以静态用例 `ADR-001` 消费它，缺口即门禁红。

## Decision

- **显式选择而非猜测**（约定优于配置、避免误报）：ADR 在标题下声明
  `PrimitiveReview: required` 才受检；历史 ADR（如 0002）无该头则忽略；可写
  `PrimitiveReview: n/a` 显式豁免。已标注：0003/0004/0005/0006。
- **只查存在性**：锚点 `## §12 Checklist 自评` 在文件中出现即通过——不解析勾选项、
  不判断内容质量（那是评审的事；机器判质量必然 theater）。
- **登记进唯一测试 Registry**：`ADR-001`（Category `ADR_REVIEW`，static，Required，
  三档全跑），`runStatic` 分发到 `checkADRPrimitiveReviews` → `adrreview.Check`。
  不加 CLI 顶层命令、不加 verify 聚合器第二处（CI 单源）。
- 顶层子目录（`freeze-and-acquisition/` 等 plan 工件集）不是 ADR，不扫描。

## §12 Checklist 自评（Primitive 宪法）

**架构**
- 单一职责？是——只枚举 ADR 并报"required 但缺 §12 节"的缺口，别的不管。
- 可继续拆分？否——`Check` + `scanFile` 已最小。
- 能被直接复用？是——纯读函数；testengine 是第一个消费者，未来 verify 聚合器可复用。
- 存在重复实现？否——与 `todolist`（读待办头部）形态相似但主题/目录/字段不同；若将来出现
  第三个"读 markdown 头部字段"的 Primitive，应抽共享 header-scan helper（路标，不预建）。
- 真的需要新 Primitive？是——宪法约定此前无机器守卫，靠人记得。

**性能**
- Fast Path？是——只读 `docs/decisions/` 顶层 `*.md`，逐文件单遍扫描、两条件命中即停。
- 无关扫描？无——**零仓库扫描**，成本与仓库大小无关（`BenchmarkCheck` 度量）。
- 重复 IO / 计算 / Agent / 工具调用？无——纯 Go、每文件读一次、无子进程无网络。
- 最小输入/输出？是——输入 `repo`，输出 `Report{items, gaps}`，不含文件正文。

**质量**
- 确定性？是——按文件名排序，相同文件相同输出。
- 接口稳定？是——新增包与静态用例，不改任何既有接口。
- 独立测试/Benchmark？是——单测覆盖 required-有节/required-缺节/无头忽略/n-a 豁免/
  子目录排除/目录缺失，加 `BenchmarkCheck`。
- 自由组合？是——testengine 静态用例组合它；`TestRegistryHasPrimitiveChecklistGate`
  守卫登记不被移除。

## Rollback

删除 `internal/adrreview`、engine.go 的 `ADR-001` 登记与 `checkADRPrimitiveReviews`、
engine_test 的守卫测试、各 ADR 的 `PrimitiveReview:` 头；内核零改动。
