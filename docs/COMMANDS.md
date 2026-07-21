# Commands

Taskfile 是人机短路由；Go CLI 是默认控制面；业务逻辑位于 Go 的 `internal/*` 包中。PowerShell/Python 只作为专项工具保留，不承载默认 Smoke、CI、Full 或 Release profile。

C/H 风格命令见 [C99 Standard C Skill](guides/C99_STANDARD_C_SKILL.md)。

## 默认入口

| 场景 | 命令 | 控制面 |
|---|---|---|
| Bootstrap | `go run ./cmd/aicoding bootstrap --json` | Go |
| 生命周期 | `bin\aicoding.exe lifecycle ... --json` | Go |
| 产品诊断 | `bin\aicoding.exe doctor --all --json` | Go |
| 产品验证 | `bin\aicoding.exe verify --profile Smoke\|Full\|Release --json` | Go |
| 产品测试 | `bin\aicoding.exe test --profile Smoke\|Full\|Release --json` | Go |
| 发布闭环 | `bin\aicoding.exe release verify\|gate --json` | Go |
| 最近测试报告 | `bin\aicoding.exe test latest` | Go |

## 正式产品命令

| 目的 | 命令 |
|---|---|
| Bootstrap | `bin\aicoding.exe bootstrap --json` |
| Kit 合规脚手架 | `bin\aicoding.exe kit init <id> [--external] [--dry-run] --json` |
| Kit 能力深投影 | `bin\aicoding.exe kit describe --kit <id>\|--all [--with-state] --json` |
| Kit 生命周期 plan/apply | `bin\aicoding.exe lifecycle plan --action install\|update\|uninstall --scope kit --all --json` / `lifecycle install\|update\|uninstall --scope kit --all --json` |
| 全域生命周期 plan | `bin\aicoding.exe lifecycle plan --action install\|update --scope all --runtime-profile runtime\|full\|skill-development --json` |
| 全域生命周期状态/诊断 | `bin\aicoding.exe lifecycle status\|doctor --scope all --json` |
| 产品诊断 | `bin\aicoding.exe doctor --all --timeout-sec 180 --json` |
| 产品验证 | `bin\aicoding.exe verify --profile Smoke\|Full\|Release --timeout-sec 180 --json` |
| 官方测试 | `bin\aicoding.exe test --profile Smoke\|Full\|Release --json` |
| 最近测试报告 | `bin\aicoding.exe test latest` |
| Release 结构验证 | `bin\aicoding.exe release verify --json` |
| Release 测试门禁 | `bin\aicoding.exe release gate --json` |

`doctor` 只做环境和状态诊断；`verify` 只组合确定性静态/结构验证；`test` 独占
Smoke/Full/Release 测试 Registry、timeout、runner、report 和 exit code；`release`
不创建 Tag 或 Release，只执行结构验证或复用 Release test profile。
测试 profile 对 rollback 只执行 `lifecycle rollback --scope kit --help` 的只读契约检查，不会应用
本地 rollback snapshot。

## 命令契约固化

顶层 command ID、名称/alias、是否要求 subcommand、handler 和 `aicoding --help` form
由 `internal/cli` 的 typed command catalog 统一描述。新增或删除顶层命令必须先更新该
catalog，并通过 catalog 完整性与 CLI contract 测试；不能再在 router、help 和
namespace 判断中分别维护字符串列表。

Full 保留 Go/race/vet、CLI/JSON、治理和领域结构验证，但不创建发布 ZIP、也不执行 hermetic
fresh clone；对应覆盖由 EXP-002 export manifest 静态验证和 FRESH-003 clone 契约静态验证承担。
Full/Release 还会执行固定版本的 Staticcheck `v0.7.0`（GO-005，首个 release 为 WARN）和
govulncheck `v1.6.0`（GO-006，真实漏洞为 REQUIRED；仅可识别的网络访问失败降级为 WARN）。
Release 仍执行真实 `export --zip` 与一次 `fresh-clone --profile Release`，因此 Release 比 Full
更慢是刻意的发布边界，而不是性能回退。

