# TODO 0038: 延后拆分观察项与触发条件

Status: Done
Verify: bin/aicoding.exe docsync all --json && bin/aicoding.exe governance dependencies --json && bin/aicoding.exe plan verify --json && bin/aicoding.exe test --profile Full --reuse off --json && bin/aicoding.exe test --profile Release --reuse off --json

## 范围与实测

本项只在 `docs/architecture/AICODING_CORE_ARCHITECTURE.md` 的领域包相关位置记录观察项，
不修改 Primitive Constitution、acquisition 专章或任何生产/测试代码，不执行拆包。

1. `internal/cstyle`：核心架构 §4 已允许受控 specialty；生产 import 实测只来自
   `internal/cli`。第二个语言 Kit 出现时重新评估抽离。
2. `internal/kit`：2026-07-23 实测 5839 source LOC，而提示数值为 5814；按仓库实测为准。
   生产消费者实测为 cache、CLI、repohealth、testengine、lifecycle。两个消费者需要独立复用
   同一子域，或出现跨文件循环依赖时触发拆分评估；体量本身不是拆分理由。
3. Test Registry：`internal/testengine/engine.go` 实测 39 个 `Kind: "command"` leaf。
   静态检查命令是 Primitive，Registry 是组合器；不合并静态检查命令。

## 完成定义

- 章节标题为“延后拆分观察项与触发条件”，三条观察、现状与触发条件完整且没有执行拆分。
- 改动只含核心架构、TODO 生命周期与 CHANGELOG 文档。
- Full、Release 各真跑一次，固定 summary 路径分别为
  `test-results/0038-final-full/summary.json` 与
  `test-results/0038-final-release/summary.json`。
- docsync all、governance dependencies/lint、plan verify、todolist、capability 投影全绿。
- 正常 hooks 提交；不使用 `--no-verify`。

## 验收证据

- Full：`test-results/0038-final-full/summary.json`，`73 total / 69 pass / 0 fail /
  0 warn / 4 skip`，结论 `PASS`，Receipt
  `sha256:41bbcae0d598d6a2584fcc88025d87c5165e8fd70f638142e1351fec5a4ed0a1`。
- Release：`test-results/0038-final-release/summary.json`，`73 total / 73 pass / 0 fail /
  0 warn / 0 skip`，结论 `PASS`，Receipt
  `sha256:a0c65f03d3987cf0dbaa6201e9d2dc0f4e089c1c4b03f03858c3e0ceaf9f5415`。
- 两次独立执行均绑定 index Tree `7dc979318868bf8a5f9dbd53f5f3e1d006894b33`，未使用
  Receipt 复用。
- `docsync all`、governance dependencies/lint/capabilities、plan verify/check 与 todolist
  均通过；staged diff 只有四个文档生命周期文件。
