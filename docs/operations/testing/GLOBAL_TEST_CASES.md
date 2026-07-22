# AiCoding 全局测试用例文档

> 本文档是全局测试用例清单。测试驱动会把实际执行结果写入 `report.md` 与 `results.json`。
> 每个测试命令均由 Go runner 加超时执行，禁止不限时运行。

## 1. ENV：基础环境

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| ENV-001 | 仓库根目录识别 | 静态检查 `.git`、`go.mod`、`README.md` | 仓库根目录有效 | REQUIRED |
| ENV-002 | Go 版本 | `go version` | 可执行，Go >= 1.22 | REQUIRED |
| ENV-003 | Git 版本 | `git --version` | 可执行 | REQUIRED |
| ENV-004 | Task 可用性 | `task --version` | 可执行；未安装记 WARN | WARN |
| ENV-005 | 模块路径 | 读取 `go.mod` | module 为 `github.com/JiaxI2/AiCoding` | REQUIRED |

## 2. BOOTSTRAP：Go CLI 前置条件

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| BOOT-002 | CLI 基础可用 | `bin/aicoding.exe bootstrap --no-build --json` | 退出码 0，JSON 合法，不重复构建 | REQUIRED |
| BOOT-003 | bootstrap 前置条件 | 进程内 `bootstrap.Check` | repo、Go、Git、go.mod 与 bin 检查齐全 | REQUIRED |

## 3. GO：Go 单元、并发、race

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| GO-001 | 全仓 Go 单元测试 | `go test ./...` | 退出码 0 | REQUIRED |
| GO-002 | Go race 检查 | Full：`go test -race <raceScope.packages>`；Release：`go test -race ./...` | 退出码 0；环境不支持 race 或历史包不兼容记 WARN；Release 必须保持全仓 | WARN |
| GO-003 | Go vet 基础检查 | `go vet ./...` | 退出码 0 | WARN |
| GO-004 | CLI 并发只读调用 | 并发运行 C99 status/templates/governance lint | 全部退出码 0，无 timeout | REQUIRED |
| GO-005 | Staticcheck 静态分析 | `go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...` | 零告警；首个 release 失败记 WARN | WARN |
| GO-006 | Go 漏洞扫描 | `go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...` | 无可达漏洞；仅可识别的网络失败记 WARN | REQUIRED |
| GO-007 | 并发包 raceScope 登记 | AST 扫描全仓 `.go` 文件，并与 `config/impact-policy.json` 对账 | goroutine、channel 或 `sync` 所在包全部登记；漏登即 Full/Release 失败 | REQUIRED |

## 4. C99_SKILL：C 语言 skill 风格一致性

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| C99-001 | C99 skill status | `bin/aicoding.exe skill c99-standard-c status --json` | 输出 JSON，包含 skill 状态 | REQUIRED |
| C99-002 | 注释模板校验 | `bin/aicoding.exe skill c99-standard-c templates --json` | 模板 JSON 合法，退出码 0 | REQUIRED |
| C99-003 | 样例路径格式检查 | `bin/aicoding.exe skill c99-standard-c check --scope paths --path testdata/style-samples/foc_sample.c --json` | 样例存在时退出码 0 | REQUIRED |
| C99-004 | staged C/H 检查入口 | `bin/aicoding.exe skill c99-standard-c check --scope staged --json` | 退出码 0 或明确无 staged 文件 | REQUIRED |
| C99-005 | source-of-truth 配置 | 检查 `config/skills/c99-standard-c/*` 与 `.clang-format` | 配置文件存在，投影包含关键字段 | REQUIRED |
| C99-006 | 排除目录与自包含 Kit 边界策略 | 解析 `skill.json` 并执行 Go 回归测试 | 包含常规目录名排除项与 `CodingKit/tools/c-userstyle-kit` 仓库相对路径排除项，且不误排其他 `tools` 内容 | REQUIRED |
| C99-007 | C UserStyle Kit 快速验证 | `bin/aicoding.exe skill c99-standard-c verify --json` | fast profile 成功，输出统一 JSON，且不调用固件工具链 | REQUIRED |
| C99-008 | C Kit 资产与参考完整性 | 检查 kit manifest、黄金/高级样例、规则目录、snippets、PDF 和 Markdown 参考 | 资产存在，manifest version 为 1.2.0 | REQUIRED |
| SKILL-001 | 全部启用 Skill Smoke 验证 | `bin/aicoding.exe skill verify --all --profile Smoke --json` | 统一 Skill registry 验证通过 | REQUIRED |

