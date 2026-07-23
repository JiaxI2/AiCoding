# TODO 0041: 发布流程只执行 Release

Status: Done
Verify: bin/aicoding.exe test --profile Release --reuse off --out test-results/0041-final-release --json && bin/aicoding.exe docsync all --json

## 范围

只删除同一棵树发布流程中重复的 Full 步骤；`test --profile Full` 命令、profile 定义、leaf、
Severity、raceScope 与测试实现均不修改。若直接证据出现 Full-only leaf、Full 更严格，或
共同执行 leaf 的实际 Command 不同，则不得删除 Full 步骤。

## 删除前的直接证据

对照输入是同一 Tree `d339e9a5f93349ced376d9bd0edc639ba4afd137` 的两份真实报告：

- Full：`test-results/0040-final-full/results.json`，69 PASS / 4 SKIP，duration 389,080ms；
- Release：`test-results/0040-final-release/results.json`，73 PASS / 0 SKIP，duration 310,651ms。

对 73 个 leaf 按 ID 对齐，选中状态以实际结果是否为 `SKIP` 判定；Severity 与 Command
直接取两份 `results.json`。PowerShell 使用大小写敏感 `-ceq` 比较原始字符串，没有做路径
或空白规范化。表中的 `""` 表示报告里实际 Command 是空字符串（static leaf）；未选中的
Full leaf 不存在实际执行 Command，因此记为“未选中”。

汇总：

- Full selected=69，Release selected=73；
- Full-only=0，Release-only=4（DOC-003、EXP-001、FRESH-001、FRESH-004）；
- Severity mismatch=0；
- 69 个共同执行 leaf 的 Command mismatch=0。

