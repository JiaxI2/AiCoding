# AiCoding 架构手册（Owner 视角）

Status: Derived View（派生视图）

> 本文面向仓库 owner，解释设计理由、升级决策框架和扩展路径。它不定义任何新契约，
> 全部内容可由文末权威文档重建；与契约文档冲突时，以契约文档为准。
> 类比：本文之于架构文档，如同 Git 的用户手册之于 Git 源码中的格式定义。

## 1. 一句话模型

> AiCoding = 规范化事实快照（snapshot）+ 确定性意图（plan）+ 领域无关调度（runner）
> + 静态领域翻译（adapter）+ 可验证证据（report）+ 领域自有状态（state），
> 坐在 Git 事实层之上，只处理 Git 无法表达的本机运行时领域。

对照 Git 的一句话模型（内容寻址对象库 + Merkle DAG + 可移动 refs + index + 传输协议），
两者结构同源：**少量稳定原语 + 简单组合方式 + 清晰边界**。

## 2. 分层地图

```text
L4  调用方        Agent / Skill / CI / Hook             （Git 类比：GUI、IDE 插件）
L3  porcelain    aicoding CLI typed commands            （git add / commit / push）
L2  领域          kit / mcp / runtime-skill              （refs 与 index 的领域规则）
L1  plumbing     snapshot / plan / runner / report      （hash-object / write-tree / update-ref）
L0  事实层        Git 本体：对象库、refs、传输、gitlink    ← 不包装，直接用
```

三条方向规则：

1. 依赖只向下：L4 → L3 → L2 → L1 → L0，永不反向。
2. L1 对 L0 是消费者关系，不是适配关系（没有也永远不会有 `git` adapter）。
3. 每层可以单独替换实现，只要公开契约不变（见 §5 验证半径）。

## 3. 内核设计

### 3.1 六个正交模块

| 模块 | 唯一职责 | Git 对应物 | 为什么不能与相邻模块合并 |
|---|---|---|---|
| snapshot | 规范化事实、内容树、digest | blob/tree + OID | 合并进 plan 会让"事实"随"意图"漂移，digest 失去可比性 |
| plan | 确定性意图、选择、digest | index（候选状态） | 合并进 runner 会让调度器理解业务，变成 God Core 的第一步 |
| runner | timeout、cancel、有界并发、稳定顺序 | git 的进程/事务外壳 | 它领域无关才能被任何领域复用；认识 Kit/MCP 即污染 |
| adapter | 统一 action → 领域调用的静态翻译 | porcelain → plumbing 的映射 | 承载业务策略就成了第二领域实现 |
| report | envelope、证据、错误分类 | commit + 可追溯证据 | 能触发修复就有了执行权，证据与执行必须分离 |
| domain state | 领域自有状态与 rollback | refs 的所有权 | 全局状态仓库 = 无法回答"谁拥有、失败恢复什么" |

连接方式只有四种：不可变值对象、descriptor、snapshot、result。
禁止：模块间函数回调、共享可变全局、跨领域状态写入。

### 3.2 关键设计决策与依据

**决策一：不设统一 Core。**
六模块独立而非合并为单一核心。依据：合并后任何一处优化都要求理解全部职责，
验证半径永远是全量；独立则各模块可单独替换实现。对照案例：Git 的
`blob/tree/commit/ref` 各自独立，packfile 格式迭代十余版、传输协议升级到 v2、
gc 策略多次重写，快照语义始终未变。

**决策二：adapter 静态编译，不引入动态插件。**
依据：动态加载使"本次运行执行了哪份代码"不可回答，破坏运行证据的可对账性；
静态 catalog 与 digest 使每次运行可完整追溯。系统的组合自由来自稳定组件的
自由组合，而非身份与来源的不确定性。

**决策三：plan 与 apply 共用同一领域路径（plan = action + dryRun）。**
依据：预览与执行若为两套实现则必然漂移（plan 判定可安装而 apply 失败）。
对照案例：Git 的 index——`git add` 写入的即是 commit 将使用的 tree 条目，
不存在独立的预览格式。