## 5. DOCSYNC：文档同步

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| DOC-001 | DocSync CI | `bin/aicoding.exe docsync ci --json` | 退出码 0，JSON 合法 | REQUIRED |
| DOC-002 | DocSync all | `bin/aicoding.exe docsync all --json` | 退出码 0，JSON 合法 | WARN |
| DOC-003 | DocSync release | `bin/aicoding.exe docsync release --json` | 退出码 0，JSON 合法 | REQUIRED |
| DOC-004 | 文档索引一致性 | 静态检查 README/COMMANDS/C99 文档 | README 只索引稳定 hub 文档；leaf skill 文档由 COMMANDS 与 leaf 文档自身覆盖 | REQUIRED |

## 6. LIFECYCLE：外部 skill / kit 生命周期规范

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| LIFE-001 | registry 结构 | 解析 `config/kit-registry.json` | schemaVersion/name/kits/manifest 完整 | REQUIRED |
| LIFE-002 | manifest 存在性 | 检查 registry 中每个 manifest | 文件存在 | REQUIRED |
| LIFE-003 | install plan | `bin/aicoding.exe lifecycle plan --action install --scope kit --all --json` | 退出码 0，JSON 合法 | REQUIRED |
| LIFE-004 | update plan | `bin/aicoding.exe lifecycle plan --action update --scope kit --all --json` | 退出码 0，JSON 合法 | REQUIRED |
| LIFE-005 | uninstall plan | `bin/aicoding.exe lifecycle plan --action uninstall --scope kit --all --json` | 退出码 0，JSON 合法 | REQUIRED |
| LIFE-006 | rollback 只读契约 | `bin/aicoding.exe lifecycle rollback --scope kit --help` | 退出码 0，帮助中包含 `--last`；测试不得应用 rollback snapshot | REQUIRED |
| LIFE-007 | kit lifecycle 结构验证 | `bin/aicoding.exe kit verify --all --profile Lifecycle --json` | 所有启用 kit 的 lifecycle 结构有效 | REQUIRED |
| MCP-001 | MCP registry inventory | `bin/aicoding.exe mcp list --json` | MCP registry 与 Codex 配置 inventory 可读取，`registryDigest` 为稳定 SHA-256 摘要 | REQUIRED |

## 7. EXPORT / FRESH_CLONE

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| EXP-001 | Release export zip | `bin/aicoding.exe export --all --zip --json` | 仅 Release 执行；生成 zip/manifest | REQUIRED |
| EXP-002 | Full export manifest | 进程内 dry-run manifest 校验 | include 均匹配，outputName token 可解析，不生成 ZIP | REQUIRED |
| FRESH-001 | 物化源码 Release 验证 | testengine 私有 materialized leaf | 对验证主体 Tree 执行本地 `git archive` 并用 Go 标准库读取 tar 流，递归物化 pinned gitlink，源码树内无 `.git`/worktree-only 文件；重建 CLI 并执行 `release verify` | REQUIRED |
| FRESH-003 | Full fresh-clone/物化契约 | 静态检查 `.gitmodules`、skills gitlink与三个 profile 分支 | 不 clone，契约完整 | REQUIRED |
| FRESH-004 | 真 clone 传输面变化提示 | 比较 Git common-dir advisory baseline、当前 Tree 与未暂存路径 | `.gitmodules`、`.gitattributes`、`.githooks/**` 或 bootstrap 路径变化时 WARN 并建议显式 `fresh-clone`；不阻断 Release | WARN |

定期 CI 不新增 Registry ID：每周/手动 `clean-clone-full` job 直接运行正式 leaf command
`bin/aicoding.exe fresh-clone --profile Full --json`，在临时真 clone 中执行 `go test ./...`。公共
`fresh-clone --profile Smoke|Full|Release` 始终保持 `sourceMode=cloned`，成功后更新本地 advisory
baseline；它不进入日常 Release Registry，也不创建第二种 Receipt。

