# TODO 0042: `--reuse` 默认值晋级为 `auto`

Status: Done
Verify: go test ./internal/testengine ./internal/cli ./internal/validationevidence -count=1 && bin/aicoding.exe test --profile Release --reuse off --out test-results/0042-final-release-exact --json && bin/aicoding.exe docsync all --json

## 范围

依据 Validation Evidence Budget §13 的 `toolchainDigest.v2` 远端 3/3 证据与 ADR 0014，把
`test --profile ...` 默认值从 `off` 翻转为 `auto`。显式 `--reuse off`、`--force`、
`--verify-reuse`、Profile/Registry/raceScope 与 CI release-gate 两条命令均不改变。

唯一允许同步的既有测试断言是：

1. `internal/testengine/evidence_test.go` 的默认值精确相等锚点；
2. `internal/cli/validation_test.go` 的 CLI 默认值精确相等锚点。

二者只做 `off → auto` 的确定值替换并同步测试名；若需要第三处既有断言变更，本项停止。

## 默认值锚点鉴别力

原始输出：`test-results/0042-negative-matrix/00-default-anchor-force.txt`。

只把 `NormalizeConfig` 与 CLI flag 的生产默认值临时改成第三值 `force`，不改测试：

- `TestNormalizeConfigDefaultsReuseAutoAndRejectsAuditForce`：exit=1，
  `invalid reuse mode "force"`；
- `TestTestCommandWiresExplicitEvidenceFlagsAndDefaultsAuto`：exit=1，
  `unsupported test reuse mode: force`。

恢复后两文件相对 staged baseline 的 `git diff --exit-code=0`，两条精确相等测试均 PASS。
因此锚点不是“任意值”“非空”或永远通过。

## 八项负例矩阵

| # | 负例 | 预期 | 实测与原始输出 |
|---:|---|---|---|
| 1 | tracked 源文件变化 1 字节 | miss，真实执行，不命中 | `01-source-byte-change-smoke.txt` 与同名目录：注释只增加一个 ASCII `!`；250,468ms，executed，ratio=0，subject=dirty，ReceiptID 空，51 PASS / 22 SKIP |
| 2 | PATH 解析到不同 Go/Git 版本 | toolchain identity 变化并 miss | `02-toolchain-path-version-miss.txt`：真实 `go.exe`/`git.exe` identity 命中；PATH 前置代理报告 Go/Git 9.99 后 toolchain digest `3f056e… → c1f68a…`，同 Tree check exit=1 / `VALIDATION_RECEIPT_MISS` |
| 3 | dirty 工作树 | 不查询、不发布可被后续误命中的 Receipt | `03-dirty-no-poison.txt` 与 `03-clean-after-dirty/`：第 1 项 ReceiptID 空；恢复一字节后 333ms 命中原 seed Receipt `1ba3af…`、ratio=1，证明 dirty 运行未污染 store |
| 4 | 显式 `--reuse off` | 强制全量执行 | `04-explicit-off-smoke.txt` 与同名目录：先证明 matching Receipt hit，再显式 off；291,915ms，executed，ratio=0，51 PASS / 22 SKIP |
| 5 | `--force` | 忽略命中并真实执行 | `05-force-unchanged.txt`：`TestRunCreatesReusesForcesAndAuditsReceipt` PASS；同一 temp repo 内先命中，再断言 force 后 executionMode=executed 且执行计数递增 |
| 6 | Receipt store 损坏 | fail-closed 非零，不覆盖损坏 Receipt | `06-corrupt-store-fail-closed.txt`：查询时损坏为 inner exit=1 / `VALIDATION_RECEIPT_INVALID` / 执行数=0 / 原字节保持；普通 miss 后发布时才发现孤儿损坏目录为 inner exit=1 / `VALIDATION_STORE_ERROR` / whole Receipt 未写出 |
| 7 | 已有同 profile Receipt 后引入真实失败 | 必须 FAIL，不得判绿；临时断言原字节恢复 | `07-regression-not-masked.txt` 与同名目录：`SlowestCases` 精确断言临时 `5 → 6`；185,135ms，executed，ratio=0，GO-001 exit=1，整体 FAIL，ReceiptID 空。恢复后 SHA-256=`95FA…ADBC`、blob=`a3d4ad0…`、`git diff --exit-code=0`，目标测试 PASS |
| 8 | 跨 profile | Smoke Receipt 不得被 Full/Release 命中 | `08-cross-profile-isolation.txt`：同一 Tree 上 Smoke check hit/exit=0；Full 与 Release 均 `VALIDATION_RECEIPT_MISS`/exit=1，三者 identity 不同 |