**决策四：digest 为三元组（catalog/input/plan），不合并为单一摘要。**
依据：三者回答三个独立问题——使用了哪组契约、针对什么事实、执行什么意图。
合并后无法定位漂移来源。对照案例：Git 区分 tree OID 与 commit OID，
分别标识内容与历史位置。

### 3.3 内核清单（冻结物）

以下五项是内核，修改必须走 ADR + 三条件（现实问题 + 稳定变化点 + 两个真实消费者）：

1. 六模块职责与依赖禁令；
2. 八个 action 动词表（见 §4.2）；
3. digest 三元组语义与 `report.Result` JSON 契约（schemaVersion、ok、errorKind、退出码）；
4. gitx 宪法五条与 Git 复用边界；
5. 三条可执行门禁（git 进程所有权、gitx importer 白名单、porcelain 动词禁用）。

以下**不是**内核，按验证半径自由优化：各模块内部实现、governance 扫描算法、
领域模块内部逻辑、新增 Kit/MCP/Skill 登记、性能与缓存策略。

### 3.4 核心原语与自由度

六模块升格为原语视角陈述——这是 AiCoding 的专属能力清单：

| # | 原语 | 一句话定义 | Git 对应物 | 代码位置 |
|---|---|---|---|---|
| ① | 规范化快照 | 任意事实 → 内容树 + 稳定 digest | hash-object / write-tree | `internal/registry` |
| ② | 确定性意图 | 选择 + action → 可摘要、不可变的 plan | index / commit-tree | `internal/runner.ExecutionPlan` |
| ③ | 领域无关有界执行 | plan → 带 timeout/cancel/稳定顺序的结果 | **无（专属）** | `internal/runner` |
| ④ | 证据信封 | 结果 + digest 三元组 → 可对账 JSON | commit + 可追溯证据 | `internal/report` |
| ⑤ | 所有权判定 | 写操作只触碰登记且属于自己的资产 | refs 所有权 | 各领域 state 规则 |
| ⑥ | 静态翻译登记 | 统一动词 → 领域调用的编译期映射 | porcelain → plumbing | `internal/lifecycle` catalog |

③ 是 Git 没有的独创：Git 对象是被动数据，AiCoding 必须执行有副作用的外部操作。
③④⑤ 连起来构成"**副作用执行的确定性**"——执行前意图可摘要、执行中有界、
执行后可对账、越界被所有权拦住。这是"让 AI 安全地动真实系统"的最小原语集，
也是本仓库区别于 Git 的核心价值。

**通用性检验**：原语类型签名里没有 Kit/MCP/Skill/Codex/模型等任何时代性名词，
已由三个异构领域（Go 内置、Python venv、PowerShell specialty）同时消费证明。
外界 AI 变化（新工具、新 Agent、新分发形态、新运行时、模型更迭、协议换代）
全部落在登记层/调用方层/领域 adapter 层吸收，内核零改动——协议换代的构造性
证明即 runtime-skill 领域进入时六模块零修改的先例。

**高效性立场**：核不需要 C，但必须简单通用高效。当前热点是进程启动、磁盘与
子进程而非纯计算（见核心架构 §8 的测量结论），Go 单二进制 + 一次
parse/normalize/digest + detached snapshot 是正确取舍；C 出口不焊死，五条件
齐备时可进——如同 Git 用 C 是 1991 年的测量结论，不是信仰。

**自由度的严格定义**：不是"什么都能改"，而是"变化被引导到预留出口，原语永不
重写"。三个已识别的极限场景及其出口：

| 极限场景 | 出口位置 | 原语是否存活 |
|---|---|---|
| 流式/交互式执行（进度流、中途审批） | `report.Result` 传输形态扩展（如增量事件），类比 Git protocol v1→v2 换传输不换对象 | ④ 存活，信封投递方式演进 |
| 真正的多 Agent 并发写 | 补 expected-digest 守卫（控制面 §6.1 预留的定义方式） | ⑤ 存活，加守卫不重写 |
| 无法快照的事实（远程托管、不可 inspect） | 分类吸收：可快照部分归 input facts，其余归 mutable observation（runtime-skill 先例） | ① 存活，事实被分类而非破坏 |

