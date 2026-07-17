# Architecture Overview

## Architecture Authority

[AiCoding 内核与扩展图架构](architecture/AICODING_CORE_ARCHITECTURE.md) 是总体控制面、
稳定内核、扩展契约、性能预算和迁移纪律的权威文档。本文只保留仓库导航和当前实现
总览；Kit、MCP、PowerShell、DocSync 等专项文档必须服从该总体架构。

## Repository Role

AiCoding 是本地 AI coding 工作流的平台仓库。它拥有 kit registry、kit manifest、本地 hook、Taskfile 路由、Go CLI 控制面、发布治理文档和 CodingKit 平台资产。

AiCoding 不拥有嵌入式 skill 源码。权威 skill/plugin 源位于 `CodingKit/agents/skills` 子模块及其生成包资产。

## Layer Model

```text
platform
  -> integration
     -> capability
        -> runtime
```

平台层包含 User/Agent、Taskfile 和 Go CLI；integration 层包含 registry、lifecycle、插件绑定与安装状态；capability 层包含通用 Kit、Skill、MCP 和 CodingKit 模块；runtime 层包含语言、协议、操作系统与外部应用。依赖只允许同层或向下，详见 [依赖方向与稳定身份治理](governance/DEPENDENCY_DIRECTION_POLICY.md)。

具体执行链仍为：

```text
User / Agent
  -> Taskfile routing
     -> Go CLI
        -> registry / lifecycle / governance
           -> reusable capability
              -> external runtime
```

## Go CLI Control Plane

Go CLI 是唯一正式产品控制面，提供稳定 `report.Result` JSON 与共享
`StandardReport`/check schema。正式产品入口只有：

- `bootstrap`；
- `lifecycle ...`；
- `doctor --all`；
- `verify --profile Smoke|Full|Release`；
- `test --profile Smoke|Full|Release` 和 `test latest`；
- `release verify` 和 `release gate`。

Hook、governance、DocSync、Skill、MCP、export、fresh-clone、C99 和专项 doctor/verify
命令属于领域子命令；旧 `smoke`、`ci`、`full`、位置参数 test、`kit lifecycle`、
MCP lifecycle 动词和 `status --all` 只保留一个版本并输出 `CLI_DEPRECATED`。

## Single Implementation Authorities

```text
internal/cli        -> 参数、帮助、兼容路由、退出码
internal/lifecycle  -> Kit / MCP / runtime Skill 静态 adapter
internal/repohealth -> product doctor / verify 的确定性检查组合
internal/testengine -> 唯一 Smoke / Full / Release Registry 与执行器
internal/report     -> Result / StandardReport / Check / errorKind Schema
internal/runner     -> ExecutionPlan、snapshot/digest、有界并发、超时、取消与稳定输出
internal/registry   -> 规范化 snapshot、稳定 digest 与只读 view
internal/cli        -> typed command catalog、handler routing、help 与退出契约
```

`doctor` 只诊断环境和状态；`verify` 只执行静态/结构验证；`test` 独占测试执行；
`release` 只执行发布结构验证或复用 Release test profile。CI 直接调用
`test --profile`，不再叠加第二个聚合器。

目标架构不增加动态 Go plugin 或第二控制面。现有深模块逐步收敛为 root/path、
manifest snapshot、capability graph、plan/runner、report/contract 和 state/journal
六类稳定 plumbing；领域功能通过声明式 descriptor 与静态 adapter 组合。

## Execution Plan Boundary

`internal/runner.ExecutionPlan` 是计划意图的核心对象。每个 task 用稳定 `action` 和参数
描述，可生成 snapshot 与 digest；选择或删除 task 会返回新 plan，不修改原对象。执行函数
不进入摘要，因此摘要不依赖进程地址。当前只读检查继续有界并发并保持输出顺序；写操作
串行化与 journal 属于后续迁移，不能在领域 adapter 中自行实现第二调度器。

## Registry And Command Catalog Boundary

Kit 与 MCP registry 通过 `internal/registry.Snapshot` 规范化和摘要，原 `LoadRegistry`
兼容入口只返回 snapshot 的只读副本。Registry digest 只表示 registry 文档，manifest
内容由后续 manifest snapshot 单独建模。

`internal/cli` 的 typed command catalog 是顶层 command ID、alias、namespace、handler 和
全局 help form 的权威源。CLI 不再维护平行的字符串 switch 与 help 列表；Taskfile 和命令
文档的机器一致性校验仍是后续工作。

写状态、写 ZIP、安装/卸载等有副作用路径保持在对应 Go 包中串行执行。Doctor/Verify
以及 MCP 子进程使用显式总超时或 context，避免外部工具无限等待。

## Manifest Contract

Kit manifest 使用当前分类：

- `mode`: `go-builtin`, `external-cli`, `powershell-specialty`, `declarative`。
- `type`: `builtin-check`, `builtin-lifecycle`, `builtin-package`, `external-command`, `go-composed`, `specialty-pwsh`, `unsupported`。

## PowerShell/Python Boundary

PowerShell/Python 只保留在专项边界：tag planning / overlay compatibility、PowerShell 质量、安全、Plan Mode、外部 skill 和硬件/工具链专项流程。它们不是默认 Full、Release、lifecycle、export、fresh-clone、DocSync 或 skill verify 控制面。

## 目录治理

`config/repository-layout.json` 是目录治理门禁的唯一机器配置；`aicoding governance layout --json` 据此校验根目录 allowlist、遗留目录、暂存生成物、文档位置、Prompt 归属、`tests`/`testdata` 重叠和 Skill 多 source-of-truth。`config/repository-navigation.json` 仅供 IA 导航生成器生成 hub 和 README 标记区，不引入第二个运行时门禁。

- `cmd`、`config`、`internal`、`CodingKit` 与 `.aicoding/memory` 是 source-of-truth；
- `docs` 是文档域，`testdata` 是测试夹具域，`tools` 是迁移与专项工具域；
- `.agents`、`.codex`、`.github`、`.githooks` 是平台固定集成路径，不移动其根目录；
- `.git` 是 Git 必需的元数据目录，不是业务目录，也不参与 human navigation；
- `bin`、`dist`、`test-results` 是忽略的短生命周期生成目录，不能暂存或提交；
- `tools/skill-template` 是 `skill-template` 的临时归属：待 Codex-Skills 在子模块上游提供对应模板域后，再按跨仓库升级流程迁入；
- C Skill V3 本轮不部署；后续经上游验证后唯一允许的运行时挂载点是 `CodingKit/agents/skills/plugins/AiCoding/skills/aicoding-c99-standard-c`，不能再建立顶层或 standalone 镜像。
- `config/dependency-governance.json` 负责 layer、registry binding、namespace、MCP/Skill 职责、稳定身份版本不可观察和 README badge 权威链接门禁。
