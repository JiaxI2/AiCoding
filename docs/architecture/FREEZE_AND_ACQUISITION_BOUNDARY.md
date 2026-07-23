# 契约冻结与获取/激活边界

Status: Accepted and Frozen

## 1. 结论

本文完成发布后的两项收敛：

1. 将九个已被多方消费、语义稳定的契约面正式冻结（§2）；
2. 定义外部来源（GitHub/URL 起源的 Skill、MCP、Kit）的获取/激活分离边界与
   标准接入流程，使"慢"被隔离在唯一的获取步骤中，激活保持本地快速（§3）。

原则与 Git 相同：Git 快不是因为网络快，而是因为几乎所有命令都是纯本地操作，
网络被隔离在 `fetch/push` 两个显式动作里。本仓库将同一结构固化为契约。

## 2. 九项冻结声明

### 2.1 JSON 报告契约

- **冻结面**：`report.Result`、`StandardReport`、共享 check 结构、
  `config/schemas/cli-report.schema.json`、退出码 0/1/2、`errorKind` 枚举语义。
- **规则**：schemaVersion=1 冻结。只允许新增可选字段；删除、改名或改变既有
  字段语义必须提升 schemaVersion 并通过 ADR。
- **理由**：正式发布后该契约是 Agent、Skill、CI、Hook 的共同判读面，属于对外承诺。

### 2.2 测试三档语义

- **冻结面**：Smoke/Full/Release 三个 profile 的副作用边界
  （Smoke/Full 不启动可见外部工具，Release 才执行真实桌面回归）。
- **规则**：不新增第四档。新的验证需求以 registry 条目进入三档之内；
  档位语义变化必须通过 ADR。
- **理由**：三档已被 CI、组件 manifest、测试引擎与文档四方引用，变化点已稳定。

### 2.3 config schema 集

- **冻结面**：`kit-manifest`、`kit-registry`、`mcp-component`、`mcp-registry`、
  `dependency-governance` 五份 schema。
- **规则**：schemaVersion=1 冻结，扩展只允许新增可选字段；必填项与既有字段
  语义变化必须提升 schemaVersion 并通过 ADR。
- **理由**：每份 schema 均有两个以上真实消费者（领域实现 + governance + 契约测试）。

### 2.4 Taskfile 纯路由地位

- **冻结面**：Taskfile 的职责定义——每个 task 一对一映射既有 CLI 命令或已登记
  specialty 脚本，不承载业务逻辑、判断或组合语义。
- **规则**：本项当前为声明性冻结，不新增门禁；出现第一次真实违规证据后再以
  ADR 升级为可执行检查（遵循停止规则，不为推测预建）。

### 2.5 Loop Engineering 裁决面

- **冻结面**：Loop 只裁决下一步，不拥有尝试执行；typed catalog 的 `work` 入口保持
  `validate / next / status / record`，不得出现 `run / prepare / step`。
- **规则**：`transition.Decide` 保持 `spec / history / gates / now` 四参数事实注入与
  `(Decision, error)` 返回，不读取隐藏全局状态。
- **理由**：Agent 执行与 AiCoding 裁决的边界已经由 CLI、架构图和端到端 quickstart 共同消费。

### 2.6 Plan Mode 批准树与 scope

- **冻结面**：Plan 批准 clean `HEAD^{tree}`，以 `approvedTree` 和显式 scope 判定后续漂移；
  pre-commit 对敏感路径执行 fail-closed 覆盖检查。
- **规则**：Plan 不执行实现、不签发 Receipt、不调度 Loop；改变这三条边界必须走 ADR。
- **理由**：CLI、pre-commit、Validation Evidence 与 Loop 已共同依赖内容树批准语义。

### 2.7 Validation Evidence 核心 Receipt 契约

- **冻结面**：`Fingerprint` 的身份字段清单、完整 PASS 才可签发 Receipt、查询/完整性失败
  fail-closed，以及失败结果不进入 Receipt cache。
- **规则**：身份仍由 repository、Tree、profile、plan、engine、config、toolchain、options
  及既有节点字段组成；不得以 commit、时间戳或失败结果替代。
- **理由**：TestEngine、Context Gate、push gate 和复用审计均消费同一内容身份。

### 2.8 Kit pinned reference source

- **冻结面**：Kit manifest v2 的 `source` 是可选输入；Git 只接受完整 40-hex commit，
  content 只接受 SHA-256 digest，缺省时保持旧 manifest 的仓库内路径语义。
- **规则**：register/prefetch 属于获取阶段，install/update 只从本地内容寻址 cache 物化；
  cache 缺失必须 fail-closed，不得静默联网。
- **理由**：registry、Kit lifecycle、pins cache 与 Validation Evidence 已共同消费该语义。

### 2.9 Typed 子命令与 alias 唯一登记面

- **冻结面**：`internal/cli` typed command catalog 不仅登记顶层命令，还唯一登记递归子命令、
  alias、help route 与 pluginview quickstart route。此项是新增冻结面，不追溯解释为 2.5 的
  顶层/`work` 登记已经覆盖。
- **规则**：外部 argv 必须先经 catalog 解析和 alias 规范化；handler 只按 typed
  `SubcommandID` 分派。help 命令路径与 pluginview quickstart 命令路径只能由同一 descriptor
  投影，不得在领域层另写可路由命令字符串。