## 4. 核心指令集设计

### 4.1 正式入口（porcelain）

```text
bootstrap                              初始化
lifecycle plan|install|update|uninstall|status|doctor|verify|rollback
doctor --all                           环境诊断
verify --profile Smoke|Full|Release    静态/结构验证
test --profile Smoke|Full|Release      测试执行（唯一测试引擎）
release verify|gate                    发布门禁
```

领域子命令（kit/mcp/skill/governance/docsync/export/fresh-clone/powershell）服从
同一 JSON 契约。兼容期已结束，旧命令不再路由；迁移映射保留在 `docs/COMMANDS.md`。

### 4.2 八个 action：动词表的设计逻辑

| Action | Effect | 设计理由 |
|---|---|---|
| plan | read | 意图与执行分离；Agent 先拿到可审计计划再请求批准 |
| install | write | 从登记事实建立 owned runtime，只创建自己拥有的资产 |
| update | write | 同 identity 收敛，不创建平行身份；"sync"是它的内部步骤，不是新动词 |
| uninstall | write | 只删除登记且 ownership 检查通过的资产，永不"全局清理" |
| status | read | 对比登记事实与本机现实 |
| doctor | read | 诊断不修复——修复权留给显式 write action |
| verify | read | 确定性契约验证，不碰状态 |
| rollback | write | 恢复领域自有证据，不是跨域业务补偿 |

动词表的稳定性承诺与 Git 相同：**只增不改**。`git add` 的语义 20 年未变，新能力
（如 sparse-checkout）以新命令进入而不改旧动词。AiCoding 同理：新需求优先问
"是不是某个动词的领域内部步骤"，答案是否才考虑经 ADR 增加动词。

### 4.3 与 Git 指令的分工（你给出的核心/扩展清单）

你的核心清单（init/add/commit/remote/push/pull）与扩展清单（branch/checkout/merge）
**全部**留在 Git，AiCoding 零包装。对应关系：

| 你的 Git 动词 | AiCoding 侧的对应物 |
|---|---|
| init/add/commit | 无——仓库事实的编辑权完全归 Git |
| remote/push/pull | 无——同步与并发保护归 ref 语义 |
| branch/checkout/merge | 无——AiCoding 的"轻量可移动选择"是 profile/registry entry，不是分支包装 |
| cat-file（探索 API） | `... list --json` + digest 重算对账（事实未变时可反查） |
| revert | git revert（事实层）+ 领域 rollback（运行时层），由 Skill 编排，永不合并 |

详见 [Git 复用边界](GIT_REUSE_BOUNDARY.md)。

## 5. 升级与优化：决策框架

### 5.1 变更分类（先分类，再决定验证半径）

| 类别 | 例子 | 验证半径 | 是否触碰架构 |
|---|---|---|---|
| 功能扩展 | 新 Kit/MCP/Skill 登记 | 领域测试 + lifecycle consumer | 否 |
| 模块维护 | snapshot 规范化算法、runner 调度、gitx 内部 | 模块契约测试 + 直接消费者 | 否 |
| 公开契约变化 | JSON 字段语义、action 语义、digest 覆盖范围 | CLI contract + Full/Release | 是，走 ADR |
| 边界变化 | 新领域 adapter、放行 porcelain 动词 | Full + 治理更新 | 是，走 ADR + 三条件 |

判断口诀：**改动能否被消费者观察到？** 不能 → 模块维护，局部验证；能 → 契约变化，
全量验证。这就是"单独优化一条 plumbing 路径"的可操作形式。

### 5.2 性能优化规则

1. 先测量（可重复 profile），后优化；没有热点证据的优化提案直接拒绝。
2. 逻辑语义先冻结，物理优化在语义之下自由进行（Git 的 snapshot 语义 vs packfile 实现）。
3. C/Rust/native 必须同时满足五条件（热点在纯计算内核、Go 优化已到预算上限、
   两个真实消费者、同一 golden tests、收益覆盖构建/供应链成本）——当前全部不满足。
