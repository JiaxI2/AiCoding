# 01 总体分层架构（System Architecture）

Status: Derived View（派生视图）

> 本文不定义新契约，全部内容可由 [核心架构](AICODING_CORE_ARCHITECTURE.md)、
> [架构手册](ARCHITECTURE_HANDBOOK.md)、[命令矩阵](../COMMANDS.md) 等权威文档重建；
> 冲突时以契约文档为准。

## 本篇回答的问题

- 为什么整个系统分成这些层？每一层的职责是什么？
- 各层之间允许哪些依赖？
- CLI、Skill、Workflow、Governance 的边界是什么？
- 哪些能力属于核心（Core），哪些属于插件（Plugin）？
- 基础功能的内核命令有哪些，各自具体做什么？（地基命令面）

## 1. 八层总图

```text
                User
                  │
         ┌──────────────┐
         │    IDE/UI    │        Codex CLI / Claude Code / IDE 插件（外部）
         └──────────────┘
                  │
        Agent Runtime Layer       Codex / Claude / GPT 等模型运行时（外部）
                  │
      Context Management Layer    AGENTS.md · Memory · Repository Index · 用户配置
                  │
           Skill Layer            C · Python · Go · Git · Docs · Embedded 技能
                  │
          Workflow Layer          Plan · Review · Verify · Release
                  │
         Governance Layer         Hook · Lint · Policy · Style · Template
                  │
         Execution Layer          Git · 编译器 · 测试引擎 · CI ·（未来：容器）
                  │
        Project Repository        本仓库与受管项目仓库
```

分层的理由只有一条：**每层只对相邻下层说话，换掉任何一层的实现，其余层不动。**
这与内核六模块"少量稳定原语 + 简单组合"的取舍同源（见
[核心架构](AICODING_CORE_ARCHITECTURE.md)）。

## 2. 每层职责与仓库对应物

| 层 | 职责（一句话） | 本仓库的真实资产 | 权威文档 |
|---|---|---|---|
| User / IDE/UI | 人发起意图、看结果 | 不在本仓库（Codex CLI、Claude Code、IDE 插件） | — |
| Agent Runtime | 模型读上下文、调工具 | 不在本仓库；接口是 plugin marketplace（`.agents/plugins/marketplace.json`）与运行时 skill 根 | [CODEX_KIT_ARCHITECTURE](CODEX_KIT_ARCHITECTURE.md) |
| Context Management | 告诉 Agent"这个项目是什么、规矩是什么" | `AGENTS.md`、`.aicoding/memory/DECISIONS.md`、REPOSITORY_MAP 生成区块（源头 `config/repository-navigation.json`）、用户配置 | [02](02-context-architecture.md) |
| Skill | 多步工作流知识，按意图触发加载 | `CodingKit/agents/skills` 子模块（权威源码）、`.agents/skills/`（RepoLocal）、C99 kit | [03](03-skill-architecture.md) |
| Workflow | 计划→执行→验证→发布的流程约定 | 计划模式（Plan Mode）specialty 工具、SDD/BDD/TDD 技能、正式命令流 | [04](04-workflow-architecture.md) |
| Governance | 强制拦截：不合规的改动进不来 | `.githooks/` + `aicoding hook`、governance 五门禁、docsync、`.clang-format`、`.github/` 模板 | [05](05-governance.md) |
| Execution | 真正动系统的动作 | `internal/gitx`（git 进程唯一出口）、`go build`、`internal/testengine`、CI workflow、clang-format；Docker 当前**不存在**，列为未来执行器（未立项） | [GIT_REUSE_BOUNDARY](GIT_REUSE_BOUNDARY.md) |
| Project Repository | 事实本体 | 本仓库 + 受管项目仓库 | — |

**Go CLI 在哪？** `bin/aicoding` 不是单独一层，而是 Workflow、Governance、Execution
三层中一切**机器可执行动作的唯一入口**（唯一控制面）。CLI 内部另有自己的四层结构
（调用方 → porcelain 命令 → 领域 → plumbing 原语 → Git 事实层，见
[架构手册](ARCHITECTURE_HANDBOOK.md) §2）；本图是系统级视图，两者不冲突。

## 3. 依赖规则

1. **只向下**：上层可以调用相邻下层，永不反向。Skill 可以调 CLI 命令；
   CLI 永远不知道哪个 Skill 在调它。
2. **跨层要有名字**：跨层交互只走四种稳定形态——命令与 JSON 报告（CLI 契约）、
   manifest/registry（登记）、SKILL.md（知识）、hook（强制）。不允许私下 import。
