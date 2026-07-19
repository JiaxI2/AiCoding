# Changelog

## [Unreleased]

- **feat(repo-context)**: 落地 repo-context 提交后增量同步（ADR 0003 阶段 3）——新增 `hook post-commit`（由 `.githooks/post-commit` 触发）读取 HEAD 变更文件、映射受影响顶层域，并把生成写入收敛为**只写内容真正变化的文件**（install/update/sync 共用 `reconcile`），未受影响的域字节不变、mtime 不动；未安装 repo-context 时静默空操作。`sync` 作为 `update` 的内部增量步骤实现，不新增动词；新增 `gitx.CommitFiles` 读取单次提交变更。 / Adds post-commit incremental sync for repo-context (ADR 0003 stage 3): a post-commit hook reconciles only the files whose content actually changed, leaving unaffected domains byte-identical, implemented as update's internal step without a new verb.

- **feat(repo-context)**: 落地 repo-context 新领域 adapter（ADR 0003 阶段 1–2）——`internal/repocontext` 确定性扫描器把仓库事实（名称、语言构成、工具链、顶层域）规范化为稳定 digest 快照（复用 `internal/registry`），并生成受管的小粒度 scoped context 文件到 `.aicoding/repo-context/`；经 `lifecycle --scope repo-context` 复用 install/update/uninstall/status/doctor/verify 六动作，manifest + 内容 digest 保证只删自己生成的文件、对账新鲜度、发现篡改，用户手写内容永不被覆盖；内核六模块零修改（catalog 增一个 descriptor）。 / Lands the repo-context domain adapter (ADR 0003 stages 1-2): a deterministic Go scanner producing a stable facts snapshot and generating managed scoped-context files under lifecycle's eight verbs, with owned-asset digest discipline and zero kernel changes.

- **docs(plan)**: 起草 repo-context 新领域立项 ADR 0003（descriptor 草案 + 三条件论证 + 六步准入应答 + 阶段验收），并把生态对照项目清单与"四象限×可沉淀知识资产"映射沉淀进 00/07 架构篇（含预留出口表新增 Skill 自进化与会话记忆采集两项）。 / Drafts ADR 0003 proposing the repo-context domain adapter and sediments the verified ecosystem reference list plus the quadrant knowledge-asset mapping into the architecture series.

- **docs(architecture)**: 新增 00–07 编号平台架构系列——八层系统总图与层职责、Context/Skill/Workflow/Governance 体系、扩展规范与四象限演进路线；内核命令面以直白功能清单固化为地基，repo-context（参照 `aspenkit/aspens` 与 `Bollwerkio/werkstatt`，均 MIT）列为分阶段扩展主线。 / Adds the numbered platform architecture series 00-07 covering the layered system view, context/skill/workflow/governance architecture, the extension SDK, and the quadrant-based roadmap that freezes the concrete core-command baseline and stages the repo-context capability plan.

- **feat(skills)**: 将经三重门禁验证（`aicoding-skill.ps1 verify`、`quick_validate`、`skill_gate`）与架构审计（八动词编排、命令真实性实测、分域 rollback、获取/激活分离）的用户 Skill `aicoding-upgrade-train`（升级列车）与 `aicoding-environment-rebuild`（环境重建）从 Draft 安装进 RepoLocal（`.agents/skills/`），进入版本管理并可被 agent 发现；adopt 进 Kit 留待真实试用反馈。 / Installs the gate-verified and architecture-audited user skills aicoding-upgrade-train and aicoding-environment-rebuild from draft into the version-controlled repo-local runtime path; Kit adoption is deferred pending real usage.

- **docs(architecture)**: 废弃从未进入运行时且已不可恢复的 `aicoding-external-integration` 草稿，修正架构手册 §7.3 与冻结边界文档 §3.2 的两处悬空引用，改指现存 RepoLocal 工作流 Skill 与既有获取侧四步流程；不改动任何契约条款。 / Deprecates the never-activated, unrecoverable external-integration draft and repairs the two dangling documentation references without touching any frozen contract clause.

