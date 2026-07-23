# TODO 0039: GO-001 执行时间收敛

Status: Done
Verify: go test ./... -count=1 && bin/aicoding.exe test --profile Full --reuse off --out test-results/0039-final-full-green --json

## 范围

本项只优化测试夹具与安全并行度，不修改产品行为、测试断言、用例数量或 Test Registry：

1. `internal/cli` 的外部 CLI 路由测试由 `TestMain` 一次 `go build`，测试执行构建产物，
   构建失败时在测试开始前 fail-fast 并输出明确错误。
2. `internal/cli`、`internal/kit`、`internal/validationevidence`、`internal/testengine`
   的 `TestMain` 各初始化一个只读空 Git 模板；需要 Git 仓库的测试复制到自己的
   `t.TempDir`，不共享可写状态。
3. `t.Parallel()` 只用于拥有独立 temp dir、未调用 `os.Chdir`、不写环境变量且不修改包级
   注入点的测试。会替换 `runTestEngine`、`runPinFetch`、hook 输入或 Codex 环境的测试保留串行。

## 未优化基线（`main@603d866`）

2026-07-23 在未改代码、`-count=1` 且每包测量前清空 test cache 的同机基线：

| 包 | Go 报告时间 | 外部墙钟 |
|---|---:|---:|
| `internal/cli` | 43.390s | 46,485.750ms |
| `internal/kit` | 28.913s | 31,737.011ms |
| `internal/validationevidence` | 26.792s | 28,416.990ms |
| `internal/testengine` | 27.235s | 28,886.685ms |

同一未改代码树的 `test --profile Full --reuse off` 固定输出为
`test-results/0039-baseline-full/summary.json`：`73 total / 69 pass / 0 fail /
0 warn / 4 skip`，总报告 478,072ms，GO-001 `go test ./...` 为 260,147ms
（execute 260,139ms），与提示中的 262,049ms 同量级。

`go test -v -json` 的 PASS 计数基线：

| 包 | PASS events | 顶层 PASS |
|---|---:|---:|
| CLI | 99 | 78 |
| Kit | 49 | 47 |
| Validation Evidence | 30 | 25 |
| Test Engine | 61 | 52 |

## 优化后局部证据

Go 1.22 兼容的最终实现以相同四包命令复测：

| 包 | 外部墙钟 | 相对基线 |
|---|---:|---:|
| `internal/cli` | 25,917.624ms | -20,568.126ms |
| `internal/kit` | 20,185.531ms | -11,551.480ms |
| `internal/validationevidence` | 10,479.900ms | -17,937.090ms |
| `internal/testengine` | 23,927.079ms | -4,959.606ms |

优化后 PASS events/顶层 PASS 仍分别为 `99/78`、`49/47`、`30/25`、`61/52`，
与基线逐包完全一致。Test Engine 的长测试因使用包级注入点而有意保持串行，没有为追求
耗时减少断言或改产品代码。

最终代码连续三次 `go test ./... -count=1` 均 exit 0，墙钟为
`35,650.800 / 34,154.661 / 35,216.496ms`；实际测试执行稳定低于 50 秒。

## Full 对照与兼容性纠正

- 首次优化后 Full 运行保留在 `test-results/0039-final-full/summary.json`，GO-001 已降至
  115,931ms，但 GO-003 原始输出指出 `os.CopyFS requires go1.23 or later (module is go1.22)`；
  该次 `PASS_WITH_WARNINGS` 不作为验收结果。
- 将模板复制最小改为 Go 1.22 可用的 `filepath.WalkDir`、`os.ReadFile` 与 `os.WriteFile`
  后，`go vet ./...` exit 0，未修改产品代码或测试语义。
- 纠正后的 Full 固定输出为 `test-results/0039-final-full-green/summary.json`：
  `73 total / 69 pass / 0 fail / 0 warn / 4 skip`，总报告 235,556ms；
  GO-001 为 135,525ms（execute 135,516ms），相对同树优化前 260,147ms 减少
  124,622ms（47.9%）。GO-001 仍包含测试二进制编译，独立的实际测试执行三次均低于 50 秒。
- Receipt 为
  `sha256:161c6b6ec5c11cdb9a168c84d4aef08a60250c60cc39547bade60247f23ff5a7`，
  subject tree 为 `2bab91677efc553d4f9ccad089952d2bd842dcec`。

## 验收结果

- `go test ./... -count=1` 全绿，四包 `-v` PASS 计数逐项不变，连续三次结论稳定。
- docsync all、governance dependencies/lint、plan verify、todolist 与 capability 投影全绿。
- 同批 CHANGELOG、正常 hooks 提交；未使用 `--no-verify`。