3. **语义分层的执行版**是 `platform -> integration -> capability -> runtime`
   依赖治理（`aicoding governance dependencies --json` 机器校验，
   见 [依赖方向政策](../governance/DEPENDENCY_DIRECTION_POLICY.md)）。

## 4. CLI、Skill、Workflow、Governance 的边界

| 角色 | 管什么 | 不管什么 |
|---|---|---|
| CLI | 机器可执行动作 + JSON 证据（装、查、测、发） | 不承载多步编排知识，不猜用户意图 |
| Skill | 多步工作流的**知识**（何时做、按什么顺序、如何确认） | 不自带执行权，动作全部转调 CLI |
| Workflow | 流程**约定**（先计划后执行、先验证后声明完成） | 自己不留代码：知识在 Skill、强制在 Hook、执行在 CLI |
| Governance | **强制**（不合规的提交/依赖/布局直接拦下） | 只拦不修：修复权留给显式的写命令 |

## 5. 核心（Core）vs 插件（Plugin）

判定口诀：**卸掉它，其他领域还转吗？** 转 → 插件；停 → 核心。

| 核心（冻结物，改动走 ADR） | 插件/扩展（按登记进出，随时可卸） |
|---|---|
| 内核六模块：snapshot（`internal/registry`）、plan（`internal/runner.ExecutionPlan`）、runner（`internal/runner`）、adapter catalog（`internal/lifecycle`）、report（`internal/report`）、领域 state 规则 | 各 Kit（`config/kits/*.json` 登记） |
| 命令目录（`internal/cli` typed catalog）与八个 action 动词 | 各 MCP component（`config/mcp/components/*.json` 登记） |
| 唯一测试引擎（`internal/testengine`）与三档 profile | 各运行时 Skill（plugin 内嵌 / 独立 / 外部） |
| JSON 报告契约（schemaVersion/ok/errorKind/退出码） | PowerShell 专项工具（六类，已冻结不增长） |
| Git 复用边界（`internal/gitx` 唯一 git 进程出口） | 未来的 repo-context 生成物（受管资产） |

## 6. 地基命令面：内核命令及对应功能

这是平台的"git 基础命令"。约定：全部命令支持 `--json`（输出统一 JSON 报告）；
退出码 `0`=成功、`1`=检查或执行失败、`2`=参数错误。下文用 `aicoding` 指代
构建出的二进制（Windows 为 `bin\aicoding.exe`）。精确旗标与 Taskfile 短路由见
[命令矩阵](../COMMANDS.md)。

### 6.1 最短日常路径（类比 git init → add → commit → push）

```powershell
go run ./cmd/aicoding bootstrap --json                              # 1. 把 CLI 编译出来
bin\aicoding.exe lifecycle plan --action install --scope all --json # 2. 预览要装什么（不动系统）
bin\aicoding.exe lifecycle install --scope all --json               # 3. 真正安装
bin\aicoding.exe doctor --all --json                                # 4. 一键体检
bin\aicoding.exe verify --profile Smoke --json                      # 5. 静态验证
bin\aicoding.exe test --profile Smoke --json                        # 6. 官方测试
bin\aicoding.exe release verify --json                              # 7. 发布前结构检查
```

### 6.2 准备

| 命令 | 它具体做什么 |
|---|---|
| `bootstrap` | 检查/创建 `bin/` 目录，把 Go 源码编译成 `aicoding` 二进制。`--no-build` 只检查不编译。相当于"先把工具本体装好"。 |
| `provision` | 对目标目录执行幂等 `git init`、hook/本地 marker 接线，并放置不覆盖既有内容的最小 SDD 文档骨架；不安装 kit、不写平台 policy。 |

### 6.3 生命周期：`lifecycle <动词> --scope kit|mcp|runtime-skill|all`

一套动词管三类资产（kit、MCP component、运行时 skill）。`--scope` 必须显式给出。

| 动词 | 读/写 | 它具体做什么 | Git 类比 |
|---|---|---|---|
| `plan --action <A>` | 读 | 只计算"接下来会对哪些组件做什么"的清单并给出 planDigest，**不改任何文件**。先看后做。 | `git status`（看将要发生什么） |
| `install` | 写 | 按注册表把选中资产装到本机，只创建自己拥有的资产。 | — |
| `update` | 写 | 把已装资产收敛到注册表当前登记的版本；同一身份，不产生平行副本。 | `git pull`（收敛到最新事实） |
| `status` | 读 | 对比"注册表登记的"和"本机实际存在的"，逐项报告一致或漂移。 | `git status` |
| `doctor` | 读 | 诊断已装资产哪里坏了（缺文件、指错目标、版本漂移），**只报告不修复**。 | — |
| `verify` | 读 | 验证已装资产是否满足契约（结构、schema、指向），不碰状态。 | `git fsck` |
| `uninstall` | 写 | 只删除登记过且所有权检查通过的资产，**永不"全局清理"**。 | — |
| `rollback --scope kit --last` | 写 | 把上一次 kit 写操作回滚到快照（当前仅 kit 域支持）。 | `git revert` 的运行时版 |

