# TODO 0001: 测试引擎登记"新 Primitive ADR 必含 §12 自评"门禁

Status: Planned
Verify: go test ./internal/testengine/... -run TestRegistryHasPrimitiveChecklistGate

## 背景

[Primitive 宪法](../architecture/PRIMITIVE_CONSTITUTION.md) §12 约定：每个新 Primitive/新领域
的 ADR 必须包含"§12 Checklist 自评"小节。这是**可机器化的存在性检查**（不是判断质量，只查
是否写了自评），因此适合登记为唯一测试 Registry 的一个 leaf gate，而不是靠人工评审记得。

范例：ADR 0003 已含"§12 Checklist 自评"小节；本项要把"必须有该小节"变成门禁。

## 实现计划

1. **判定哪些 ADR 属于"新 Primitive/新领域"**（约定优于配置）：
   - 约定：这类 ADR 在正文包含标记锚点 `## §12 Checklist 自评`。
   - 判定"是否需要自评"的信号：ADR 引入了新的 `internal/<domain>` 包或新的 lifecycle
     `Scope*`/adapter。为避免误报，第一版采用**显式选择**：在 ADR 头部加一行
     `PrimitiveReview: required`（或 `n/a` + 理由）。没有该行的历史 ADR 不受约束。
2. **新增 Primitive**：`internal/adrreview`（单一职责：枚举 `docs/decisions/*.md`，解析
   `PrimitiveReview:` 头与是否含 `## §12 Checklist 自评` 锚点，返回缺口列表）。
   - 只读 `docs/decisions/`，不扫描仓库；确定性（按文件名排序）；可独立测试 + benchmark。
3. **登记 leaf gate**：`internal/testengine/engine.go` 增加一条 static 用例
   `ADR-001`（Category `GOVERNANCE` 或新 `ADR_REVIEW`），跑 `adrreview` 并在有缺口时 fail。
   - 或复用 `verify` 聚合器加 `verify.adr-review`，二选一（倾向测试 Registry，保持 CI 单源）。
4. **给 0003/0004 补 `PrimitiveReview: required` 头**，确保门禁在既有范例上绿灯。
5. **测试**：`internal/adrreview` 单测（有自评→通过、标 required 但缺自评→报缺口、
   历史 ADR 无该头→忽略）；`TestRegistryHasPrimitiveChecklistGate` 断言 Registry 含 `ADR-001`。
6. **文档**：ADR 0005 记录本门禁 + §12 自评；COMMANDS.md/CHANGELOG 同步。

## 完成定义（绿灯）

- `go test ./internal/adrreview/...` 全绿；
- `go test ./internal/testengine/... -run TestRegistryHasPrimitiveChecklistGate` 绿（Registry 已登记 `ADR-001`）；
- `bin/aicoding.exe test --profile Smoke --json` 中 `ADR-001` 通过；
- 本项 `Status` 改为 `Done`。

## 宪法对齐（本门禁自身）

- **可机器化、非 theater**：只查"自评小节是否存在"，不判断质量——判断留给人工评审。
- **单一职责**：`adrreview` 只枚举 ADR 并报自评缺口；跑 gate 由测试引擎（Workflow）组合。
- **Execution Cost First**：只读 `docs/decisions/`，零仓库扫描。
- **不新增顶层命令**：经既有 `test`/`verify` 入口暴露，不加 porcelain。