到期的兼容命令已从 catalog、router 和 help 删除；旧写法返回 usage error（退出码 2），
不会再静默转发。`lifecycle` 现在要求显式 `--scope kit|mcp|runtime-skill|repo-context|all`。
`--scope repo-context` 作用于整个仓库，不接受 `--kit`、`--component` 或 `--all`；它扫描仓库
事实（目录、语言/工具链、依赖边）生成受管的小粒度上下文文件到 `.aicoding/repo-context/`，
并用 facts digest 与生成物 digest 对账新鲜度。`hook post-commit`（由 `.githooks/post-commit`
触发）在每次提交后读取本次变更文件，增量重生成受影响域的上下文，并为 exact-tree Receipt
补 profile/commit alias——只写内容真正变化的文件，
未受影响的域字节不变；未安装 repo-context 时该 hook 静默空操作。它是 `update` 的内部增量步骤
（`sync` 不是新动词）。完整性由聚合 `doctor --all`（`doctor.repo-context`，零扫描）、
`verify --profile`（`verify.repo-context`，零扫描）与测试 `RC-001/RC-002` 对账：owned 文件缺失或
被篡改报 error 阻断门禁；新鲜度漂移是 `lifecycle status --scope repo-context` 的单一职责、
由 post-commit hook 自愈，不进聚合门禁的扫描路径（未安装时全部空操作）。详见
[ADR 0003](decisions/0003-repo-context-domain.md) 与 [07 演进路线](architecture/07-roadmap.md) §3。
`kit list --json` 与 `mcp list --json` 的外层报告包含
`inputDigest`；MCP inventory 同时保留 `registryDigest` 并增加 `catalogDigest`。前者只标识
规范化 registry，后者标识 registry 与全部 referenced manifests 的内容树。
`kit describe --kit <id>|--all --json` 在 `report.Result.data` 返回稳定 `[]PluginView`，其中
`quickstart` 从 manifest description、首个 read command 与 Skill 描述派生；非 JSON 输出由
`report.WriteText` 即时渲染，不生成 QUICKSTART 文件。默认不读安装 state，只有
`--with-state` 才附加不含时间戳的摘要。详见 [Kit Plugin View](reference/KIT_PLUGIN_VIEW.md)
与 [Kit 管理标准](reference/KIT_MANAGEMENT_STANDARD.md)。

`kit init <id> --json` 从 `config/templates/kit/` 生成 schema v2 manifest、disabled registry
entry、`testdata/kits/<id>/workspec-example.json`，并在 `dependency-governance.json` 追加
`capability`、`platformAgnostic:true`、空 roots/dependsOn 的保守 binding；`--external` 额外生成
`docs/reference/kits/<id>-BOUNDARY.md`，并固定 `thirdParty:true`、`updatePolicy:pinned`。
`--dry-run` 只返回目标路径、字节数与内容摘要。命令不自动 enable、不写 Skill，也不猜测真实
上游依赖；目标或 ID 已存在时 fail-closed。正式启用前须评审并按实际实现修正 binding。
正式 `lifecycle ... --json` 在 `data` 中返回静态 adapter `catalogDigest`、本次
`planDigest`，并在每个 adapter result 中返回 `inputDigest`。Agent/Skill 应使用这些字段
追踪“对什么事实执行了什么意图”，不解析人类文本或直接调用 specialty 脚本。
`aicoding version` 从构建注入值或 `config/codex-kit.json` manifest 元数据读取版本，
不再把实现代际标签硬编码到 Go 文件。
Fast Path 的稳定 cache identity 为 `.aicoding/cache/fast-path`；旧的 versioned cache
是可删除的临时数据，不再由当前 `cache status|clean` 管理。

## 本地生成物观测与保留

`cache` 是仓库内本地生成物的唯一观测/清理面，只扫描以下四个注册根，不遍历无关目录：

