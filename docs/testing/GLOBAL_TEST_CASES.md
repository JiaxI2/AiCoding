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

## 2. BOOTSTRAP：Go CLI 构建

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| BOOT-001 | bootstrap 构建 | `go run ./cmd/aicoding bootstrap --json` | 退出码 0，输出 JSON，生成 `bin/aicoding.exe` 或 `bin/aicoding` | REQUIRED |
| BOOT-002 | CLI 基础可用 | `bin/aicoding.exe bootstrap --json` | 退出码 0，JSON 合法 | REQUIRED |

## 3. GO：Go 单元、并发、race

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| GO-001 | 全仓 Go 单元测试 | `go test ./...` | 退出码 0 | REQUIRED |
| GO-002 | Go race 检查 | `go test -race ./...` | 退出码 0；环境不支持 race 或历史包不兼容记 WARN | WARN |
| GO-003 | Go vet 基础检查 | `go vet ./...` | 退出码 0 | WARN |
| GO-004 | CLI 并发只读调用 | 并发运行 C99 status/templates/governance lint | 全部退出码 0，无 timeout | REQUIRED |
| GO-005 | JSON envelope 稳定性 | 解析核心 CLI 输出 | JSON 可解析 | REQUIRED |

## 4. C99_SKILL：C 语言 skill 风格一致性

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| C99-001 | C99 skill status | `bin/aicoding.exe skill c99-standard-c status --json` | 输出 JSON，包含 skill 状态 | REQUIRED |
| C99-002 | 注释模板校验 | `bin/aicoding.exe skill c99-standard-c templates --json` | 模板 JSON 合法，退出码 0 | REQUIRED |
| C99-003 | 样例路径格式检查 | `bin/aicoding.exe skill c99-standard-c check --scope paths --path tests/style-samples/foc_sample.c --json` | 样例存在时退出码 0 | REQUIRED |
| C99-004 | staged C/H 检查入口 | `bin/aicoding.exe skill c99-standard-c check --scope staged --json` | 退出码 0 或明确无 staged 文件 | REQUIRED |
| C99-005 | source-of-truth 配置 | 检查 `config/skills/c99-standard-c/*` 与 `.clang-format` | 配置文件存在，投影包含关键字段 | REQUIRED |
| C99-006 | 排除目录策略 | 解析 `skill.json` | 包含 vendor/third_party/generated/Drivers/device/build/out/dist | REQUIRED |

## 5. DOCSYNC：文档同步

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| DOC-001 | DocSync CI | `bin/aicoding.exe docsync ci --json` | 退出码 0，JSON 合法 | REQUIRED |
| DOC-002 | DocSync all | `bin/aicoding.exe docsync all --json` | 退出码 0，JSON 合法 | WARN |
| DOC-003 | DocSync release | `bin/aicoding.exe docsync release --json` | 退出码 0，JSON 合法 | REQUIRED |
| DOC-004 | 文档索引一致性 | 静态检查 README/COMMANDS/FAST_PATH/C99 文档 | README 只索引稳定 hub 文档；leaf skill 文档由 COMMANDS 与 leaf 文档自身覆盖 | REQUIRED |

## 6. LIFECYCLE：外部 skill / kit 生命周期规范

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| LIFE-001 | registry 结构 | 解析 `config/kit-registry.json` | schemaVersion/name/kits/manifest 完整 | REQUIRED |
| LIFE-002 | manifest 存在性 | 检查 registry 中每个 manifest | 文件存在 | REQUIRED |
| LIFE-003 | install plan | `bin/aicoding.exe lifecycle plan --action install --all --json` | 退出码 0，JSON 合法 | REQUIRED |
| LIFE-004 | update plan | `bin/aicoding.exe lifecycle plan --action update --all --json` | 退出码 0，JSON 合法 | REQUIRED |
| LIFE-005 | uninstall plan | `bin/aicoding.exe lifecycle plan --action uninstall --all --json` | 退出码 0，JSON 合法 | REQUIRED |
| LIFE-006 | rollback 入口 | `bin/aicoding.exe lifecycle rollback --last --json` | 无历史时应有明确 JSON 结果；可为 WARN | WARN |