- **docs(architecture)**: PowerShell 专项命令面声明停止增长——不新增专项脚本、不新增保留类别，新能力一律进入 Go 控制面（verify-codex-kit 退役先例的一般化）。 / Declares the PowerShell specialty command surface frozen: no new specialty scripts or categories; new capabilities land in the Go control plane.

- **docs(architecture)**: 记录契约冻结与获取/激活边界的独立 Phase 0–5 验收证据，将架构文档状态收敛为 `Accepted and Frozen`，并将实施计划标记为已批准。 / Records the independent Phase 0–5 acceptance evidence for the contract-freeze and acquisition/activation boundary, freezes the accepted architecture contract, and marks the implementation plan approved.

- **feat(governance)**: 新增契约冻结与获取/激活分离边界，通过 `activation manifests URL-free` 和 `cloneable sources registry` 两条依赖治理检查阻断激活 manifest URL 与越界可克隆源。 / Adds the contract-freeze and acquisition/activation boundary with two dependency gates for activation URLs and cloneable-source ownership.

- **fix(testengine)**: 在执行与写报告前拒绝无效 UTF-8 或含 `U+FFFD` 的 Registry `title`，并以 JSON 往返回归锁定“仓库根目录识别”“Go 版本”等中文标题，防止不可读文本进入所有测试报告消费者。 / Rejects unreadable registry titles before execution and locks Chinese title preservation through the JSON report contract.

- **docs(maintenance)**: 完成 `verify-codex-kit.ps1` 退役 Phase 1，将 `AGENTS.md`、`CodingKit/README.md` 与仓库内 Agent Patch Kit 的活跃门禁引用迁移到正式 `test --profile Full --json` 入口；只读 Codex-Skills 子模块中的旧引用保留为上游升级事项。 / Completes retirement Phase 1 by migrating repository-owned gate references to the canonical Full profile while leaving read-only submodule references for an upstream upgrade.

- **fix(specialty)**: `tools/specialty/verify-codex-kit.ps1` 从 v1.0.0 已移除的 `full` 兼容命令改为正式 `test --profile Full --json` 入口，并按 JSON `ok`/`errorKind` 契约判读退出码（ok→0、usage→2、其余→1）；按 PowerShell 边界"单独计划和验证"规则新增 [退役计划](docs/decisions/verify-codex-kit-retirement/RETIREMENT_PLAN.md)，并修正 KIT_LIFECYCLE_TEST_PROFILES 中该脚本仍是 Smoke 默认门禁的过时描述。 / Repairs the broken wrapper onto the canonical Full test entry with JSON-contract exit codes, adds the boundary-mandated retirement plan instead of deleting, and corrects stale profile-policy claims.

## [1.0.0] - 2026-07-18

- **refactor(cli)**: 移除已到期的 `smoke`、`ci`、`full`、位置参数 test、`kit lifecycle`、MCP lifecycle 动词与 `status --all` 兼容路由；正式测试统一使用 `test --profile`，lifecycle 调用必须显式声明 `--scope`。 / Removes the expired compatibility routes and requires canonical test profiles plus explicit lifecycle scopes.