| scope | 路径 | `clean` 策略 |
|---|---|---|
| `fast-path` | `.aicoding/cache/fast-path` | 全部删除，可重新生成。 |
| `test-results` | `test-results/aicoding-global-test-*` | 保留最近 N 份（默认 5）并额外保留全部 FAIL 或无法判定结论的证据。 |
| `validation-reports` | `<git-common-dir>/aicoding/validation/reports/*` | 仅删除没有 Receipt 或 commit alias 引用的报告；完整性读取复用 Validation Evidence store。 |
| `work-state` | `.aicoding/state/work/*` | 只报告大小；审计轨迹（尤其 `attempts.jsonl`）不允许由 cache 清理。 |

```powershell
bin\aicoding.exe cache status --json
bin\aicoding.exe cache clean [--scope fast-path|test-results|validation-reports|work-state] [--keep 5] [--dry-run] --json
```

不指定 `--scope` 时应用全部可清 scope 的默认策略，但跳过 `work-state`；`--dry-run`
只返回计划删除清单和预计释放字节，零落盘。成功的 `test --profile ...` 在报告写入完成后自动对
`test-results` 应用 keep-last-5；失败或取消的运行不触发清理。`doctor --all` 的
`doctor.cache-bloat` 在测试结果超过 20 份或 50MB 时给出 warning 和显式 clean 命令，绝不代替用户清理。

## Validation Evidence

Validation Evidence 把一次完整 PASS 测试绑定到 Git Tree 和验证语义。当前
`test --profile ...` 默认是 `--reuse off`；`--reuse auto` 仅显式启用。默认值只有在 main 的
远端 `release-gate` 连续 3 次完成 seed/audit 且切换提交引用三次 run URL 后才能晋级。
`--force` 忽略命中并完整执行，
`--verify-reuse` 同样完整执行并审计实际结论及逐用例状态，二者不能同时使用；
`--allow-dirty` 只允许脏主体继续执行，所得报告永远不可复用。

```powershell
bin\aicoding.exe validation status --json
bin\aicoding.exe validation check --profile Smoke --target HEAD --json
bin\aicoding.exe validation check --profile Release --target HEAD --bind-alias --json
bin\aicoding.exe validation check --profile Full --target INDEX --json
bin\aicoding.exe validation explain --profile Release --target HEAD --json
bin\aicoding.exe validation list [--profile Release] --json
bin\aicoding.exe validation clean [--profile Release] --json
```

`validation check` 只按完整 validation identity 查找精确 Receipt 路径，不扫描或逐文件哈希
仓库。`HEAD` 证明当前提交 Tree；`INDEX` 通过 `git write-tree` 证明暂存区 Tree，因此不会修改
工作区、暂存区或 HEAD，但可能向 Git object database 写入 tree object。仅改 commit message
不会改变 Tree，Receipt 仍可命中；tracked 内容、暂存内容、非 ignored 未跟踪文件或子模块
工作区变脏都会 fail-closed。ignored files 明确不在 Receipt 的证明范围内。

`validation explain` 先走与 `validation check` 相同的精确检查；命中时返回 identity 与完整
fingerprint，miss 时才扫描完整性校验后的同 profile Receipt，并按 Receipt 文件 mtime 选择
最新一份作为“仅诊断参考”。输出的 `changed` 逐字段列出 old/new，`unchanged` 列出其余字段；
命令只读，不创建或绑定 Receipt。普通 `validation check` 仍只读取内容寻址的精确路径，不因
explain 增加目录扫描。

普通 `validation check` 不写 alias。`--bind-alias` 只接受 `--target HEAD`，并且只在 Receipt
完整命中后把当前 commit tip 绑定到该 Receipt；报告返回 `commitAliasBound: true`。它用于
同 Tree 的 metadata-only 历史重写：仅重排/改 message 的 interactive rebase、未触发 hook 的
message-only amend、同 Tree cherry-pick 或 rebase 到同一 base。rebase 到已更新 main 通常会
改变 Tree，此时 check 必须 miss，应重新运行 Release；该选项不会把旧 Receipt 强行绑定到新
内容。pre-push 每个 ref 只检查一个 tip `local_oid`，即使重写了多个 commit，也只需对检出的
最终 tip 执行一次 `--bind-alias`。