4. 只读路径并行化需要基准证据 + 顺序保持 + 领域线程安全声明三者齐备。

### 5.3 冻结与解冻

- 解冻唯一路径：ADR + 现实问题 + 稳定变化点 + 至少两个真实消费者。
- "未来可能需要"不是理由；空接口、兼容层、预建抽象是未来债务，不是投资。
- 每次架构提案先回答：为什么改？是否减少总复杂度？是否破坏稳定边界？
  三问任一答不出即停止。

## 6. 可扩展性：四条路径的成本阶梯

从便宜到昂贵，扩展需求应落在尽可能靠上的一级：

```text
① 新 Kit / MCP component     只加 manifest + registry entry + 领域测试（不碰内核）
② 新 external Skill          Codex-Skills submodule pin + binding + 运行时映射（跨仓库登记）
③ 新领域 adapter             新领域模块 + 静态 descriptor（内核不变，catalog 增一行；需 ADR）
④ 内核契约修改               ADR + 三条件 + Full/Release（极少发生）
```

设计意图：**95% 的需求应停在 ①②**。如果一个需求看起来要 ③④，先对抗性追问：
它真的不能表达为现有领域的 manifest 变体吗？runtime Skill 领域已经证明了 ③ 的
可行性——它进入时六模块零修改。

明确不提供的扩展点（拒绝清单摘要）：动态 Go plugin、第二 CLI/lifecycle/test engine、
capability graph、全域事务、远程控制 API、跨领域 SystemManager。完整清单见
[核心架构](AICODING_CORE_ARCHITECTURE.md) §11。

### 6.1 用户定制阶梯

用户定制与功能扩展同构但独立成梯。铁律：

> 用户定制永远流经输入（参数、配置文件、IR），永远不进入 owned 资产
> （plugin cache、junction、managed block、内核代码）。

两个免费收益：升级安全（update 只收敛 owned 资产，用户定制天然不被碰）；
可复现（定制流经输入即自动进入证据链，藏在环境里的定制才会让 Run 无法对账）。

```text
① 调用参数层    单次生效：CLI flag、tool 参数、Diagram IR 字段
② 用户配置层    持久偏好：有 schema 的用户配置文件（.gitconfig 的对应物）
③ 组合层        新工作流：用户 Skill 编排正式命令 / MCP tools
④ 登记层        新能力：新 registry entry（Kit / component / skill source）
⑤ 源码层        改能力行为：上游改 → 提交 → 更新 pin → install/update
```

永远关闭的进入点：直接编辑 plugin cache 或 junction 目标（等于直接改
`.git/objects`）、为单个用户加 CLI 顶层命令、在 capability 源码打本地补丁不提交上游。

典型场景落位：

| 场景 | 层级 | 方式 |
|---|---|---|
| 常用命令组合 | ③ | 用户 Skill 走 Draft → RepoLocal → Kit 阶梯；不加 CLI 命令 |
| 命令短名 | （未来出口） | git-alias 式用户配置，展开为既有正式命令，过同一 porcelain 禁用集合；真实重复出现前不建 |
| Skill 风格/偏好 | ② | `config/skills/<id>/` 配置文件（c99 先例），改数据不改源码 |
| Skill 行为逻辑 | ⑤ | 完整 pin 链；绝不编辑已安装副本 |
| 个人专属 Skill 变体 | ③/④ | 独立身份登记，同名 active 冲突由 audit 拒绝 |
| visio-mcp 单图风格 | ① | 风格字段进 Diagram IR，capability 保持输入的纯函数 |
| visio-mcp 持久风格 | ② | 用户风格 profile（schema 归 capability，值归用户，存非 owned 位置），由上层 Skill 合并进 IR；manifest env 注入是降级备胎 |

可验收不变量：**对任何组件执行 update/uninstall，② 层用户配置与 ③ 层用户 Skill
必须字节不变**。配置格式迁移只能走 schemaVersion 升级 + 显式迁移工具，
永远不是 update 的隐式副作用。