## 7. EXPORT / FRESH_CLONE

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| EXP-001 | export zip | `bin/aicoding.exe export --all --zip --json` | 退出码 0，JSON 合法，生成 zip/manifest | REQUIRED |
| FRESH-001 | fresh-clone Smoke | `bin/aicoding.exe fresh-clone --profile Smoke --json` | 退出码 0；网络失败保留日志 | WARN |
| FRESH-002 | fresh-clone Release | `bin/aicoding.exe fresh-clone --profile Release --json` | release profile 时执行 | WARN |

## 8. README_DOCS：README 和命令文档治理

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| DOCS-001 | README 三件套 | 检查 `README.md`、`README_CN.md`、`README_EN.md` | 文件存在 | REQUIRED |
| DOCS-002 | README 架构声明 | 静态搜索 Go CLI/Fast Path/DocSync/skill verify/lifecycle/export/fresh-clone | 关键入口存在 | REQUIRED |
| DOCS-003 | COMMANDS 命令矩阵 | 检查 `docs/COMMANDS.md` | 包含 bootstrap/smoke/ci/full/release/C99/DocSync/lifecycle/export/fresh-clone | REQUIRED |
| DOCS-004 | Fast Path 文档 | 检查 `docs/FAST_PATH_COMMANDS.md` | 包含 Go 默认控制面和 PowerShell boundary | REQUIRED |
| DOCS-005 | C99 skill 文档 | 检查 `docs/C99_STANDARD_C_SKILL.md` | 包含配置边界和 CLI 入口 | REQUIRED |

## 9. GIT_GOVERNANCE：Git 仓库治理

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| GIT-001 | 工作区状态 | `git status --short` | 输出保存；有改动不直接失败 | WARN |
| GIT-002 | hooks verify | `bin/aicoding.exe verify hooks --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-003 | repo text verify | `bin/aicoding.exe verify repo-text --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-004 | release notes verify | `bin/aicoding.exe verify release-notes --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-005 | governance lint | `bin/aicoding.exe governance lint --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-006 | tag audit | `bin/aicoding.exe tag audit --json` | 退出码 0，JSON 合法 | REQUIRED |
| GIT-007 | `.gitattributes` 策略 | 字段级静态检查 EOL/binary 策略，允许多空格/Tab/注释 | md/go/json/yml/yaml LF；ps1/psm1 CRLF；zip/exe binary | REQUIRED |

## 10. PWSH_BOUNDARY：PowerShell 边界

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| PWSH-001 | PowerShell inventory | `bin/aicoding.exe doctor pwsh --json` | 输出当前 PS 文件清单 | WARN |
| PWSH-002 | PowerShell budget | `bin/aicoding.exe doctor pwsh-budget --json` | 不超预算，或输出超预算明细 | REQUIRED |
| PWSH-003 | 默认入口不经 PS 编排 | 检查 Taskfile 是否存在 Go-native 默认路由 | Smoke/Full/Release 至少存在 Go CLI 路由；允许变量、Windows/Unix 路径分隔符和拆分 smoke+ci | REQUIRED |

## 11. RELEASE_GATE：Full/Release Gate

| ID | 用例 | 方法 | 期望结果 | 严重级别 |
|---|---|---|---|---|
| FULL-001 | Full 聚合 | `bin/aicoding.exe full --json` | full profile 执行，退出码 0 | REQUIRED |
| REL-001 | Release gate | `bin/aicoding.exe release gate --json` | release profile 执行，退出码 0 | REQUIRED |
| REL-002 | Release policy 文档 | 静态检查 release/tag policy 文档 | 文档存在且被 README 索引 | REQUIRED |