Receipt、完整性绑定的报告和 toolchain cache 保存在 Git common-dir 下的
`aicoding/validation/`，不提交到仓库。`validation list` 可按 profile 列出完整 Receipt；
`validation clean` 删除所选 profile（省略时为全部 profile）的本地 alias、Receipt 及其报告。

`.githooks/pre-push` 调用 `hook pre-push`，从 Git stdin 读取实际 push update。默认
`config/validation-policy.json` 规定：`refs/heads/main` 与 `refs/tags/*` 必须已有同一
`local_oid` tree 的 Release Receipt；main 只允许 fast-forward，main/tag 均禁止删除；未匹配
feature ref 明确旁路。hook 本身不运行测试或构建，缺证据时应在 exact commit 上先执行
`validation check --profile Release --target HEAD --bind-alias`：若同 Tree Receipt 命中即可重试；
若命令返回 miss，执行 `test --profile Release --reuse off` 生成新证据后再重试 push。推送非
当前分支时应先检出该 ref 的 tip，不能用另一个 worktree HEAD 替代实际 `local_oid`。

## 领域与专项命令

| 目的 | 命令 |
|---|---|
| DocSync staged/all/ci/release | `bin\aicoding.exe docsync staged|all|ci|release --json` |
| Plan Mode 触发检查 | `bin\aicoding.exe plan check (--staged | --paths PATH ...) --json` |
| Plan Mode 产物校验 | `bin\aicoding.exe plan verify --json` |
| Plan Mode 产物状态 | `bin\aicoding.exe plan status [--id <id>\|--all] --json` |
| Plan Mode 内容批准 | `bin\aicoding.exe plan approve --id <id> --json` |
| Skill verify | `bin\aicoding.exe skill verify --all --profile Smoke|Full|Release --json` |
| MCP inventory | `bin\aicoding.exe mcp list --json` |

`plan check` 先按 `config/plan-policy.json` 判断路径是否敏感，再要求每条敏感路径被至少
一个 `approved` plan 的 scope 覆盖；它不运行验证或创建计划。`exemptPaths` 优先于
`sensitivePaths`，pre-commit 对未覆盖路径 fail-closed。

