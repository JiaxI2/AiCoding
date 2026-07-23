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
- **证据保留有界且不毁证**：成功运行写入报告后自动保留最近 5 份，并额外保留全部 FAIL；失败或取消的运行不触发清理。validation report 只有在无 Receipt/alias 引用时才可清理。
- **CI 失败可诊断**：GitHub Actions 的 Smoke 与 `release-gate` 无论结论如何均上传本次 `test-results/` artifact，保留逐用例原始输出。
- **验证绑定内容**：成功运行可把 PASS 结论绑定到 Git Tree 与验证语义；commit message amend 不失效，tracked/untracked/submodule 脏状态 fail-closed。
- **复用可审计**：默认保持 `--reuse off`；`--reuse auto` 只显式启用。main 远端 `release-gate` 连续 3 次完成 off seed + `--verify-reuse` audit，并在独立切换提交引用三次 run URL 后，才允许晋级默认值。workflow 已接线不等于已跑绿。
- **节点复用是私有加速层**：整树 Receipt miss 后可按 Registry 节点复用；节点 Receipt 不进入 alias、push gate 或公共 CLI 列表，整树 Receipt 仍是唯一外部凭证。
- **工具链安全**：Full/Release 固定运行 Staticcheck v0.7.0 与 govulncheck v1.6.0；真实漏洞保持 REQUIRED，只有可识别的网络访问失败可降级为 WARN。
- **race 降频不降级**：Full 的 GO-002 只跑 `impact-policy.json` 登记的并发包，GO-007 以 AST 门禁阻断漏登；Release 与每周 schedule 仍跑全仓 race。
- **测试夹具并行边界**：CLI、Kit、Validation Evidence 与 Test Engine 的包测试各由
  `TestMain` 一次建立只读 Git 模板，测试复制到自己的 `t.TempDir`；只有不调用
  `os.Chdir`、不写进程环境且不修改包级注入点的测试可使用 `t.Parallel()`。CLI 外部路由
  测试复用 `TestMain` 一次构建的二进制，构建失败在执行用例前 fail-fast。
- **数据化输出**：统计总用例、通过、失败、告警、跳过、总耗时、各命令耗时。
- **标准 Markdown 文档**：测试计划、测试用例、报告均使用 `.md`。
- **全局分功能框架**：用例按照 ENV/BOOTSTRAP/GO/C99_SKILL/DOCSYNC/LIFECYCLE/EXPORT/FRESH_CLONE/README_DOCS/GIT_GOVERNANCE/PWSH_BOUNDARY/RELEASE_GATE 分类。

## 3. 测试层级

| 层级 | 说明 | 代表用例 |
|---|---|---|
| L0 静态治理 | 不执行仓库命令，只检查文件、配置、文档、registry | README、typed command catalog、registry digest、C99 skill config |
| L1 快速命令 | 执行基础 CLI 命令，验证 JSON 和退出码 | bootstrap no-build、doctor、verify、`test --profile Smoke` |
| L2 功能门禁 | 执行唯一 Registry 中的功能域 gate | docsync、governance、export manifest、lifecycle plan |
| L3 并发/一致性 | 验证 ExecutionPlan 稳定摘要、并发执行只读命令或 race 检查 | runner/catalog unit tests、`go test -race`、并发 C99 status/templates |
| L4 发布/定期 hermetic 门禁 | Release profile 的真实打包与 Git-object 物化 leaf；定期 CI 的真实 clean-clone Full | Release ZIP、materialized Release、每周 fresh-clone Full |

## 4. 测试数据

测试驱动记录以下数据：

| 字段 | 含义 |
|---|---|
| `id` | 用例编号 |
| `category` | 功能域 |
| `title` | 中文优先的人读用例名称；必须是无 `U+FFFD` 的有效 UTF-8 |
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
| `TestCase.Node` | Registry 内部节点名；空值保守归一为 `repo`，不新增报告 schema 字段 |
| `executionMode` | `executed` 或 `reused`；不是新的结论状态 |
| `validationIdentity` | Tree、profile、plan、engine、config、toolchain、options 的内容身份 |
| `resultsDigest` | 当前 profile 选中用例的排序 `(id,status)` 摘要；Receipt 与审计逐用例核对 |
| `reusable` | 本次报告能否生成或继续引用 PASS Receipt |
| `reusableReason` | 不可复用的稳定原因，例如 dirty 或执行期间内容漂移 |

### 4.1 节点 Receipt 分组

| 节点 | Registry 用例 | 输入范围与失效语义 |
|---|---|---|
| `go` | GO-001…006 | `*.go`、`go.mod`、`go.sum`、任意 `testdata/`；Go 输入变化即失效 |
| `docsync` | DOC-001/002/004 | 根 README/CHANGELOG、`docs/`、DocSync/Test Registry/报告/CLI 实现及 DocSync kit 配置 |
| `governance` | GIT-002…009 | Go、根文档、`.github/`、`docs/`、`config/` 与根治理文件 |
| `lifecycle-readonly` | LIFE-001…007、RC-001/002 | Go、`config/`、`CodingKit/` 与 `Taskfile.yml` |
| `repo` | 所有未标注用例 | 全部 tracked Tree entry；任何变化都失效 |