## 7. Agent 知识面

Agent 知识面（agent knowledge surface）指 Agent 在本仓库中获取"应该做什么、
怎么做、做错了如何纠正"的全部信息来源。本节定义其构成、增长策略与治理机制。

设计目标：**功能数量可以无限增长，知识进入点数量保持恒定。** 会话中的口头约定
不属于知识面——未持久化到下列进入点的知识，对下一次 Agent 会话不存在。

### 7.1 四个知识进入点

| # | 进入点 | 承载内容 | 权威位置 | 增长策略 |
|---|---|---|---|---|
| ① | AGENTS.md | 仓库总纪律：会话开始必读的边界、指向其余知识的索引 | 仓库根目录 | 只增索引行，不承载正文；总量受控 |
| ② | SKILL.md 集合 | 多步工作流知识：按 frontmatter description 匹配用户意图后加载 | 运行时 Skill 根（经 lifecycle 暴露） | 按工作流增加，每份经 verify 门禁与同名审计 |
| ③ | CLI 自描述 | 命令知识：`help` 输出、typed command catalog、JSON 报告的 `schemaVersion`/`ok`/`errorKind` 契约 | `internal/cli` catalog 与 `internal/report` schema | 随正式命令同步维护，形态统一 |
| ④ | 门禁错误信息 | 纠错知识：违规操作的失败信息必须指明违反的规则与正确路径 | 各 governance/audit/contract 检查的 error 文案 | 随门禁同步维护 |

四个进入点各有唯一职责，不互相复制内容：AGENTS.md 不展开工作流细节（指向
Skill）；SKILL.md 不复制命令参考（指向 CLI help）；错误信息不重述完整政策
（指向文档）。重复即漂移的开端。

### 7.2 知识与功能的解耦

统一 action 动词表（§4.2）是知识面保持恒定的机制基础：Agent 掌握一次
`plan/install/update/uninstall/status/doctor/verify/rollback` 与 JSON 判读规则，
即可操作全部现存与未来领域。各类新增对知识面的影响如下：

| 新增内容 | 知识面变化 | 说明 |
|---|---|---|
| 新 Kit / MCP component | 无 | 八动词与 `--scope` 语法不变；新组件自动出现在 `list`/`status` JSON 中，属于数据变化而非接口变化 |
| 新 external Skill 登记 | 无 | 同上；运行时发现由 lifecycle 暴露机制完成 |
| 新领域 adapter | 无 | 仅新增一个 `--scope` 取值；动词表与 JSON 契约复用 |
| 新的多步工作流 | ② 增加一份 SKILL.md | 按工作流切分，不按对象类型切分；须过 verify 门禁 |
| 新 CLI 子命令（罕见，须 ADR） | ③ 增加 help 条目与 JSON 契约 | 命令形态必须与既有目录一致 |
| 新门禁 | ④ 增加错误文案 | 文案须满足 §7.4 检查项 4 |

该表的对照原型是 Git：子命令数量持续增长，但命令形态、manpage 结构与退出码
语义恒定，用户的边际学习成本趋近于零。

### 7.3 知识资产的生命周期（自举）

SKILL.md 形态的知识本身是本平台管理的一类资产，其治理复用包管理机制，
不设第二套流程：

- **准入**：`aicoding-skill.ps1 verify` 校验 frontmatter、when-to-use/when-not-to-use、
  验证命令、安全边界的完整性；不合格不得进入运行时。
- **唯一性**：运行时审计拒绝同名 active Skill，保证任一工作流主题只有一份
  生效知识，不出现互相矛盾的多份指引。
- **成熟度阶梯**：Draft（`.aicoding/user-skills/`）→ RepoLocal（`.agents/skills/`）
  → Kit（上游 Codex-Skills 收编）。试用、稳定、正式三阶段，任一阶段可卸载。
- **可盘点**：`lifecycle status` 可枚举当前运行时暴露的全部知识资产及其来源。

