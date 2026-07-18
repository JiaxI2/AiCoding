# 契约冻结与获取/激活边界

Status: Accepted and Frozen

## 1. 结论

本文完成发布后的两项收敛：

1. 将四个已被多方消费、语义稳定的契约面正式冻结（§2）；
2. 定义外部来源（GitHub/URL 起源的 Skill、MCP、Kit）的获取/激活分离边界与
   标准接入流程，使"慢"被隔离在唯一的获取步骤中，激活保持本地快速（§3）。

原则与 Git 相同：Git 快不是因为网络快，而是因为几乎所有命令都是纯本地操作，
网络被隔离在 `fetch/push` 两个显式动作里。本仓库将同一结构固化为契约。

## 2. 四项冻结声明

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
（pin / fork-pin / adopt，见 `aicoding-external-integration` Skill）正交组合：

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

## 4. 可执行门禁（两条）

1. **激活面 URL-free**（governance dependencies 新 check）：
   `config/kit-registry.json`、`config/kits/*.json`、`config/mcp-registry.json`、
   `config/mcp/components/*.json`、`config/codex-kit.json` 的任何字符串值不得
   含 `://`。激活所需的一切必须以仓库相对路径表达。
2. **可克隆源只在获取登记面**（governance dependencies 新 check）：
   在 `config/**` 与 `.gitmodules` 范围内，可克隆 git 源 URL（`*.git` 结尾或
   仓库根形态的 github/gitcode 地址）只允许出现在 `.gitmodules` 与
   `config/skill-sources.json`。文档链接、badge 权威 URL、schema `$id` 不属于
   可克隆源，不在本检查范围。

两条检查在当前 main 上应当零违规通过（冻结的是既成现实，不是新行为）。

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