### 6.4 体检与验证

| 命令 | 它具体做什么 |
|---|---|
| `doctor --all` | 一条命令做五项只读体检：仓库状态、kit、MCP、运行时 skill、PowerShell 预算。哪项失败报哪项，不修复。 |
| `verify --profile Smoke\|Full\|Release` | 跑十余项确定性静态检查（git hooks、仓库文本、release notes、治理、依赖方向、目录布局、复用证据、docsync、kit/skill/MCP 注册表、运行时 skill）。**不执行测试、不改状态**。 |

分工一句话：`doctor` 查"环境和安装现状"，`verify` 查"仓库内容和契约"，`test` 才是跑测试。

### 6.5 测试

| 命令 | 它具体做什么 |
|---|---|
| `test --profile Smoke\|Full\|Release` | 唯一官方测试引擎。跑完把 `report.md` + `summary.json` + `results.json` 写进 `test-results/aicoding-global-test-<时间戳>/`，退出码即结论。三档只有覆盖面差异，无第四档。 |
| `test latest` | 找到最近一次 `test-results/` 目录，重印那次的摘要。只读。 |

### 6.6 发布

| 命令 | 它具体做什么 |
|---|---|
| `release verify` | 发布前的结构快检（release notes、tag 政策等）。**不建 tag、不发 release**。 |
| `release gate` | 发布测试门禁，等价于 `test --profile Release`。 |

### 6.7 领域与专项命令（辅助面，一句话表）

以下命令服从同一 JSON 契约，是地基之上的领域面，不构成第二产品入口：

| 命令族 | 它具体做什么 |
|---|---|
| `kit list\|doctor\|verify` | 列出/诊断/验证 kit 注册表本身。 |
| `mcp list\|status\|doctor\|verify` | MCP 组件清单与只读检查；**写操作一律走 `lifecycle --scope mcp`**。 |
| `skill verify` | 校验技能结构完整性。 |
| `skill c99-standard-c status\|templates\|fmt\|check\|verify` | C 代码风格套件：查状态、验模板、格式化（写）、检查（只读）、宿主验证。 |
| `capability list\|describe\|index --write` | 从 `config/internal-capabilities.json` 查询 28 个内部能力，或只刷新 README 与 `docs/CAPABILITIES.md` 的生成索引。 |
| `governance lint\|dependencies\|layout\|reuse\|capabilities` | 五个治理门禁：提交规范、依赖方向、目录布局、复用证据、能力孤儿与文档义务。 |
| `docsync staged\|all\|ci\|release` | 文档同步门禁：代码改了文档没改就拦下。 |
| `export --all --zip` | 把平台资产打包成一个 zip。 |
| `fresh-clone --profile <P>` | 把本仓库临时克隆到干净目录并跑门禁，证明"新机器拿到就能用"。 |
| `tag audit` | 只读审计 tag 命名空间是否合规。 |
| `cache status\|clean` | 观测五类已注册本地生成物；`temp` 由 Git common-dir 追加式 ledger 登记，并与失败证据、Receipt/alias、跨 worktree 和审计轨迹保护规则共用既有保留权威。 |
| `codex usage parse\|run` | 解析/采集 Codex token 用量并输出标准报告。 |
| `hook pre-commit\|commit-msg` | 给 git hook 用的聚合检查入口（五项并行检查/提交信息校验）。 |
| `powershell regex-lint` | PowerShell 正则快检。 |

### 6.8 与 Git 的分工

仓库事实的编辑权 100% 归 Git，AiCoding 零包装：

| 你要做的事 | 用什么 |
|---|---|
| 初始化、暂存、提交、分支、合并、推拉 | `git init/add/commit/branch/merge/push/pull`，原样使用 |
| 查"装没装、坏没坏、测没测过"（Git 表达不了的本机运行时状态） | AiCoding 地基命令面（上文） |
| 回滚 | 事实层用 `git revert`；运行时层用 `lifecycle rollback`——两者由 Skill 编排，永不合并 |

详见 [Git 复用边界](GIT_REUSE_BOUNDARY.md)。