- **docs(architecture)**: 将 Git MOC、12 个索引和 Orthogonal Architecture Design Kit 落为“snapshot 事实、plan 意图、runner 调度、adapter 翻译、report 证据、state 领域所有”的正交深模块架构；固化仓库 lifecycle 与 Agent CLI/JSON 边界、Skill/MCP 生命周期、局部测试半径、C/native 采用条件及闭环后的架构冻结规则，删除 speculative capability graph/global journal 无限迁移表。
- **feat(core)**: 将 runner plan 提升为可验证、不可变选择、可 snapshot/digest 的 `ExecutionPlan`；增加通用 Registry Snapshot + Digest 并迁移 Kit/MCP loader；建立 Typed Command Catalog 统一 CLI handler routing、alias、namespace contract 与全局 help；`aicoding version` 改从构建或 manifest 元数据读取，不再硬编码实现代际标签。
- **feat(catalog)**: 增加通用内容树 `CatalogSnapshot`，将规范化 registry digest 与有序 referenced manifest digest 组合；Kit/MCP catalog 在单次命令中只解析 manifest 一次，并让 list、plan/apply、status、doctor、verify 消费 detached snapshot values。
- **refactor(lifecycle)**: 用静态 Adapter Catalog 替换 Kit/MCP/runtime Skill scope switch，明确 input kind、state owner、entrypoint 与 read/write effect；lifecycle 将 adapter selection 转为 `ExecutionPlan` 串行执行，成为第二个真实消费者，同时保留各领域独立 state/rollback 语义。
- **feat(evidence)**: CLI report 增加可选 `inputDigest`/`planDigest`，lifecycle report 增加 adapter `catalogDigest`、`planDigest` 和每领域 `inputDigest`；MCP inventory 保留 `registryDigest` 并增加包含 referenced manifests 的 `catalogDigest`。
- **feat(governance)**: 在现有 dependency gate 增加 production Go package boundary 检查，机器阻断 snapshot/runner/report 反向依赖领域、Kit/MCP 互相依赖及领域反向依赖 CLI/repohealth/testengine，使正交模块和局部测试边界可执行。
- **feat(git-boundary)**: 固化 Git 事实层复用边界，将生产 Git 进程统一收编到零 internal 依赖的 `internal/gitx` 薄封装，并以进程所有权、importer 白名单和 CLI porcelain 动词禁用三条门禁阻断重复实现 Git 能力。
- **docs(architecture)**: 记录 Git 复用边界 Phase 0–5 独立验收证据并将契约冻结，同时补全 Agent 知识面的进入点、生命周期与新功能知识检查。
- **feat(mcp)**: 将 MIT 许可的 PowerPoint COM MCP 源码收养为仓库私有维护的 `ppt-mcp` canonical component，补齐 provenance、隔离依赖、doctor、Smoke/Full/Release 与受管 lifecycle 登记，不保留上游 VCS 或自动更新关系。
- **fix(validation)**: 将仓库级 Markdown link audit 限定为 AiCoding 所有内容，由各自源仓库验证只读 Skill 与外部 fixture submodule；同时跟踪声明的 examples/platforms 稳定根，消除 fresh worktree 的虚假链接与缺失资产 warning。
- **fix(identity)**: 将 Fast Path cache 从 versioned 实现路径迁移到稳定的 `.aicoding/cache/fast-path` identity；旧 cache 仅为可删除临时数据，不再参与当前 status/clean。
- **docs(architecture)**: checkpoint CLI/MCP control-plane 与 Extension Adapter 草稿，作为本轮 Git 原理学习和有限架构闭环的可追溯输入；草稿状态不代表最终 Accepted 契约。

## [0.10.0] - 2026-07-17

