# 依赖方向与稳定身份治理

## 原则

AiCoding 的依赖方向固定为：

```text
platform -> integration -> capability -> runtime
```

上层可以组合或治理下层；下层不得依赖、命名、配置、记录或观察上层。分发者不等于能力所有者，由 AiCoding 集成的通用资产不能因此获得 `aicoding-*` 身份。

机器配置是 [`config/dependency-governance.json`](../../config/dependency-governance.json)，执行入口是：

```powershell
bin\aicoding.exe governance dependencies --json
```

该检查同时进入 `governance lint`、pre-commit、Smoke、CI、Full 和 Release 聚合。

## 分层

| 层 | 职责 | 示例 |
|---|---|---|
| `platform` | 用户入口、仓库策略、产品工作流 | Go CLI、仓库治理、`aicoding-*` 路由 |
| `integration` | composition root、registry、lifecycle、插件绑定 | Kit/MCP registry、安装状态、Codex 注册 |
| `capability` | 可复用能力 | 通用 Kit、standalone Skill、MCP、控制模块 |
| `runtime` | 外部协议与运行环境 | MCP 规范、Python、C99、Windows、Visio COM |

同层依赖必须显式登记；跨层依赖只允许从高 rank 指向相同或更低 rank。

## 命名与可观察性

- `aicoding-*`、`AICODING_*`、`aicoding.local` 只属于真正依赖平台语义的上层资产。
- 通用资产使用领域命名，例如 `visio-mcp`、`visio_mcp`、`VISIO_MCP_*`、`visio-diagram`。
- AiCoding 可以在 `config/` 中注册 `visio-mcp`，但 Visio MCP 源码不得出现 AiCoding registry、命令、路径或环境变量。
- Plugin Skill 使用 `aicoding-*`；standalone capability Skill 禁止使用该前缀。
- MCP capability 只提供 tools 和领域 resources；workflow prompt、绘图方法、人工检查流程归 Skill。

## Kit、Skill 与 MCP 集成

### Kit

Kit registry 中的每个条目必须在依赖治理配置中具有唯一 binding。可复用 Kit 的源码根必须通过平台无感扫描；平台专属 Kit 必须放在 `platform` 或 `integration` 层，不能伪装为通用 capability。

### Skill

Skill 权威源码仍由 Codex-Skills 管理。通用 standalone Skill 使用领域名，依赖通用 MCP；需要 AiCoding 专属路由时，另建上层 `aicoding-*` Skill 依赖通用 Skill，不得反向引用。

### MCP

MCP registry 是上层 composition root。component manifest 提供运行时参数、环境变量和安装合同；通用 Go 控制面不得硬编码 leaf MCP 的 ID、环境变量或目录。Capability MCP 禁止拥有 `prompts/` 工作流资产和 `@server.prompt` 注册。

## 版本不可观察

稳定身份不包含版本：

- 目录、文件、Kit/Skill/MCP ID；
- 包、模块、服务、CMake project/target；
- C/C++ 宏与符号；
- MATLAB/Simulink model；
- 运行时代码中的 `__version__` 或资产自版本分支。

允许的版本权威面：

- manifest/registry 的 `version` 元数据；
- 资产 README、设计文档与 `CHANGELOG.md`；
- 仓库 Tag/Release URL；
- README 顶部版本 badge。

三份 README 的 badge 必须完全一致。第三方明确版本链接官方 release/tag/版本文档；本地 Kit badge 链接本仓库 Kit 说明，并与对应 manifest 的 `version` 一致。README 正文不直接散落版本号。

Schema/protocol 的 `schemaVersion`、第三方规范编号、文件格式版本和外部依赖版本不属于资产稳定身份版本。

## 评审清单

新增或修改 Kit、Skill、MCP、模块时确认：

1. layer 和 binding 已登记；
2. 依赖边方向正确；
3. 下层根目录没有上层 namespace；
4. 稳定身份与代码不编码版本；
5. MCP/Skill 职责没有反转；
6. 版本 badge 指向准确权威 URL；
7. `governance dependencies` 与 `governance lint` 通过。
