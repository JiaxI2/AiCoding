# 05 治理架构：Hook、Lint、Policy（Governance Architecture）

Status: Derived View（派生视图）

> 本文不定义新契约；各政策的权威文本在 `docs/governance/`，DocSync 规范见
> [DOC_SYNC_PLUS_SPEC](DOC_SYNC_PLUS_SPEC.md)，冲突时以其为准。

## 本篇回答的问题

- Hook 如何设计？新增一个 Hook 应该如何接入？
- 如何保证一致性（多入口不漂移）？
- 治理层现在到底拦什么？

## 1. 治理层清单（现在拦什么）

| 机制 | 触发点 | 它具体拦什么 |
|---|---|---|
| `.githooks/pre-commit` → `aicoding hook pre-commit` + `plan check --staged` | 每次 `git commit` 前 | 五项并行只读强制检查；另按 `plan-policy.json` 做毫秒级 Plan Mode 触发判断。TODO 0004 阶段后者只告警，待批准绑定完成后升为强制。 |
| `.githooks/commit-msg` → `aicoding hook commit-msg` | 提交信息写完后 | 提交信息格式（type taxonomy：`feat/fix/docs/style/refactor/perf/test/build/ci/chore`）。 |
| `governance lint` | hook / 手动 / verify 聚合 | 仓库治理规范快检。 |
| `governance dependencies` | 同上 | 依赖方向与稳定身份（下层不得观察上层、身份不得编码版本、激活 manifest 不得带 URL）。 |
| `governance layout` | 同上 | 目录布局（如：文档必须在 `docs/` 或白名单位置）。 |
| `governance reuse` | 同上 | 复用治理证据（新实现必须先证明不能复用既有实现）。 |
| `governance capabilities` | 同上 | `internal/` 一级包孤儿、公共命令、架构文档、stable 验证与生成索引一致性。 |
| `docsync staged\|all\|ci\|release` | hook / CI / 发布 | 风险文件变更必须携带文档更新；并拒绝手改 README/`docs/CAPABILITIES.md` 能力生成索引。 |
| Style | pre-commit / 手动 | `.clang-format` + C99 kit 的 `fmt`/`check`（[01](01-system-architecture.md) §6.7）。 |
| Template | GitHub 侧 | `.github/` Issue 表单、PR 模板、`RELEASE_TEMPLATE.md` + `verify release-notes` 机器校验。 |
| CI | push / PR / 每周 / 手动 | `.github/workflows/aicoding-ci.yml`：push/PR 跑 Smoke；每周/手动跑 Release，并以独立 clean-clone Full leaf command 执行临时 clone 中的 `go test ./...`。 |

## 2. Hook 设计原则

1. **hook 是薄壳**：`.githooks/` 脚本只转调预构建 Go CLI。检查逻辑全部在 Go
   （`internal/*`），脚本里不写 pattern 或批准判断业务。
2. **快且并行**：pre-commit 的五项强制检查并行执行（有界并发），Plan Mode 路径判定
   单独保持在 200 ms 内；
   慢检查不进 hook，进 `verify`/`test`。
3. **只读**：hook 只判定不修复；修复权留给显式的写命令（如 `skill c99-standard-c fmt`）。
4. **失败信息可纠错**：门禁文案必须指明违反的规则与正确路径，而非仅报告失败——
   这是 Agent 的第④个知识进入点（[02](02-context-architecture.md) §1）。

## 3. 新增一个 Hook / 门禁的接入步骤

1. **先对抗性提问**：它是不是既有门禁的内部步骤？多数"新检查"应扩展现有检查项，
   而不是加新门禁。
2. 确需新增：检查逻辑写在 Go `internal/` 对应包（不写在脚本/CI 里）。
3. 挂接点二选一：提交时强制 → 挂进 `hook pre-commit` 的并行计划；
   仓库级验证 → 挂进 `verify` 聚合器的检查列表。
4. 写门禁错误文案（满足上文原则 4），并把检查登记进唯一测试 Registry
   （`test --profile` 能跑到它）。
5. 同步文档与 `CHANGELOG.md`（docsync 会拦没带文档的接入）。

禁止：在 `.githooks/` 脚本内直接写检查逻辑；在 Taskfile/CI/PowerShell 里建第二套
聚合器；绕过唯一测试 Registry 单独跑一套检查。

## 4. 如何保证一致性

| 机制 | 一句话 |
|---|---|
| 单一权威 | 命令目录=`internal/cli` catalog；能力目录=`config/internal-capabilities.json`；测试=`internal/testengine`；报告=`internal/report`；生命周期=`internal/lifecycle`。Taskfile、CI、hook 全是到这些权威的短路由，自己不携带逻辑。 |
| digest 对账 | 契约/事实/意图三个 digest 让"两次运行是否同一件事"可机器判断。 |
| docsync | 代码与文档强制同步，文档不一致在提交时就被拦。 |
| 同一检查只登记一次 | CI 不重复调用 `doctor`/`verify` 聚合器——唯一测试 Registry 已登记对应 leaf 检查，避免同一检查两套结果。 |
| 机器优先 | 政策尽量落成 `governance *` 可执行门禁；写在文档里但机器不拦的规则，视为待机器化的债务。 |