- **docs(validation)**: 刷新产品收敛后的 Smoke/Full/Release、Kit、Skill、MCP、DocSync、Git Hook、Governance、Dependency 与 Markdown link 最终验收记录。
- **fix(test)**: Full/Release 的 rollback 用例改为只读 `lifecycle rollback --help` 契约检查，禁止测试 profile 在存在 snapshot 时意外应用仓库状态。
- **docs(control-plane)**: README 三件套、COMMANDS、Architecture、Maintenance、AGENTS、Taskfile 与测试文档统一指向 lifecycle/doctor/verify/test/release 正式入口；旧 CLI 只保留在一个版本兼容表或历史决策中，并扩展 DocSync 对 Go CLI、test engine、Taskfile、CI 与 report schema 的语义绑定。
- **feat(verify)**: 新增带总超时的正式 `doctor --all` 与 `verify --profile Smoke|Full|Release` 产品边界；doctor 将未安装的 worktree-local MCP 作为可操作 warning，verify 只运行确定性静态/结构检查，不递归调用 test engine 或启动 Release 可见工具；未知 JSON 命令保持 stdout-only、`errorKind=usage` 和退出码 `2`。
- **feat(report)**: 为 `report.Result` 增加兼容的 `errorKind`/validation error 契约，为 `StandardReport`/共享 check 增加 schemaVersion、PASS_WITH_WARNINGS 与统一汇总，并发布 `config/schemas/cli-report.schema.json`。
- **refactor(cli)**: 正式 `lifecycle` 命名空间新增 `kit|mcp|runtime-skill|all` scope 与 status/doctor/verify，旧 `kit lifecycle` 和 MCP lifecycle 动词路由到同一 adapter 并输出 `CLI_DEPRECATED`；兼容期内未指定 scope 的 `--all` 保持 Kit 语义。
- **refactor(lifecycle)**: 新增唯一 `internal/lifecycle` 静态编排层，将 Kit、MCP 与 runtime Skill 的 plan/apply/status/doctor/verify 接入同一报告；runtime Skill install/update 必须显式指定 profile，并能从 Git common repository root 安全解析 worktree 外的 Codex-Skills source。
- **refactor(test)**: 兼容 `smoke`、`ci`、`full` 与 `release gate` 直接映射到唯一 `test --profile` 引擎，删除 CLI aggregate plan 和测试注册表中的 `FULL-001`/`REL-001` 自调用；fresh-clone 改为 leaf probe，CI/Taskfile 直接使用正式 profile，消除 Full→Full、Release→Release 与 CI→Smoke 聚合链。
- **refactor(test)**: 将 global tester 的 Config/Profile/Registry/Timeout/Result/Summary/Report/ExitCode 内聚到唯一 `internal/testengine`，正式 CLI 改为进程内调用并复用同一报告存储；`tools/aicoding-global-tester` 退化为兼容薄壳，不再拥有测试实现。
- **feat(cli)**: 新增可测试的 CLI 执行契约，统一全局/命令帮助、参数错误退出码 `2`、执行失败退出码 `1`、严格 JSON stdout 和文本 warning；正式支持 `test --profile Smoke|Full|Release`，旧 `smoke`、`ci`、`full` 与位置参数测试入口输出 `CLI_DEPRECATED`。
- **docs(plan)**: 选择兼容优先的产品闭环收敛路线，定义唯一 CLI/Test/Report/Lifecycle/Release 权威面、一个版本的 `CLI_DEPRECATED` 兼容边界及分阶段验证/回滚计划；selects the compatibility-first product convergence plan and phased gates.

## [0.9.1] - 2026-07-16

- **fix(test)**: FOC no-compile 报告不再版本化墙钟耗时和本机 Python 绝对路径，改为记录确定性迭代数/checksum，并统一生成文件末尾换行；removes machine-dependent timing/path drift from versioned FOC validation reports.
- **fix(cstyle)**: 仓库级 C 文件头模板删除 `@version`/`version` 变量，并由模板 validator 阻断源码头重新暴露资产版本；keeps reusable C source headers version-opaque.
- **fix(plugin-lifecycle)**：`lifecycle install|update` 现在比较源码包与 installed cache 的 `BUILDINFO.json`，仅通过 Codex 官方 `plugin remove/add` 刷新漂移包，并在 CLI 不可用、插件禁用或刷新后仍漂移时阻止写入虚假的 install state；refreshes stale plugin caches through the supported Codex plugin lifecycle instead of editing cache files.
- **fix(skill-runtime)**：runtime audit 改为直接枚举两个 user Skill root 的 junction，默认只核验 AiCoding active cache，并校验精确 profile、source target 与 package digest；profile 切换可显式备份 unmanaged/mismatched 注册路径后统一到 `.agents\skills`，同时生成 rollback manifest。 / Makes Windows junction discovery, profile matching, migration and rollback evidence deterministic.
- **build(codex-kit)**：推进 `CodingKit/agents/skills` gitlink 到已验证的 deterministic plugin metadata 修复提交。 / Advances the released Skill dependency used by plugin drift comparison.