## 8. README_DOCS：README 和命令文档治理

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| DOCS-001 | README 三件套 | 检查 `README.md`、`README_CN.md`、`README_EN.md` | 文件存在 | REQUIRED |
| DOCS-002 | README 架构声明 | 静态搜索 Go CLI/lifecycle/doctor/verify/test/release | README 只展示正式产品入口 | REQUIRED |
| DOCS-003 | COMMANDS 命令矩阵 | 检查 `docs/COMMANDS.md` | 包含正式产品命令、领域命令和一个版本兼容表 | REQUIRED |
| DOCS-004 | 命令控制面文档 | 检查 `docs/COMMANDS.md` | 包含唯一 test engine、共享 report 和 PowerShell boundary | REQUIRED |
| DOCS-005 | C99 skill 文档 | 检查 `docs/guides/C99_STANDARD_C_SKILL.md` | 包含配置边界、C Kit 资产边界和统一 CLI 入口 | REQUIRED |
| DOCS-006 | 架构图格式、命令与节点预算 | 解析 README 的 Visio light/dark SVG、其余五个 Mermaid 载体与 `internal/cli/catalog.go` | README 双主题 SVG 节点集合一致；每图节点不超过 20；图内命令来自 typed catalog | REQUIRED |

## 9. CAPABILITY：平台能力目录

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| CAP-001 | internal capability registry | `bin/aicoding.exe governance capabilities --json` | `internal/` 一级包无孤儿、公共入口存在、架构文档与生成索引同步 | REQUIRED |

## 10. GIT_GOVERNANCE：Git 仓库治理

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| GIT-001 | 工作区状态 | `git status --short` | 输出保存；有改动不直接失败 | WARN |
| GIT-002 | hooks verify | `bin/aicoding.exe verify hooks --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-003 | repo text verify | `bin/aicoding.exe verify repo-text --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-004 | release notes verify | `bin/aicoding.exe verify release-notes --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-005 | governance lint | `bin/aicoding.exe governance lint --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-006 | tag audit | `bin/aicoding.exe tag audit --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-007 | `.gitattributes` 策略 | 字段级静态检查 EOL/binary 策略，允许多空格/Tab/注释 | md/go/json/yml/yaml LF；ps1/psm1 CRLF；zip/exe binary | REQUIRED |
| GIT-008 | repository layout | `bin/aicoding.exe governance layout --json` | 仓库 ownership 与 layout 规则通过 | REQUIRED |
| GIT-009 | reuse governance | `bin/aicoding.exe governance reuse --json` | reuse evidence gate 通过 | REQUIRED |
| GIT-010 | config/schema 双向完备性 | `bin/aicoding.exe governance dependencies --json` | 每个 `config/**/*.json` 配置均绑定 strict schema 或有理由精确排除；每个 schema 均被 binding/standalone 反向登记，幽灵配置、schema、排除和模糊通配失败并指出路径 | REQUIRED |

## 11. PWSH_BOUNDARY：PowerShell 边界

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| PWSH-001 | PowerShell inventory | `bin/aicoding.exe doctor pwsh --json` | 输出当前 PS 调用点与 `remainingScripts/thinShells/deprecated` 退役计数；计数只报告、不设门禁 | WARN |
| PWSH-002 | PowerShell budget | `bin/aicoding.exe doctor pwsh-budget --json` | 调用点不超预算；顶层脚本集合等于实测基线且基线历史只追加严格子集，新增/替换/unspecified 指出路径并非零 | REQUIRED |
| PWSH-003 | 默认入口不经 PS 编排 | 检查 Taskfile 是否存在 Go-native 默认路由 | doctor/verify/Smoke/Full/Release 均直达正式 Go CLI；允许变量和 Windows/Unix 路径分隔符 | REQUIRED |
| HEALTH-001 | typed command 延迟门禁 | `bin/aicoding.exe doctor perf --json` | fast/standard 注册命令各进程内实测 3 次取中位数；1.5× Warn、3× Fail | REQUIRED |

## 12. REPO_CONTEXT：仓库上下文领域

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| RC-001 | repo-context 扫描与结构验证 | `bin/aicoding.exe lifecycle verify --scope repo-context --json` | 事实快照可构建；已安装时 manifest 结构完整（未安装为空操作通过） | REQUIRED |
| RC-002 | repo-context 生成计划 | `bin/aicoding.exe lifecycle plan --action install --scope repo-context --json`（Full/Release） | 生成计划可计算且不写盘 | REQUIRED |

## 13. ADR_REVIEW：Primitive 宪法自评门禁

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| ADR-001 | 新 Primitive ADR 含 §12 自评 | 静态检查 `docs/decisions/*.md`：声明 `PrimitiveReview: required` 的 ADR 必含 `## §12 Checklist 自评` 节（`internal/adrreview`） | 无缺口；缺失时报出具体 ADR 文件与修复指引 | REQUIRED |

## 14. RELEASE_GATE：Release policy

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| REL-002 | Release policy 文档 | 静态检查 release/tag policy 文档 | 文档存在且被 README 索引 | REQUIRED |