以上文件均位于 `test-results/0042-negative-matrix/`。第 7 项的破坏后辅助
`validation check` 因 tracked worktree change 返回 `VALIDATION_SUBJECT_NOT_REUSABLE`，这是
预期拒绝；破坏前同一 Receipt 的命中由第 3/4 项原始输出直接证明。

## Release 冷/热对照

最终实现的同一 staged Tree `68a2fc3a23bead9b3db0c9ff21e0abe7da943e23` 未改任何 tracked
字节，连续运行两次无显式 `--reuse` 的 Release：

| 运行 | 结论 | execution mode | cache hit ratio | 实际命令墙钟 | Receipt |
|---|---|---|---:|---:|---|
| 冷 | 73 PASS / 0 FAIL / 0 WARN / 0 SKIP | executed | 0 | 217,369ms | `sha256:c4fc312…1a95a` |
| 热 | 同上 | reused | 1 | 344ms | 同一 Receipt |

冷报告：`test-results/0042-final-release-cold/summary.json`；热报告：
`test-results/0042-final-release-warm/summary.json`。复用视图有意保留原始 evidence 的
`summary.duration_ms=217369`；第二次命令的真实返回墙钟取 CLI `elapsedMs=344`。

## CI 命令不变证明

原始对照：`test-results/0042-negative-matrix/09-ci-command-proof.txt`。P0-a commit
`4e475c4` 与当前工作树的两行命令大小写敏感逐字相同：

- `run: .\bin\aicoding.exe test --profile Release --reuse off --json`，
  两边 SHA-256 均为 `546fcd29…05c0`；
- `run: .\bin\aicoding.exe test --profile Release --verify-reuse --json`，
  两边 SHA-256 均为 `2ce56856…2cb`。

帮助文本实测为 `auto|off (default "auto")`。

全仓 `reuse off` / `默认.*off` 复核后，当前文档中的剩余项都属于显式冷执行、miss 后重跑、
CI seed、回滚动作或报告 schema 的 ratio 定义；CHANGELOG、已完成 TODO、旧 spec PLAN 与
BUDGET §1–§13 中的旧默认值均保留为带时间语境的历史证据，§13 末尾已明确由 §15 完成后续
晋级。README 无旧默认值描述，未发现第三处默认值测试锚点。

## 完成判据

- ADR 0014 与代码同批提交，并登记三次远端 run URL、`toolchainDigest.v2`、auto 边界、
  可判定回滚条件、CI 不变证明及两个唯一规格锚点。
- 两个锚点以第三值 `force` 真破坏时均失败，恢复后精确断言通过。
- 八项负例全部真跑并保留原始输出；第 7 项失败则不得翻转。
- 默认 `auto` Release 冷/热两次结论一致，第二次 `cache_hit_ratio > 0`。
- BUDGET、帮助、当前架构/命令/运维文档同步；DocSync、governance、plan、todolist 全绿。
- 最终 exact commit Release 全绿、工作区干净，并由 pre-push Receipt 门禁放行。

最终 exact commit 的 Release summary 固定保存于
`test-results/0042-final-release-exact/summary.json`；该次显式 `--reuse off` 全量运行用于
为第二笔提交绑定 pre-push 所需的 exact-tree Receipt。