路径集合允许重叠，优先保证不出现静默假 PASS。每个可复用主体只执行一次
`git ls-tree -r -z --full-tree`，再从 mode/type/OID/path 批量派生所有节点摘要，不读取工作区文件。
dirty subject 完全禁用节点查询与发布。`--reuse off` 不查询节点，但成功执行后可发布 PASS 节点
Receipt；`--reuse auto` 可部分复用；`--verify-reuse` 始终真实执行并逐节点核对，失败节点永不缓存。

## 5. 通过标准

### 5.1 Smoke 通过标准

- ENV required 全部通过。
- Task `ensure-bin` 通过 checksum 在输入变化时生成 `bin/aicoding.exe`；BOOT-002/003 验证 CLI 与静态前置条件且不重复构建。
- `doctor --all --json` 和 `verify --profile Smoke --json` 成功或仅产生可解释 warning。
- `test --profile Smoke --json` 成功。
- C99 status/templates 与 C UserStyle Kit fast verify 成功。
- README/COMMANDS/C99 文档存在且包含必要入口。
- Git hooks/repo-text 至少可执行或给出明确失败原因。

### 5.2 Full 通过标准

- Smoke 全部通过。
- `go test ./...` 成功。
- GO-002 对登记并发包执行 race，GO-007 证明全仓并发包没有漏登。
- GO-005 Staticcheck 零告警；GO-006 govulncheck 无可达漏洞，网络访问失败须保留可诊断 WARN。
- DocSync `ci` 成功。
- lifecycle install/update/uninstall plan 成功。
- EXP-002 export manifest 静态验证成功，且 Full 不创建 ZIP。
- FRESH-003 fresh-clone/物化契约静态验证成功，且 Full 不复制仓库。
- governance lint/tag audit 成功。
- PowerShell budget 检查通过或输出可解释 WARN。
- Go 并发只读测试通过。

### 5.3 Release 通过标准

- Full 全部通过。
- `test --profile Full --json` 成功，且不经旧位置参数入口。
- `test --profile Release --json` 成功，且 Release gate 不递归回调 test CLI。
- GO-002 的实际命令保持 `go test -race ./...`，不得继承 Full 的 scoped race。
- FRESH-001 从当前验证主体 Tree 和本地递归 submodule 对象物化无 `.git` 源码树，重建 CLI 并
  执行 `release verify`；该 REQUIRED 路径不得出现 `git clone` 或网络获取。
- FRESH-004 在真 clone baseline 之后发现传输敏感路径变化时给出可解释 WARN；该提示不替代
  显式/周期性真 clone。
- release notes/tag policy/release policy 对齐检查通过。

### 5.4 定期 clean-clone Full 标准

- `.github/workflows/aicoding-ci.yml` 的每周 schedule 和手动触发必须运行
  `fresh-clone --profile Full --json`。
- 同一 schedule 必须运行 Release profile；其 GO-002 是周期性的全仓 race 证明。
- 该命令必须在临时递归 clone 中重新构建 CLI 并执行 `go test ./...`，用于提前发现子模块、
  gitignore、干净检出和 Go 构建漂移。
- 这是一条独立的正式 leaf command，不新增 test Registry 聚合器，也不回到日常 Full profile。

## 6. 风险与边界

| 风险 | 处理 |
|---|---|
| 本机未安装 Task | 标记 WARN，不影响 Go CLI 主路径 |
| `go test -race` 受 CGO/系统工具链影响 | 标记 WARN，但保留日志 |
| Release 物化时本地 submodule 对象未初始化或 tar 流非法 | FRESH-001 REQUIRED 失败并保留登记后的 `aicoding-materialize-*` 现场；路径越界 fail-closed |
| 显式/周期性 fresh-clone 需要网络 | 公共命令报告失败并保留登记后的 `aicoding-fresh-clone-*` 现场；不把网络成本绑回 Release Registry |
| lifecycle install/update/uninstall 可能改用户状态 | 默认只执行 plan |
| `.exe` 仅 Windows | Linux/macOS 自动查找 `bin/aicoding` |
| 子模块未初始化 | bootstrap/governance/docsync 可能失败，报告中保留证据 |
| 本地测试报告持续累积 | `doctor.cache-bloat` 只告警；`cache clean --scope test-results --dry-run` 先审计清单，FAIL 永不自动删除 |
| 系统临时目录持续累积 | `doctor.cache-bloat` 对严格小写 `aicoding-*` 超过 20 个或 100MB 只告警；历史孤儿必须先 `cache clean --scope temp --dry-run --adopt` 审计，再显式回收 |

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