- **理由**：CLI routing、用户帮助和 Kit pluginview 已共同消费子命令路径，登记面已达到冻结条件。

## 3. 获取/激活边界

### 3.1 定义

```text
获取平面（acquisition）   网络动作：clone、submodule update、包下载、pin 更新
激活平面（activation）    本地动作：install / update / uninstall / status / doctor / verify
```

**不变量**：来源一旦本地化（submodule 已检出、源码已 vendor、依赖已缓存），
激活平面的全部操作必须可离线完成。网络只允许出现在获取平面。

### 3.2 外部来源标准接入流程

所有 GitHub/URL 起源的 Skill、MCP、Kit 一律走四步，与集成决策
（pin / fork-pin / adopt 三路径）正交组合：

| 步骤 | 动作 | 网络 | 权威面 |
|---|---|---|---|
| ① 登记 | URL + pin + trust 写入获取登记面 | 无 | `.gitmodules`、`config/skill-sources.json` |
| ② 取回 | 将源实体化到本地（submodule 检出 / clone / vendor） | **唯一网络步骤** | Git 原生传输 |
| ③ 核验 | 本地内容与 pin 对账 | 无 | Git OID（内容寻址天然完成）；包依赖以 lock/hash 对账 |
| ④ 激活 | lifecycle install/update 从本地事实执行 | 无（目标） | AiCoding lifecycle |

### 3.3 加速插槽

加速手段（镜像、代理、缓存）只允许作用于步骤②，且只通过底层工具的原生机制
实现（git `insteadOf`/proxy、pip index/cache 配置、包管理器缓存），配置位置为
用户环境或获取登记面数据。

禁止：把镜像/代理/下载逻辑写入激活面 manifest；为加速新增 AiCoding 下载器、
镜像封装或 CLI 命令。加速是获取步骤的可替换件，有唯一插槽，不允许渗入其他层。

### 3.4 已识别偏差与收敛出口

当前 MCP install 在激活期间执行 pip 依赖下载（获取混入激活）。处理：

- 本偏差被显式登记，不视为违规；manifest 侧已满足 URL-free（依赖声明经
  requirements/pyproject 本地文件表达）；
- 收敛出口为领域内部优化：wheel 本地缓存或 uv 评估，须先有可重复测量，
  验证半径 = `internal/mcpcontrol` + component verify，不触碰任何契约；
- 完全离线激活作为验收抽查项（断网 spot-check），不进入默认门禁。

## 4. 可执行门禁（八条）

1. **激活面 URL-free**（governance dependencies 新 check）：
   `config/kit-registry.json`、`config/kits/*.json`、`config/mcp-registry.json`、
   `config/mcp/components/*.json`、`config/codex-kit.json` 的任何字符串值不得
   含 `://`。激活所需的一切必须以仓库相对路径表达。
2. **可克隆源只在获取登记面**（governance dependencies 新 check）：
   在 `config/**` 与 `.gitmodules` 范围内，可克隆 git 源 URL（`*.git` 结尾或
   仓库根形态的 github/gitcode 地址）只允许出现在 `.gitmodules` 与
   `config/skill-sources.json`。文档链接、badge 权威 URL、schema `$id` 不属于
   可克隆源，不在本检查范围。
3. **FREEZE-004：Loop 无执行入口**：typed catalog 不得登记 `work run|prepare|step`。
4. **FREEZE-005：Decide 注入面**：AST 断言 `Decide` 的四参数与返回类型不漂移。
5. **FREEZE-006：Receipt 身份字段**：AST 断言 `Fingerprint` 字段清单与顺序不漂移。
6. **FREEZE-007：pinned source 可选性**：schema 必须声明 `source`，且顶层
   `required` 集合不得包含它。
7. **FREEZE-008：typed 子命令唯一登记**：所有带子命令的 handler 必须经 catalog
   解析为 `SubcommandID`，`Execute` 必须先执行 catalog guard；字符串 case 或 catalog 外
   `args[0]` 路由直接失败并点名 handler/子命令。
8. **FREEZE-009：产品 profile 词汇表**：`--profile` 固定为 Smoke/Full/Release，flag help
   与运行时规范化器必须消费同一词汇表；第四档或独立 help 词汇直接失败。

八条检查在当前 main 上应当零违规通过；FREEZE-008 是 ADR 0012 新增冻结面，FREEZE-009
把既有 2.2 三档契约升级为可执行检查。

## 5. 明确拒绝

- 第四个测试 profile；
- 激活面 manifest 中的任何网络 URL 或镜像配置；
- AiCoding 自建下载器、镜像管理器、加速 CLI 命令；
- 以"加速"为由绕过登记面直接 clone 到运行时位置；
- 在无测量证据时引入 uv/缓存实现（出口保留，凭数据进入）。

## 6. 冻结与解冻

本文与既有冻结文档同级。解冻沿用停止规则：现实问题 + 稳定变化点 +
至少两个真实消费者，经 ADR 修改对应章节。§2 各契约面的 additive 扩展
（新增可选字段、新增 registry 条目）不属于解冻，按各自验证半径执行。
