# TODO 0015: 工具链加固（补 lint/漏洞扫描空白 + CI 版本对齐 + 供应链钉死）

Status: In-Progress
Verify: bin/aicoding.exe test --profile Full --json 含 lint 节点全绿；CI 与本地实际 Go 版本均为 go1.26.5

> 工具链评审结论（2026-07-20）：**现有选型全部正确，不换任何东西** ——
> Task 只做薄路由（正确用法）、Go 单外部依赖（BurntSushi/toml，极简）、
> PowerShell 面已冻结在退役轨道、lychee 管链接、clang-format 钉版本。
> 真正的问题是**四个空白**，不是选错工具。

## 实测出的四个空白

| # | 空白 | 实证 | 风险 |
|---|---|---|---|
| 1 | **除 go vet 外零 lint** | 全仓 grep 无 golangci/staticcheck/gofumpt | 长期开发的最大缺口：~25 个包靠 vet 兜底 |
| 2 | **零漏洞扫描** | 无 govulncheck | 依赖虽少，stdlib CVE 仍需跟踪 |
| 3 | **CI 与本地 go 版本漂移** | CI 三处硬编码 `go-version: '1.22'`，原本地 go1.26.4，go.mod 无 toolchain 行 | toolchain digest 已入 Receipt 身份——**本地与 CI 的 Receipt 互不可比**，release-gate 三绿证据的可信度打折 |
| 4 | **Actions 按 tag 引用** | `actions/checkout@v4` 等 | 供应链：tag 可被上游移动 |

## 实现计划

1. **staticcheck 而非 golangci-lint**（先验收窄：单工具、零配置起步，符合"约定优于配置"；
   golangci-lint 的聚合配置面太大，等出现第二个真实需求再评估）：
   - 固定版本装入 CI 与本地（`go run honnef.co/go/tools/cmd/staticcheck@<pinned>`）；
   - testengine 注册 `GO-005 staticcheck`（Full/Release profile，WarnOnly 起步，
     一个 release 后升 Required——给存量问题一个清理窗口）；
   - 存量告警一次性清零或逐条 `//lint:ignore` 附理由，**不允许无理由压制**。
2. **govulncheck**：注册 `GO-006`（Full/Release，Required——漏洞不给宽限期）；
   CI 的 schedule 任务（已有 cron）顺带跑，网络失败时 Warn 不 Fail（环境因素豁免）。
3. **go 版本对齐**：因 GO-2026-5856 安全修复，go.mod 增 `toolchain go1.26.5` 行；CI 曾尝试三处
   `go-version-file: 'go.mod'`，但远端 workflow_dispatch #29798713550 的 setup-go 日志明确显示
   `Setup go version spec 1.22`，只读取了 `go 1.22` 而非 toolchain 行。因此三处退回显式
   `go-version: '1.26.5'`。**已知欠账：版本单源未达成，退化为双处维护。**
4. **Actions 钉 SHA**：四个 uses 全部改 `@<full-sha> # vN` 注释保留可读性。
5. **可选（工时富余才做）**：CI 加一个 ubuntu smoke job（仅 build + Smoke），
   抓 `.githooks/*` sh 脚本与路径分隔符回归——Windows 第一等不变，linux 是烟雾探测。

## 明确不做

- 不引入 golangci-lint 全家桶 / 不写 .golangci.yml（先验最小化）。
- 不换 Task（现用法正确；just/make 无增量收益且 make 在 Windows 是二等公民）。
- 不加 pre-commit lint（lint 属 Full 半径；pre-commit 预算毫秒级，已被 0004/0014 占满）。
- 不引入 dependabot/renovate（1 个依赖 + 4 个 action，人工季检足够）。

## 自测（可信任方式）

```powershell
go test ./internal/testengine/...
# 负例：临时写一段 staticcheck 必报的死代码 → GO-005 必须 Warn → 撤销
bin\aicoding.exe test --profile Full --json          # GO-005/GO-006 出现且结论符合预期
# 版本单源验证：
go version ; grep "^toolchain" go.mod                # 一致
grep -n "go-version" .github/workflows/aicoding-ci.yml   # 三处均为显式 1.26.5
grep -n "uses:" .github/workflows/*.yml              # 全部 @sha
# Receipt 身份连续性：toolchain 行加入后旧 Receipt 应失效（toolchain digest 变化）——
bin\aicoding.exe validation check --profile Release --target HEAD --json   # 预期 MISS，属正确失效
bin\aicoding.exe test --profile Smoke --json
```

通过判据：staticcheck 负例被抓（贴输出）；govulncheck 在 Full 中执行；CI 工作流实测使用
go1.26.5；四个 action 均为 SHA；toolchain 变化正确导致旧 Receipt 失效（这是特性不是回归，
CHANGELOG 说明）。
