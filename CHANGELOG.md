# Changelog

## [Unreleased]

- **fix(test)**: FOC no-compile 报告不再版本化墙钟耗时和本机 Python 绝对路径，改为记录确定性迭代数/checksum，并统一生成文件末尾换行；removes machine-dependent timing/path drift from versioned FOC validation reports.
- **fix(cstyle)**: 仓库级 C 文件头模板删除 `@version`/`version` 变量，并由模板 validator 阻断源码头重新暴露资产版本；keeps reusable C source headers version-opaque.

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

[Unreleased]: https://github.com/JiaxI2/AiCoding/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/JiaxI2/AiCoding/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/JiaxI2/AiCoding/compare/v0.7.0...v0.8.0
