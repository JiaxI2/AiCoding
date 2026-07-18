# AiCoding 全局功能测试计划

## 1. 测试目标

本计划用于验证 AiCoding 仓库在“功能收敛、Go Fast Path、C99 skill、文档同步、Git 治理、生命周期/外部 skill 工作流、Release gate”上的实际可用性。

核心问题：

1. **功能是否收敛到 Go CLI 正式控制面**：lifecycle/doctor/verify/test/release 是否职责清晰，Smoke/Full/Release 是否只由唯一 test engine 执行。
2. **C 语言 skill 是否可验证风格一致性**：`c99-standard-c` 的 status、templates、fmt/check、C UserStyle Kit fast verify、`.clang-format` 投影与 source-of-truth 是否一致。
3. **下载安装外部 skill / 创建用户 skill 相关流程是否规范**：kit registry、manifest、lifecycle plan、export、rollback/fresh-clone 路径是否可检查。
4. **Go 并发和核心对象是否可靠**：ExecutionPlan 的不可变选择、snapshot/digest、runner 并发、race 检查和 CLI 并发只读调用是否稳定。
5. **README 与文档是否同步**：README/README_CN/README_EN、COMMANDS、C99 skill 文档、DocSync gate 是否对齐。
6. **Git 仓库治理是否可执行**：hook、repo-text、release-notes、tag audit、governance lint 是否可重复执行并输出 JSON。
7. **测试结果是否可交付给用户检查**：输出标准 Markdown 报告、JSON 结果、原始 stdout/stderr。

## 2. 测试原则

- **官方入口统一在 Go CLI**：`bin\aicoding.exe test --profile Smoke|Full|Release --json` 和 `bin\aicoding.exe test latest`。
- **三档语义冻结**：Smoke/Full 不启动可见外部工具，Release 才执行真实桌面回归；演进规则见 [契约冻结与获取/激活边界](../../architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md)。
- **所有命令必须有超时**：测试驱动使用 `context.WithTimeout`，禁止无限等待。
- **优先非破坏性测试**：默认只跑 plan/check/gate/export，不直接修改用户全局状态。
- **rollback 只读验证**：测试 profile 只运行 `lifecycle rollback --scope kit --help`，不读取或应用本地 rollback snapshot。
- **结果可追溯**：每个用例保留原始 stdout/stderr、耗时、退出码、判定依据。
- **数据化输出**：统计总用例、通过、失败、告警、跳过、总耗时、各命令耗时。
- **标准 Markdown 文档**：测试计划、测试用例、报告均使用 `.md`。
- **全局分功能框架**：用例按照 ENV/BOOTSTRAP/GO/C99_SKILL/DOCSYNC/LIFECYCLE/EXPORT/FRESH_CLONE/README_DOCS/GIT_GOVERNANCE/PWSH_BOUNDARY/RELEASE_GATE 分类。

## 3. 测试层级

| 层级 | 说明 | 代表用例 |
|---|---|---|
| L0 静态治理 | 不执行仓库命令，只检查文件、配置、文档、registry | README、typed command catalog、registry digest、C99 skill config |
| L1 快速命令 | 执行基础 CLI 命令，验证 JSON 和退出码 | bootstrap、doctor、verify、`test --profile Smoke` |
| L2 功能门禁 | 执行唯一 Registry 中的功能域 gate | docsync、governance、export、lifecycle plan |
| L3 并发/一致性 | 验证 ExecutionPlan 稳定摘要、并发执行只读命令或 race 检查 | runner/catalog unit tests、`go test -race`、并发 C99 status/templates |
| L4 发布门禁 | Release profile 与 fresh-clone leaf probe | Release profile、fresh-clone Full/Release |

## 4. 测试数据

测试驱动记录以下数据：

| 字段 | 含义 |
|---|---|
| `id` | 用例编号 |
| `category` | 功能域 |
| `title` | 用例名称 |
| `status` | PASS/FAIL/WARN/SKIP |
| `severity` | REQUIRED/WARN/OPTIONAL |
| `duration_ms` | 用例耗时 |
| `exit_code` | 命令退出码 |
| `timed_out` | 是否超时 |
| `command` | 实际命令 |
| `stdout_file` | stdout 保存路径 |
| `stderr_file` | stderr 保存路径 |
| `reason` | 判定原因 |
| `json_valid` | 是否成功解析 JSON 输出 |
| `profile` | smoke/full/release/manual |

## 5. 通过标准

### 5.1 Smoke 通过标准

- ENV required 全部通过。
- bootstrap 成功生成 `bin/aicoding.exe`。
- `doctor --all --json` 和 `verify --profile Smoke --json` 成功或仅产生可解释 warning。
- `test --profile Smoke --json` 成功。
- C99 status/templates 与 C UserStyle Kit fast verify 成功。
- README/COMMANDS/C99 文档存在且包含必要入口。
- Git hooks/repo-text 至少可执行或给出明确失败原因。

### 5.2 Full 通过标准

- Smoke 全部通过。
- `go test ./...` 成功。
- DocSync `ci` 成功。
- lifecycle install/update/uninstall plan 成功。
- export 成功。
- governance lint/tag audit 成功。
- PowerShell budget 检查通过或输出可解释 WARN。
- Go 并发只读测试通过。

### 5.3 Release 通过标准

- Full 全部通过。
- `test --profile Full --json` 成功，且不经旧位置参数入口。
- `test --profile Release --json` 成功，且 Release gate 不递归回调 test CLI。
- fresh-clone Release 路径成功，或因网络/远程访问失败产生可解释 WARN。
- release notes/tag policy/release policy 对齐检查通过。

## 6. 风险与边界

| 风险 | 处理 |
|---|---|
| 本机未安装 Task | 标记 WARN，不影响 Go CLI 主路径 |
| `go test -race` 受 CGO/系统工具链影响 | 标记 WARN，但保留日志 |
| fresh-clone 需要网络 | 失败时标记 WARN/FAIL 取决于 profile |
| lifecycle install/update/uninstall 可能改用户状态 | 默认只执行 plan |
| `.exe` 仅 Windows | Linux/macOS 自动查找 `bin/aicoding` |
| 子模块未初始化 | bootstrap/governance/docsync 可能失败，报告中保留证据 |

## 7. 用户检查重点

测试后请优先查看：

1. `report.md` 顶部摘要：失败/告警数量。
2. `C99_SKILL` 区域：是否能证明 C/H 风格检查可重复执行。
3. `DOCSYNC` 区域：README 与文档同步是否真正通过。
4. `LIFECYCLE` 区域：外部 skill/kit plan 是否能稳定输出 JSON。
5. `GO` 区域：并发只读命令是否稳定，race 是否可运行。
6. `PWSH_BOUNDARY` 区域：PowerShell 是否仍超出专项边界。

## 8. README / leaf skill 文档边界

README.md 只作为顶层入口文档，测试只要求它索引稳定 hub 文档，例如 `docs/COMMANDS.md`。具体 leaf skill 文档，例如 `docs/guides/C99_STANDARD_C_SKILL.md`，由 `DOCS-005` 和 `docs/COMMANDS.md` 覆盖，不要求 README 逐个列出。
