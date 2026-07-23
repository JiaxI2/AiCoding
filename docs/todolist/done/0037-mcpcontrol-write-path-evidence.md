# TODO 0037: mcpcontrol 写路径证据补齐

Status: Done
Verify: go test ./internal/mcpcontrol -count=1 -coverprofile=test-results/0037-final-cover.out && go tool cover -func=test-results/0037-final-cover.out && bin/aicoding.exe test --profile Full --reuse off --json && bin/aicoding.exe test --profile Release --reuse off --json

## 实测基线

- 2026-07-23 按包内物理行数实测 `internal/mcpcontrol` 为 2217 source LOC / 444 test LOC，
  test/source 比例 20.0%。`internal/` 下 source LOC 大于 1000 的 9 个包按同口径排序，
  比例中位数为 34.1%。
- `go test ./internal/mcpcontrol -count=1 -coverprofile=...` 的包级语句覆盖率为 46.3%，但
  `RunLifecycle`、`RunCatalogLifecycle`、`runLifecycle`、`Status`、`StatusCatalog`、
  `Verify`、`VerifyCatalog`、`writeInstallState`、`restoreConfigBackup` 九个目标函数均为 0.0%。
- `docs/architecture/CLI_MCP_CONTROL_PLANE.md` 明确规定 MCP 在单次写操作内使用 config
  backup 与 staged runtime 恢复。当前 install/update 在 `writeInstallState` 失败后直接返回，
  与该既有契约存在待负例确认的不对称；本项不新增契约或治理领域。

## 实施边界

1. 先构造真实 `writeInstallState` 失败，让 `writeManagedBlock` 已改写配置且返回 backup，
   断言操作失败后的配置最终状态；保存修复前失败输出。
2. 若负例证实配置未恢复，只在 install/update 的 state 写失败分支调用既有
   `restoreConfigBackup`，同时保留原错误；不得重构 lifecycle 或改动其他分支。
3. 用入口级测试真实覆盖九个目标函数；write effect 优先，read/status/verify 次之。
4. 最终每个目标函数的语句覆盖率必须大于 0%；test/source 比例不低于实测中位数 34.1%。

## 完成定义

- 失败恢复负例先红后绿，原始输出与配置最终状态写入本条目。
- 九个目标函数逐项有非零语句覆盖；最终 coverprofile 路径固定为
  `test-results/0037-final-cover.out`。
- Full、Release 各真跑一次，并在本条目记录固定 summary 路径。
- docsync all、governance dependencies/lint、plan verify、todolist 与 capability 投影全绿。
- 同批 CHANGELOG、正常 hooks 提交；不使用 `--no-verify`，不扩大 mcpcontrol 生产改动面。

## 失败恢复：先红后绿

负例从正式 `RunCatalogLifecycle(update)` 入口执行：临时 fake Python 完成版本探测与两个 pip
步骤，`writeManagedBlock` 真正改写 Codex config 并产生 backup；随后普通文件占据 state 父路径，
使 `writeInstallState` 的 `MkdirAll` 真实失败。修复前原始摘要：

```text
=== RUN   TestLifecycleStateWriteFailureRestoresManagedConfig
config was not restored after writeInstallState failure:
want="personality = \"pragmatic\"\n"
got="personality = \"pragmatic\"\n\n# BEGIN AICODING MCP visio-mcp\n...\n# END AICODING MCP visio-mcp\n"
BackupPath=".../config.toml.bak-20260723-121623.042803100"
Errors=["mkdir .../.aicoding/state/mcp/visio-mcp: The system cannot find the path specified."]
--- FAIL: TestLifecycleStateWriteFailureRestoresManagedConfig (2.78s)
FAIL
FAIL github.com/JiaxI2/AiCoding/internal/mcpcontrol 6.087s
EXIT=1
```

这证明已有 `docs/architecture/CLI_MCP_CONTROL_PLANE.md` §6.4 的单次 MCP 写操作恢复契约
覆盖该场景。生产修复仅在 state 写失败分支增加对既有 `restoreConfigBackup(configPath,
backup)` 的调用，并在恢复也失败时追加错误；没有改动 lifecycle 的其他分支。修复后同一命令：

```text
=== RUN   TestLifecycleStateWriteFailureRestoresManagedConfig
--- PASS: TestLifecycleStateWriteFailureRestoresManagedConfig (2.57s)
PASS
ok github.com/JiaxI2/AiCoding/internal/mcpcontrol 5.988s
EXIT=0
```

## 覆盖与比例结果

- `go test ./internal/mcpcontrol -count=1`：既有与新增 20 个测试全部通过。
- `test-results/0037-final-cover.out`：包语句覆盖率从 46.3% 提升到 67.3%。
- 九个目标函数的语句覆盖率：`RunLifecycle` 100.0%、`RunCatalogLifecycle` 100.0%、
  `runLifecycle` 57.1%、`Status` 100.0%、`StatusCatalog` 100.0%、`Verify` 100.0%、
  `VerifyCatalog` 100.0%、`writeInstallState` 92.3%、`restoreConfigBackup` 100.0%。
- 最终物理行数为 2220 source LOC / 932 test LOC = 42.0%；高于开工中位数 34.1%，也高于
  把本项新比例纳入九包后重算的 35.5%。体量增长全部来自入口级回归与测试 helper；生产代码
  只增加上述 3 行恢复调用。

## 仓库级验证路径

- Full：`test-results/0037-final-full/summary.json`，`73 total / 69 pass / 0 fail /
  0 warn / 4 skip`，结论 `PASS`，Receipt
  `sha256:65d5ee2b1d855765bd0cbc924f7875517dcdc2217047e78706be2bef1055cf5a`。
- Release：`test-results/0037-final-release/summary.json`，`73 total / 73 pass / 0 fail /
  0 warn / 0 skip`，结论 `PASS`，Receipt
  `sha256:ff039bb5bd8ab7408793af80abba5b7adeed932ab78513e5333a061819c232de`。
- 两次独立执行均绑定 index Tree `bdb9d5b49d79ba6f099f93af4eb950694c65bcec`，未使用
  Receipt 复用。
