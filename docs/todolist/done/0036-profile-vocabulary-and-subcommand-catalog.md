# TODO 0036: `--profile` 词汇表正交化与子命令登记面扩展

Status: Done
Verify: go test ./internal/cli/... ./internal/kit/... ./internal/testengine/... && bin/aicoding.exe test --profile Full --reuse off --json && bin/aicoding.exe test --profile Release --reuse off --json

## 范围

- ADR 0012 与实现同批提交；`--profile` 只保留产品 Smoke/Full/Release。
- C99 verify 改为 `--depth fast|full`；kit verify 改为 `--level smoke|lifecycle`；
  `kit test` 正式登记且 canonical 形式不带 profile。
- typed command catalog 成为子命令/alias、help route 与 pluginview quickstart route 的唯一来源。
- 显式新增冻结面和 FREEZE 负例；不新增 Primitive、治理领域或测试档。

## 兼容边界

旧 `--profile fast|full`、`kit verify --profile Smoke|Lifecycle` 与
`kit test --profile Smoke` 在 ADR 0012 窗口内继续成功并输出 deprecation warning；本轮不删除。

## 负例矩阵

1. 修复前 `kit verify` help/runtime 不一致先复现，修复后由一致性门禁消除。
2. catalog 外新增可路由子命令时 FREEZE 非零并点名路径。
3. 两类旧参数真跑成功且输出 deprecation warning。
4. 第四套 `--profile` 词汇注入时门禁非零。
5. pluginview quickstart 每条从 catalog 投影并逐条真跑成功。

原始输出统一保存到
`docs/operations/evidence/profile-vocabulary-negative-matrix.md`。

## 完成定义

- catalog/CLI/kit/testengine 直接测试和五项负例全绿，证据含退出码和原始输出。
- 当前 COMMANDS、Taskfile、capability 与 Kit/C99 文档使用 canonical 参数；历史记录不改写。
- Full、Release 各真跑一次并把固定 summary 路径写回本条目。
- docsync all、governance dependencies/lint、plan verify、todolist、capability 投影全绿。
- 同批 CHANGELOG、正常 hooks 提交；不使用 `--no-verify`。

## 验收证据

- Full 真跑：`test-results/0036-final-full/summary.json`，`73 total / 69 pass /
  0 fail / 0 warn / 4 skip`，结论 `PASS`。
- Release 的 dirty/index 预提交运行均为 `73 total / 72 pass / 0 fail / 1 warn`；
  唯一 `FRESH-004` 原因是 fresh-clone 基线只记录当前 HEAD Tree，而本轮
  transport-sensitive `Taskfile.yml` 尚未进入 HEAD。`fresh-clone --profile Release --json`
  本身已全步通过；提交后的 clean-tree Release 写回固定路径
  `test-results/0036-final-release/summary.json`。
- 项目级测试：`go test ./... -count=1` 通过；FREEZE-008/009 restored 运行通过。
- 文档/治理：`docsync all`、`governance dependencies`、`governance lint`、
  `governance capabilities`、`plan verify`、`todolist` 均通过。