每个 Plan Mode 会话位于 `docs/spec/<id>/`；`PLAN.md` 是唯一必需文件，
`OPTIONS.md`、`DECISION.md`、`TASKS.md` 按需存在。`plan verify` 只校验 frontmatter、
目录 ID、scope pattern 与文件依赖。`plan approve` 只接受 clean worktree，把当前
`HEAD^{tree}` 写入 `approvedTree` 并把状态改为 `approved`；它是 plan 域唯一写命令。
`plan status` 比较批准树与当前 `HEAD`，列出 scope 内漂移、越界路径、exempt 路径和
当前树的 validationevidence gate 引用；它只建议 `implemented`，不会自动改状态。
| MCP status/doctor | `bin\aicoding.exe mcp status|doctor <COMPONENT> --json` |
| MCP verify | `bin\aicoding.exe mcp verify <COMPONENT>\|--all --profile Smoke\|Full\|Release --configured --json` |
| C99 skill status | `bin\aicoding.exe skill c99-standard-c status --json` |
| C99 skill templates | `bin\aicoding.exe skill c99-standard-c templates --json` |
| C99 skill 快速/完整验证 | `bin\aicoding.exe skill c99-standard-c verify --profile fast\|full --timings --json` |
| C99 skill fmt/check | `bin\aicoding.exe skill c99-standard-c fmt|check --scope changed|staged|paths --json` |
| Rollback | `bin\aicoding.exe lifecycle rollback --scope kit --last --json` |
| Repo-context 生成/保鲜 | `bin\aicoding.exe lifecycle install\|update\|uninstall\|status\|doctor\|verify --scope repo-context --json` |
| Repo-context 提交后增量同步 | `bin\aicoding.exe hook post-commit --json`（由 `.githooks/post-commit` 自动触发） |
| Validation pre-push Context Gate | `bin\aicoding.exe hook pre-push --json`（由 `.githooks/pre-push` 传入 Git stdin） |
| Export | `bin\aicoding.exe export --all --zip --json` |
| Fresh clone | `bin\aicoding.exe fresh-clone --profile Smoke|Full|Release --json` |
| Governance lint | `bin\aicoding.exe governance lint --json` |
| Dependency direction / stable identity | `bin\aicoding.exe governance dependencies --json` |
| Repository layout gate | `bin\aicoding.exe governance layout --json` |
| Reuse governance evidence | `bin\aicoding.exe governance reuse --json` |
| Validation evidence 状态 | `bin\aicoding.exe validation status --json` |
| Validation Receipt 精确检查 | `bin\aicoding.exe validation check --profile Smoke\|Full\|Release --target HEAD\|INDEX --json` |
| Validation Receipt miss 解释 | `bin\aicoding.exe validation explain --profile Smoke\|Full\|Release --target HEAD\|INDEX --json` |
| Validation Receipt 列出/清理 | `bin\aicoding.exe validation list\|clean [--profile Smoke\|Full\|Release] --json` |
| Hook verification | `bin\aicoding.exe verify hooks --json` |
| Repo text verification | `bin\aicoding.exe verify repo-text --json` |
| Release notes verification | `bin\aicoding.exe verify release-notes --json` |
| Git hook 接线检测 | `bin\aicoding.exe doctor --all --json`（含 `doctor.hooks-wired`：用 `git config core.hooksPath` 检测 `.githooks` 是否已激活，未接线只 warning） |
| Provision 状态检测 | `bin\aicoding.exe doctor --all --json`（含 `doctor.provisioned`：读 `.git/config` 的 `aicoding.*` 标记零扫描判断是否已初始化；`aicoding.docsSkeleton=1` 表示文档骨架已放置，未初始化只 warning） |
| PowerShell inventory | `bin\aicoding.exe doctor pwsh --json` |
| PowerShell budget | `bin\aicoding.exe doctor pwsh-budget --json` |
| Tag audit | `bin\aicoding.exe tag audit --json` |
| Todolist（待实现工作清单） | `bin\aicoding.exe todolist --json` |
| Loop WorkSpec 校验 | `bin\aicoding.exe work validate --file <SPEC.json> --json` |
| Loop 下一步裁决/状态 | `bin\aicoding.exe work next\|status --file <SPEC.json> --json` |
| Loop 尝试记录 | `bin\aicoding.exe work record --file <SPEC.json> --attempt <ATTEMPT.json> --json` |
| 本地环境初始化 | `bin\aicoding.exe provision [--repo-root PATH] --json`（git init + 接线 `.githooks` + 写 `aicoding.*` + 建 `.aicoding` 根 + 放置 docs/ SDD 骨架；既有路径 kept、不覆盖，幂等） |
| 解析 Codex Token JSONL | `bin\aicoding.exe codex usage parse --file <FILE> --json` |
| 运行 Codex 并采集 Token | `bin\aicoding.exe codex usage run -- codex exec --json "<PROMPT>"` |

## 已移除的兼容入口

以下入口不再路由；调用会返回 usage error。迁移时使用右侧正式入口：

| 兼容入口 | 正式入口 |
|---|---|
| `smoke` | `test --profile Smoke` |
| `ci --profile <PROFILE>` | `test --profile <PROFILE>` |
| `full` | `test --profile Full` |
| `test full\|release` | `test --profile Full\|Release` |
| `kit lifecycle ...` | `lifecycle ... --scope kit` |
| `mcp install\|update\|uninstall ...` | `lifecycle install\|update\|uninstall --scope mcp ...` |
| `status --all` | `doctor --all` |

