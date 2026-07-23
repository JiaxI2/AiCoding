# TODO 0040: GO-002 race 成本裁决

Status: Done
Verify: go test ./internal/testengine -count=1 && bin/aicoding.exe test --profile Full --reuse off --out test-results/0040-final-full --json && bin/aicoding.exe test --profile Release --reuse off --out test-results/0040-final-release --json

## 待裁决问题

当前 `raceTestCommand` 只在 Full 使用 `config/impact-policy.json` 的六包 `raceScope`；
Release 仍执行 `go test -race ./...`，而 GO-002 为 WarnOnly。必须通过 ADR 0013 二选一：

1. Release 同样使用 raceScope，并记录严格度下降的接受理由与补偿措施；或
2. Release 保持全仓 race，同时把 GO-002 晋升 Required。

不得保留“全仓 + WarnOnly”的现状，不翻转 `--reuse` 默认值，不新增 Registry、runner、
profile、Primitive 或治理领域。

## 实测基线

- Full 基线 `test-results/0039-baseline-full/results.json` 中，scoped GO-002 为
  175,995ms，命令精确包含当前六个登记包。
- 使用独立空 `GOCACHE=test-results/0040-baseline-race-cache` 真跑
  `go test -race ./... -count=1`，exit 0，外部墙钟 **120,656.719ms**。
- 提示中的 Release GO-002 200,736ms 与本轮隔离实测不同；数值按本轮机器与缓存条件留证，
  规则仍按 ADR 二选一执行。

为消除 0039 最终夹具与缓存状态差异，在提交后的同一
`main@07d73b3` 上重新建立两个独立空 `GOCACHE`：

| 模式 | 原始输出 | 外部墙钟 | 结果 |
|---|---|---:|---|
| 全仓 | `test-results/0040-before-global-output-07d73b3.txt` | 78,523.332ms | PASS |
| 六包 scope | `test-results/0040-after-scoped-output.txt` | 59,750.359ms | PASS |

独立 cold 对照节省 18,772.973ms（23.9%）。`-race` 编译产物使用不同于普通 GO-001 的
build-cache key；两个物理空 cache 避免了普通测试或先后顺序的预热影响。

## 裁决与最小实现

ADR 0013 选择方案 a：Release 与 Full 共用既有 `raceScope.packages`。不选择把 GO-002
晋升 Required 的方案 b，因为它保留全部全仓编译成本，不能满足本项性能目标。

严格度变化限定为不再对未登记且未被识别为并发的包做 race instrumentation。补偿仍是既有
Required GO-007：它在 Full/Release 扫描全仓 Go AST，发现 goroutine、channel 或 `sync`
所在包漏登即失败；GO-001 仍对 `./...` 运行全部单元测试。未新增 Primitive、Registry、
schedule、配置字段或治理面。

回归测试先以原实现真实失败并输出
`Release GO-002 command = "go test -race ./..."`，随后只把 profile 条件扩为
Full/Release，目标测试与 `go test ./internal/testengine -count=1` 均通过。

## 最终验收输出

- Full：`test-results/0040-final-full/summary.json`
- Release：`test-results/0040-final-release/summary.json`

两次均要求 `--reuse off` 真实执行；summary、results 与逐 leaf 原始 stdout/stderr 由测试引擎
保留在对应目录。

## 完成判据

- ADR 0013 明确选项、依据、严格度/阻断性权衡、补偿措施和回滚条件。
- 前后 race 使用独立空 build cache 对照，并说明 `-race` 与普通 Go 构建缓存不共享产物。
- 同步 `docs/COMMANDS.md`、test plan/cases、Kit profile 文档、config catalog 与 CHANGELOG。
- Full、Release 各真跑一次并记录固定 summary 路径；共同治理门禁全绿。
- 正常 hooks 独立提交；不使用 `--no-verify`。