## [0.9.0] - 2026-07-16

- **feat(governance)**: 新增仓库级依赖方向与稳定身份门禁，统一约束 Kit、Skill、MCP、模块命名、registry binding、下层平台无感、MCP/Skill 职责和资产版本不可观察；adds an executable higher-to-lower dependency contract.
- **refactor(visio-runtime)**: 将 Visio leaf Python/environment 配置改由 MCP component manifest 注入，package/module/service/schema/example/test 保持平台无感；moves platform binding out of the reusable capability.
- **refactor(control)**: FOC/PID 的 CMake target、Simulink model、header guard 和源码注释移除 `aicoding`/版本身份，并删除 `PID_VERSION_*` 代码宏；keeps common controllers reusable and version-opaque.
- **refactor(cstyle)**: C UserStyle Kit 源码头不再承载资产 `@version`，版本仅由 manifest、资产文档、CHANGELOG 与 Tag/Release 权威面管理；removes release identity from generated and example source headers.
- **docs(readme)**: README 版本仅通过三语一致 badge 展示；Go、PowerShell、Python 与 clang-format 绑定准确上游版本页，本地 C UserStyle Kit badge 绑定权威本地说明并与 manifest 校验。 / Makes version badges authority-bound and machine-checked.
- **feat(mcp)**: 新增一等 MCP registry 与 `aicoding mcp` Go 控制面，统一 inventory、status、doctor、Smoke/Full/Release、受管安装更新卸载及当前 Codex MCP 的只读兼容性回归。 / Adds a first-class MCP registry and Go lifecycle/compatibility control plane.
- **feat(visio)**: 集成平台无感的通用 `visio-mcp` capability，并将已发布的 standalone `visio-diagram` Skill 登记到 full runtime profile；MCP 仅提供 tools 和 Diagram IR resource，不注册 workflow prompts。 / Integrates the reusable Visio capability and binds the released standalone workflow through the full runtime profile.
- **fix(visio-layout)**: 默认统一矩形框宽高，增加同层中心、主行/主列与层间距对齐检测和有限 repair，并覆盖 VSDX/PNG/SVG/PDF 导出质量。 / Makes diagram sizing and alignment consistent across editable and exported artifacts.
- **fix(visio-connectors)**: Diagram IR 增加确定性侧边端口、多端口归一化位置和正交/直线路由；真实 Visio 回归验证端点误差、双端 glue、路径穿框和路由样式。 / Adds deterministic side and port-lane geometry with live connector regression.
- **fix(visio-text)**: 连接线标签改用独立坐标并强制离线放置；结构与实际路径检查阻断文字覆盖连接线、框线或其他文字。 / Prevents connector labels from sitting on lines and adds coordinate-based collision gates.
- **fix(visio-text)**: 连接线文字新增相对位置锚点与有界漂移，节点新增上下/左右外部标题绑定；无法在中点附近满足净空时阻断而不是将文字推离所属框线。 / Adds bounded connector-label anchors and shape-bound external captions.
- **fix(visio-typography)**: 节点、外部标题和连接线文字同时设置 profile 请求的 Latin/Asian 字体，并以 80% 文本块安全区（菱形 70%）进行真实 COM 检查。 / Enforces requested Latin/Asian fonts and measured text-block safe-area ratios.
- **feat(visio-style)**: 新增精简可替换 JSON style profile，仅控制字体组、默认字号、80% 文字安全区、共享线宽和圆角；默认恢复宋体 10 pt、0.75 pt 黑线和 0.12 in 小圆角，并支持真实 COM 字体/线宽/圆角回归。 / Adds a restrained JSON style profile that preserves the compact visual baseline and verifies fonts, line weights, and corner radius in live Visio.
- **feat(visio-contract)**: 新增 renderer-effective Diagram IR 字段资源，Skill 回归只把真实影响布局、文字或拓扑且产生 PNG 变化的迭代视为有效改进。 / Exposes renderer-effective fields and rejects metadata-only visual claims.
- **fix(visio-sizing)**: 节点文字块显式水平/垂直居中，`sizeClass` 约束同角色框体尺寸，并以统一 80% 内容安全区和显式架构理由限制放大。 / Standardizes centered text, role-based size families, and one bounded 80% content envelope.
- **fix(visio-sizing)**: 尺寸门禁改为按宽高分别计算文字、同侧端口密度和容器成员包围盒；同轴节点能安全共享的维度必须一致，`sizeReason=multiport` 不再绕过过大检测。 / Makes each box dimension measurable and bounded.
- **fix(visio-arrows)**: 固定箭头样式、尺寸与线宽，检查 connector 首尾外向性、终端净空和箭头包围盒，阻断箭头或线尾穿入节点。 / Prevents arrowheads and tails from entering node boundaries.
- **fix(visio-compactness)**: 增加紧凑布局的页面利用率、同轴框间距、总线长和折点指标，并将工程回归样例收敛到语义主链、前馈带和反馈带。 / Adds compactness gates and a converged engineering layout.
- **fix(visio-spacing)**: 同一主轴、同一尺寸族的连续节点改用页面绝对边界计算框间距，结构规划和真实 Visio 页面均阻断超过 `0.03 in` 的组内间距差。 / Enforces equal absolute frame gaps for comparable same-axis peers.
- **test(visio)**: 新增脱敏双环执行器控制框图，以主链、前馈、反馈和多端口车道模拟复杂工程样例，并纳入真实 Visio Release 回归。 / Adds a de-identified engineering control simulation to the Release profile.
- **test(visio)**: Release 输出补齐 `quality.json` 与 `inspection.json`，真实 COM 回归同时验证箭头几何、文字居中、绝对端口和无孤立 `VISIO.EXE`。 / Persists machine-readable live regression evidence.
- **fix(mcp-lifecycle)**: fresh install 显式安装 component package；uninstall 先原子暂存受管 `.venv`，活跃进程锁定时不会先删 Codex 配置，避免半卸载状态。 / Makes fresh installs runnable and prevents partial MCP uninstalls.
- **feat(governance)**: 将 Issue 创建、分类、状态流转、重开和关闭证据纳入 AiCoding 仓库级 Git governance policy，新增结构化 Issue Forms、label 同步/归一化 workflow 和 Go governance lint；adds managed repository Issue lifecycle governance without adding or modifying a runtime skill.
- **feat(report)**: 新增 `codex usage parse|run` Go CLI 与可复用 `internal/report/tokenusage` 子模块，统一解析 App Server 和 `codex exec --json` Token 事件；adds a reusable Codex Token report path.
- **fix(report)**: 按官方 App Server schema 确定性区分累计 `total` 与上下文 `last`，并支持 `cacheWriteInputTokens`，避免随机选择快照和上下文比例超过 100%；separates cumulative and context usage deterministically.
- **feat(external-skill)**: 支持 `AiCoding -> Codex-Skills -> GitHub Skill` 嵌套 submodule 链，并通过 `standaloneSkillRegistry.sourcePaths` 将 `drawio-skill` 映射到上游真实 Skill 子目录；supports URL-bound external standalone Skills without copied source.
- **build(governance)**: 规定后续所有 GitHub 来源 Skill 必须由 Codex-Skills 声明外部子模块并锁定 gitlink，AiCoding 仅维护运行时名称到 Skill 子路径的映射；standardizes chained URL binding for future GitHub Skills.
- **feat(external-skill)**: runtime profile 支持按注册名称安全删除目标完全匹配的 standalone junction；外部 Skill 更新采用最新稳定 SemVer tag，仓库移除同步清理 URL binding 和 gitlink。 / Adds ownership-checked unlink and stable-tag lifecycle rules.