## Codex Skill 运行时同步

插件更新先比较 released package 与 installed cache 的 `BUILDINFO.json`；发生漂移时通过 Codex 官方 plugin CLI 重装同一 Marketplace plugin，刷新成功后才写 install state：

```powershell
bin\aicoding.exe lifecycle plan --action update --scope kit --kit aicoding-platform --json
bin\aicoding.exe lifecycle update --scope kit --kit aicoding-platform --json
```

Standalone full profile 通过正式 lifecycle adapter 统一到官方 user root，并在显式迁移时
备份同名 unmanaged path：

```powershell
bin\aicoding.exe lifecycle plan --action update --scope runtime-skill --runtime-profile full --source-repository F:\Study\AI\Codex-Skills --json
bin\aicoding.exe lifecycle update --scope runtime-skill --runtime-profile full --source-repository F:\Study\AI\Codex-Skills --migrate-unmanaged --json
bin\aicoding.exe verify --profile Smoke --runtime-profile full --source-repository F:\Study\AI\Codex-Skills --json
```

底层 PowerShell profile/audit 脚本保留为 lifecycle adapter 的显式 specialty 实现，不再作为
常规文档主入口。

## MCP 组件控制面

MCP registry、component manifest、Codex 配置、生命周期和兼容性回归统一由 Go CLI 管理：

```powershell
bin\aicoding.exe mcp list --json
bin\aicoding.exe mcp status visio-mcp --json
bin\aicoding.exe mcp doctor visio-mcp --json
bin\aicoding.exe mcp verify visio-mcp --profile Smoke --json
bin\aicoding.exe mcp verify --all --profile Smoke --json
```

Agent/Skill 执行写操作时使用正式 lifecycle plan/apply：

```powershell
bin\aicoding.exe lifecycle plan --scope mcp --action update --component visio-mcp --json
bin\aicoding.exe lifecycle update --scope mcp --component visio-mcp --json
```

`--configured` 显式包含 Codex 当前配置的 stdio/Streamable HTTP MCP 只读 initialize/discovery。
MCP 生命周期正式入口使用 `lifecycle --scope mcp`；旧 `mcp install|update|uninstall`
已移除。

Smoke 与 Full 使用 mock/benchmark 路径，不要求 Microsoft Visio；Release 会显式运行可见 Visio COM smoke 和 VSDX/PNG/SVG/PDF 导出。MCP capability 不注册工作流 prompt，画图步骤、有限 repair 和最终视觉确认由上层 `visio-diagram` Skill 负责。

详细边界见 [MCP Control Plane](architecture/MCP_CONTROL_PLANE.md)，操作说明见 [MCP Components](operations/MCP_COMPONENTS.md)。

## Codex Token 报告

`codex usage parse` 支持 Codex App Server 的 `thread/tokenUsage/updated` 通知和
`codex exec --json` 的 `turn.completed.usage` 事件；`--file -` 表示从 stdin 读取。
App Server 报告使用 `tokenUsage.total` 作为会话累计值，并使用
`tokenUsage.last.totalTokens` 计算当前上下文占用，避免把累计会话用量误当作上下文大小。

结构化结果继续使用统一 `report.Result` 外壳，标准报告的
`data.details.token_usage` 保存归一化 Token 数据。`codex usage run` 将子进程 JSONL
事件流保留到 stderr，并在 stdout 输出最终 AiCoding 报告。

## Taskfile 路由