| Leaf | Full 选中 | Release 选中 | Severity（Full / Release） | Full 实际 Command | Release 实际 Command | 共同执行时逐字相同 |
|---|---|---|---|---|---|---|
| ENV-001 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| ENV-002 | 是 | 是 | REQUIRED / REQUIRED | `go version` | `go version` | 是 |
| ENV-003 | 是 | 是 | REQUIRED / REQUIRED | `git --version` | `git --version` | 是 |
| ENV-004 | 是 | 是 | WARN / WARN | `task --version` | `task --version` | 是 |
| ENV-005 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| BOOT-002 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe bootstrap --no-build --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe bootstrap --no-build --json` | 是 |
| BOOT-003 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| GO-001 | 是 | 是 | REQUIRED / REQUIRED | `go test ./...` | `go test ./...` | 是 |
| GO-002 | 是 | 是 | WARN / WARN | `go test -race ./internal/kit ./internal/mcpcontrol ./internal/report/tokenusage ./internal/runner ./internal/testengine ./internal/validationevidence` | `go test -race ./internal/kit ./internal/mcpcontrol ./internal/report/tokenusage ./internal/runner ./internal/testengine ./internal/validationevidence` | 是 |
| GO-003 | 是 | 是 | WARN / WARN | `go vet ./...` | `go vet ./...` | 是 |
| GO-004 | 是 | 是 | REQUIRED / REQUIRED | `concurrent read-only CLI calls x4` | `concurrent read-only CLI calls x4` | 是 |
| GO-005 | 是 | 是 | WARN / WARN | `go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...` | `go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...` | 是 |
| GO-006 | 是 | 是 | REQUIRED / REQUIRED | `go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...` | `go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...` | 是 |
| GO-007 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| C99-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c status --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c status --json` | 是 |
| C99-002 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c templates --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c templates --json` | 是 |
| C99-003 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c check --scope paths --path testdata/style-samples/foc_sample.c --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c check --scope paths --path testdata/style-samples/foc_sample.c --json` | 是 |
| C99-004 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c check --scope staged --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c check --scope staged --json` | 是 |
| C99-005 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| C99-006 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| C99-007 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c verify --depth fast --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill c99-standard-c verify --depth fast --json` | 是 |
| C99-008 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| SKILL-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill verify --all --profile Smoke --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe skill verify --all --profile Smoke --json` | 是 |
| DOC-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe docsync ci --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe docsync ci --json` | 是 |
| DOC-002 | 是 | 是 | WARN / WARN | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe docsync all --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe docsync all --json` | 是 |
| DOC-003 | 否 | 是 | REQUIRED / REQUIRED | —（未选中） | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe docsync release --json` | 不适用（Release-only） |
| DOC-004 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| LIFE-001 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| LIFE-002 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| LIFE-003 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action install --scope kit --all --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action install --scope kit --all --json` | 是 |
| LIFE-004 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action update --scope kit --all --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action update --scope kit --all --json` | 是 |
| LIFE-005 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action uninstall --scope kit --all --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action uninstall --scope kit --all --json` | 是 |
| LIFE-006 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle rollback --scope kit --help` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle rollback --scope kit --help` | 是 |
| LIFE-007 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe kit verify --all --profile Lifecycle --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe kit verify --all --profile Lifecycle --json` | 是 |
| MCP-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe mcp list --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe mcp list --json` | 是 |
| EXP-001 | 否 | 是 | REQUIRED / REQUIRED | —（未选中） | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe export --all --zip --json` | 不适用（Release-only） |
| EXP-002 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FRESH-001 | 否 | 是 | REQUIRED / REQUIRED | —（未选中） | `git archive validation subject + release verify` | 不适用（Release-only） |
| FRESH-003 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FRESH-004 | 否 | 是 | WARN / WARN | —（未选中） | `""` | 不适用（Release-only） |
| DOCS-001 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| DOCS-002 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| DOCS-003 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| DOCS-004 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| DOCS-005 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| DOCS-006 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| CAP-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance capabilities --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance capabilities --json` | 是 |
| FREEZE-001 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-002 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-003 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-004 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-005 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-006 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-007 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-008 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| FREEZE-009 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| GIT-001 | 是 | 是 | WARN / WARN | `git status --short` | `git status --short` | 是 |
| GIT-002 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe verify hooks --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe verify hooks --json` | 是 |
| GIT-003 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe verify repo-text --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe verify repo-text --json` | 是 |
| GIT-004 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe verify release-notes --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe verify release-notes --json` | 是 |
| GIT-005 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance lint --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance lint --json` | 是 |
| GIT-006 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe tag audit --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe tag audit --json` | 是 |
| GIT-007 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| GIT-008 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance layout --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance layout --json` | 是 |
| GIT-009 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance reuse --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe governance reuse --json` | 是 |
| PWSH-001 | 是 | 是 | WARN / WARN | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe doctor pwsh --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe doctor pwsh --json` | 是 |
| PWSH-002 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe doctor pwsh-budget --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe doctor pwsh-budget --json` | 是 |
| PWSH-003 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| HEALTH-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe doctor perf --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe doctor perf --json` | 是 |
| RC-001 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle verify --scope repo-context --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle verify --scope repo-context --json` | 是 |
| RC-002 | 是 | 是 | REQUIRED / REQUIRED | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action install --scope repo-context --json` | `F:\Study\AI\worktrees\AiCoding\bin\aicoding.exe lifecycle plan --action install --scope repo-context --json` | 是 |
| ADR-001 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |
| REL-002 | 是 | 是 | REQUIRED / REQUIRED | `""` | `""` | 是 |

## `cfg.Profile` 全量排查

命令：

```powershell
rg -n "cfg\.Profile" --glob '*.go' --glob '!CodingKit/**' .
```

共 40 处：测试夹具 13 处；evidence/node-evidence 的 profile 选择、Receipt 分区与报告投影
12 处；engine 的解析、选择与报告字段 14 处；唯一会改变 leaf Command 的生产分支是
`internal/testengine/race_scope.go:28`。该分支条件是
`cfg.Profile == ProfileFull || cfg.Profile == ProfileRelease`，两者读取同一
`config/impact-policy.json` 六包有序 scope。真实报告中的 GO-002 Command 也已在上表逐字
相同，因此没有隐藏的 Full 更严格命令。

## 发布流程规定全仓审计

搜索范围为 `docs/`、`tools/`、`Taskfile.yml`、`.agents/` 与 CodingKit 自有文档，排除
`CodingKit/agents/skills` 和 `_external/Mklink-AI-Probe` 两个只读子模块。

| 位置 | 裁决 |
|---|---|
| `docs/operations/MAINTENANCE_METHOD.md` | 删除连续 Full 步骤，只保留 Release，并链接本证据。 |
| `docs/architecture/CODEX_KIT_ARCHITECTURE.md` | 将开发迭代的 Smoke/Full 与发布 Release 拆开，禁止读成连续发布步骤。 |
| `docs/architecture/POWERSHELL_BOUNDARY.md` | 保留三个独立 CLI 入口，明确发布只调用 Release。 |
| `docs/operations/testing/LATEST_VALIDATION.md` | 当前命令清单删除 Full；旧 Full summary 行作为历史快照保留并明确不是现行发布步骤。 |
| `Taskfile.yml`、`docs/COMMANDS.md` | 仅提供彼此独立的 `task full` 与 `task release` 路由，不聚合，不修改。 |
| `.agents/` | Full 仅用于开发/仓库验证或 profile 占位，没有“先 Full 再 Release”发布编排，不修改。 |
| CodingKit 自有文档 | 只描述 Full/Release 能力差异或上层验收归属，没有连续发布编排，不修改。 |
| 历史 ADR、spec、evidence 与已归档 TODO | 保留当时执行事实；ADR 0013 §4 已明确历史计划不回写。 |

## 验收记录

- 新流程真跑：`bin/aicoding.exe test --profile Release --reuse off --out
  test-results/0041-final-release --json`。
- 固定 summary：`test-results/0041-final-release/summary.json`。
- 结果：73/73 PASS、0 FAIL、0 WARN、0 SKIP，`execution_mode=executed`，
  `cache_hit_ratio=0`，引擎 duration **314,523ms**。
- 旧流程基线：Full 389,080ms + Release 310,651ms = **699,731ms**。
- 新流程相对旧流程减少 **385,208ms（55.05%，约 6.42 分钟）**；本轮 Release
  相对旧 Release 单次测量增加 3,872ms，属于运行波动，不改变去重判据。
- Release 内的 DOC-001/002/003 全部 PASS；独立 `docsync all`、governance
  dependencies/lint、plan verify、todolist 与 `git diff --check` 在提交前复验。