## [0.8.0] - 2026-07-15

- **feat(cstyle)**: 将 C UserStyle Kit 1.2.0 作为 `CodingKit/tools` 自包含 Go module 纳入平台，保留唯一 `skill c99-standard-c` 用户入口，并新增 `fast`/`full` 结构化验证。 / Integrates C UserStyle Kit 1.2.0 through the existing C99 Skill route with structured fast/full verification.

- **test(governance)**: 将真实 C Kit 快速验证加入 Kit registry、Taskfile、全局 Smoke/Full/Release 测试和源码事实检查，同时保持 skills submodule、插件与缓存不变。 / Adds C Kit verification to repository governance without modifying the skills submodule or plugin runtime.

- **fix(pwsh)**: 修复专项脚本从 `tools/specialty` 定位仓库根的旧路径错误，使 Codex Kit 与 runtime Skill 审计可在当前目录架构中真实执行。 / Fixes repository-root discovery for specialty Codex Kit and runtime Skill audits.

- **docs(reference)**: 随 C Kit 发布完整 PDF、规范化 Markdown、raw 转换件、139 条规则目录、黄金 demo、高级可见样例和用户可编辑 VS Code 风格 snippets；以上参考资产按用户明确授权允许公开分发。 / Publishes the complete reference and customization assets under explicit user authorization.