该机制的目的是防止知识堆积腐化：无生命周期管理的 agent 指令文件
（数千行的单体 AGENTS.md/CLAUDE.md）是已知反模式；本仓库中知识是有身份、
可验证、可卸载的对象。

工作流知识的现存实例：`aicoding-external-integration`（外部 skill/MCP/kit/tool
的集成决策与执行流程，含 pin、fork-pin、adopt 三路径），遵循本节全部规则。

### 7.4 新功能的知识检查（DoD 扩展）

任何新功能（新命令、新领域、新组件、新门禁）的完成定义必须包含以下四项
检查，与 §5.1 的验证半径检查并列：

1. **可发现**：命令进入 typed catalog，`help` 可见；组件进入 registry，
   `list`/`status` 可枚举。
2. **可判读**：JSON 报告契约齐全，Agent 能以 `schemaVersion`/`ok`/`errorKind`
   程序化判断结果，无需解析人类文本。
3. **可遵循**：涉及多步编排或用户决策的功能，对应 SKILL.md 已存在或已更新。
4. **可纠错**：门禁与校验的失败信息指明违反的规则和正确路径，而非仅报告失败。

四项均不满足的功能对 Agent 不存在；部分满足的功能会以支持成本的形式
持续收税。

## 8. 权威文档地图

| 文档 | 管什么 | 什么时候读 |
|---|---|---|
| [AICODING_CORE_ARCHITECTURE](AICODING_CORE_ARCHITECTURE.md) | 六模块契约、冻结条件、拒绝清单 | 评审任何架构提案前 |
| [CLI_MCP_CONTROL_PLANE](CLI_MCP_CONTROL_PLANE.md) | 唯一控制面、Agent API、JSON 证据契约 | 写 Agent/Skill 集成时 |
| [EXTENSION_ADAPTER_CONTRACT](EXTENSION_ADAPTER_CONTRACT.md) | adapter descriptor、action 契约、扩展步骤 | 走扩展路径 ①②③ 时 |
| [FREEZE_AND_ACQUISITION_BOUNDARY](FREEZE_AND_ACQUISITION_BOUNDARY.md) | 四项契约冻结、获取/激活分离与可执行门禁 | 演进冻结面或接入外部来源时 |
| [GIT_REUSE_BOUNDARY](GIT_REUSE_BOUNDARY.md) | Git/AiCoding 分工、gitx 宪法、三条门禁 | 任何涉及 git 调用的改动前 |
| [KIT_LIFECYCLE_ARCHITECTURE](KIT_LIFECYCLE_ARCHITECTURE.md) | Kit 领域 manifest 模型与生命周期 | 增改 Kit 时 |
| [MCP_CONTROL_PLANE](MCP_CONTROL_PLANE.md) | MCP 领域边界与验证 profile | 增改 MCP component 时 |
| [CODEX_KIT_ARCHITECTURE](CODEX_KIT_ARCHITECTURE.md) | 仓库所有权、submodule 链、依赖方向 | 跨仓库变更时 |
| [POWERSHELL_BOUNDARY](POWERSHELL_BOUNDARY.md) | PowerShell 专项保留边界 | 想写 .ps1 之前 |
| 本文 | 设计理由与决策框架（派生视图） | 需要"为什么"而不是"是什么"时 |

## 9. Owner 检查清单

收到任何架构提议（无论来自 Agent 还是人）时，依次问：

1. **归属**：过 Git 三问判据（可由 commit 重建？并发即 ref 移动？涉及本机可变资源？），
   它属于 Git 域还是 AiCoding 域？Git 域 → 直接用 git，提案结束。
2. **层级**：它落在扩展阶梯 ①②③④ 的哪一级？能不能降一级表达？
3. **消费者**：第二个真实消费者存在吗？不存在 → 具体实现，不抽象。
4. **半径**：改动是模块维护还是契约变化？验证计划与之匹配吗？
5. **对账**：改完之后，给定一个 Run 的 JSON 证据，还能回答"对什么事实、
   用什么契约、执行了什么意图"吗？

五问全部通过才进入实现；任何一问卡住，先解决问题本身而不是扩大架构。
