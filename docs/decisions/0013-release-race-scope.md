# ADR 0013: Release 复用登记并发包 raceScope

PrimitiveReview: n/a

## Status

Accepted。本 ADR 只改变既有 GO-002 对既有 `raceScope.packages` 的 profile 选择，不新增
Primitive、Test Registry leaf、配置字段、治理领域或证据权威。

## 1. Context

裁决前 Full 的 GO-002 使用 `config/impact-policy.json` 登记的六个并发包，Release 则执行
`go test -race ./...`；两者的 GO-002 都是 WarnOnly。该组合让 Release 为所有 Go 包重新支付
race instrumentation 的编译与执行成本，却不能凭 GO-002 本身阻断发布。

0022 的历史完成记录和 `loop-engineering-backlog` 已批准实施计划曾要求 Release 与每周 CI
保留全仓 race；那是当时以“周期性全仓证明”补偿 Full 收窄的前向决策，不是 Primitive 或
冻结架构。0040 的 owner 裁决明确重新打开该选择，并要求在“Release 同样收窄”与“全仓但
晋升 Required”之间二选一。

在提交后的 0039 同一代码树 `main@07d73b3` 上，两条命令分别使用独立空 `GOCACHE` 真跑：

| 模式 | 命令 | 墙钟 | 结果 |
|---|---|---:|---|
| 裁决前 | `go test -race ./... -count=1` | 78,523.332ms | PASS |
| 裁决后 | `go test -race <六个 raceScope 包> -count=1` | 59,750.359ms | PASS |

隔离实测节省 18,772.973ms（23.9%）。提示中的 200,736ms 与本机当前实测不同，按规则保留
实测值，不把数值本身升级为契约。`-race` 会改变编译选项和 build ID，因此与普通 GO-001
使用不同的 Go build-cache key 空间；即使共用物理 cache 目录，也不能复用普通测试二进制。
本对照使用两个物理独立的空 cache，避免把预热顺序误写成收益。

## 2. Decision

选择方案 a：Full 与 Release 的 GO-002 都从同一 `raceScope.packages` 投影命令。GO-002
继续保持 WarnOnly；GO-007 继续在 Full/Release 中保持 Required。

不选择方案 b。把 GO-002 晋升 Required 可以提高单个信号的阻断性，但仍保留全仓 race 的
全部成本，不能满足本项收敛 Release 耗时的目标。方案 a 只改变既有 profile 条件，不改变
Registry、runner、报告 schema、severity 或配置结构。

## 3. 严格度变化与补偿

接受的严格度变化是：Release 不再对未登记且未被识别为并发的包进行 race instrumentation。
以下既有机制构成补偿，不新增治理面：

1. GO-007 扫描全仓 Go AST；出现 goroutine、channel 或 `sync` 的包若未登记，Full/Release
   都以 Required 失败。
2. `raceScope.packages` 仍由 schema 闭合、规范化、排序和目录存在性检查保护；当前六包为
   `internal/kit`、`internal/mcpcontrol`、`internal/report/tokenusage`、`internal/runner`、
   `internal/testengine`、`internal/validationevidence`。
3. GO-001 仍在全部 profile 对 `./...` 运行完整单元测试；本裁决不删除测试、不减少断言。
4. `go test -race ./...` 保留为显式诊断命令，但不另建 schedule、Registry leaf 或第二策略面。

若出现以下任一事实，必须重新评审并优先恢复 Release 全仓 race：GO-007 未识别的并发模式导致
漏登、race 缺陷出现在 scope 外包、或 scope 登记不能在新增并发代码时 fail-closed。回滚只需
恢复 Release 的全仓分支与对应回归测试、当前运维文档；配置和 Receipt 无迁移。

## 4. Supersession

本 ADR 自接受提交起向前取代 TODO 0022、其已批准实施计划及当时 CHANGELOG 中
“Release/每周 CI 全仓 race”的运行要求。那些文件是历史计划与执行证据，保持原文以保留
可追溯性；当前命令和运维文档以本 ADR 为准。

## 5. Verification

- 回归测试先证明 Release 仍生成 `go test -race ./...`，再以最小条件修改证明 Full/Release
  都生成同一有序 scope。
- `go test ./internal/testengine -count=1` 与 `go test ./... -count=1` 全绿。
- Full 与 Release 各以 `--reuse off` 真跑；报告保留 GO-002 的实际命令和耗时。
- DocSync、governance dependencies/lint、plan、todolist 与 capability 投影全部通过。

## §12 Checklist 自评

不适用：本 ADR 复用既有 testengine、GO-002、GO-007 和 `impact-policy.json`，没有新增
Primitive、公开接口、注册表、配置方言或治理权威。