## [0.7.0] - 2026-07-10

- **feat(governance)**: 新增可复用模块登记与证据门禁；以 Go CLI 接入 Skill Verify、hook、CI、DocSync 和 lifecycle，首轮仅采用可回滚的原生实现。 / Adds a reusable-module evidence gate integrated with the Go control plane.

- **ci**: 修复 Windows GitHub Actions 的相对 CLI 路径，避免 `cmd` 将 `bin/aicoding.exe` 解析为命令加参数。 / Fixes Go CLI invocation from Windows CI.

## [0.6.0] - 2026-07-10

- **refactor(layout)**: 收敛文档分类、Plan Mode 产物路径与工具路径，新增 IA 导航配置和生成的目录导航 hub。

- **feat(test)**: 新增全局测试器，并提供 `test full`、`test release` 与 `test latest` 的结构化验证和报告。

- **docs(readme)**: README 只保留平台/kit/plugin/skill 母级架构入口，具体 leaf skill 命令下沉到命令文档；补充 clang-format 17.0.2 badge 和 README 可见性规则。
- **refactor(cli)**: 默认用户入口统一为 `bin/aicoding.exe smoke|ci|full|release gate` 和 `skill c99-standard-c ...`。
- **feat(runner)**: 新增 `internal/runner` 并发 Plan，支持按任务 ID 快速新增、移除和组合只读验证任务。
- **docs**: README、命令文档、架构文档、PowerShell 边界文档、Tag policy 和 Release policy 只描述当前 main 的可观测标准。
- **chore(pwsh)**: Go 默认控制面之外只保留 PowerShell 专项质量、安全、Plan Mode、外部 skill、tag planning / overlay compatibility 和硬件/工具链边界脚本。

[Unreleased]: https://github.com/JiaxI2/AiCoding/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/JiaxI2/AiCoding/compare/v0.10.0...v1.0.0
[0.10.0]: https://github.com/JiaxI2/AiCoding/compare/v0.9.1...v0.10.0
[0.9.1]: https://github.com/JiaxI2/AiCoding/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/JiaxI2/AiCoding/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/JiaxI2/AiCoding/compare/v0.7.0...v0.8.0