| Task | 命令 |
|---|---|
| `task setup` | `go run ./cmd/aicoding bootstrap --json` |
| `task doctor` | `bin/aicoding.exe doctor --all --json` |
| `task verify` | `bin/aicoding.exe verify --profile Smoke --json` |
| `task smoke` | `bin/aicoding.exe test --profile Smoke --json` |
| `task ci` | `bin/aicoding.exe test --profile Smoke --json` |
| `task full` | `bin/aicoding.exe test --profile Full --json` |
| `task release` | `bin/aicoding.exe test --profile Release --json` |
| `task test:latest` | `bin/aicoding.exe test latest` |
| `task style:c:status` | `bin/aicoding.exe skill c99-standard-c status --json` |
| `task style:c:templates` | `bin/aicoding.exe skill c99-standard-c templates --json` |
| `task style:c:verify` | `bin/aicoding.exe skill c99-standard-c verify --profile fast --timings --json` |
| `task fmt:c` | `bin/aicoding.exe skill c99-standard-c fmt --scope changed --json` |
| `task fmt-check:c` | `bin/aicoding.exe skill c99-standard-c check --scope changed --json` |
| `task fmt-check-staged:c` | `bin/aicoding.exe skill c99-standard-c check --scope staged --json` |

所有依赖 `ensure-bin` 的公开 Task 使用 Task checksum 比较 `cmd/**/*.go`、`internal/**/*.go`、
`go.mod` 与 `go.sum`；仅输入变化或 `bin/aicoding.exe` 缺失时直接执行一次 `go build`。
Task 的 `.task/` checksum 元数据属于已登记且被 Git 忽略的本地 runtime-state。
测试 Registry 的 BOOT-002 只运行 `bootstrap --no-build` 验证 CLI/JSON 契约，BOOT-003
在进程内验证 repo、Go、Git、go.mod 与 bin 前置条件，不重复构建 CLI。默认 `task setup`
仍执行带 build 的 `bootstrap`；`internal/bootstrap` 使用隔离的临时 Go module 单测真实覆盖
`Options{Build:true}` 和二进制落盘，不把真实构建重新加入任何测试 profile。

外部候选可追加 `--target path/to/verify-target.json`；项目差异配置可用可重复的
`--overlay path/to/project-overlay.json`。完整 target/schema 说明见
[C99 Standard C Skill](guides/C99_STANDARD_C_SKILL.md)。

## CI

当前默认 CI workflow 是 `.github/workflows/aicoding-ci.yml`：

```text
go build -o bin/aicoding.exe ./cmd/aicoding
.\bin\aicoding.exe test --profile Smoke --json
```

手动或定时 release job 运行：

```text
bin\aicoding.exe test --profile Release --json
```

同一个每周 schedule（也可手动触发）另设 clean-clone Full job：

```text
bin\aicoding.exe fresh-clone --profile Full --json
```

该 job 在临时递归 clone 中重新构建 CLI 并执行 `go test ./...`，专门发现子模块、gitignore、
干净检出或 Go 构建漂移；它不回填到日常 `test --profile Full` 的成本边界。

CI 不额外调用 `doctor` 或 `verify` 聚合器；唯一 test Registry 已直接登记相应 leaf gate，
避免同一检查在一个 job 中重复执行。

## 运行模型

默认控制面统一由 Go CLI 承担。Smoke、Full 和 Release 只通过 `test --profile` 进入唯一
test engine，全局测试报告落在临时的 `test-results/`；Doctor、Verify、DocSync、
Lifecycle、Skill verify 和 fresh-clone 均由上表中的 Go CLI 路由执行。PowerShell 仅保留
专项慢路径，边界见 [PowerShell Boundary](architecture/POWERSHELL_BOUNDARY.md)。

## PowerShell 专项入口

当前源码仅保留专项脚本类别：tag planning / overlay compatibility、PowerShell 质量、安全、Plan Mode、外部 skill 和硬件/工具链专项流程。默认验证入口不调用这些脚本。

## JSON 和退出码

所有 Go CLI 默认入口输出 `report.Result` envelope。退出码：`0` 表示 `ok=true`，`1`
表示验证或执行失败，`2` 表示参数错误；`errorKind` 使用 `usage`、`execution` 或
`validation`。共享 Schema 见
[CLI、验证与测试报告 Schema](operations/testing/REPORT_SCHEMA.md)。
JSON 报告契约的冻结面、兼容扩展与解冻规则见
[契约冻结与获取/激活边界](architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md)。
