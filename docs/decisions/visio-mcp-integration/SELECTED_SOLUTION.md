# Visio MCP 集成已选方案

Decision Status: Selected

User selected: Option A，一等 MCP component registry + `aicoding mcp` Go 控制面。

## 所有权

- AiCoding 接管 `CodingKit/tools/windows-automation/visio-mcp` 的仓库托管，但该 capability 的稳定身份保持通用，只能被上层 registry/Go 控制面观察，不能反向观察 AiCoding。
- 不修改 `CodingKit/agents/skills`、插件生成物或 Codex plugin cache。
- 不把 MCP 伪装成 Skill，也不复制 Skill 源码；通用 `visio-diagram` Skill 的权威源属于 Codex-Skills。
- MCP 只提供 tools 和 Diagram IR resource，不维护 prompt 目录，不注册 workflow prompt。

## 控制面

- MCP registry：`config/mcp-registry.json`。
- Component manifests：`config/mcp/components/*.json`。
- Schema：`config/schemas/mcp-registry.schema.json` 和 `config/schemas/mcp-component.schema.json`。
- Go package：`internal/mcpcontrol`。
- CLI：`aicoding mcp list|status|doctor|verify|install|update|uninstall`。
- Skill 工作流：`validate -> plan -> render/open -> snapshot/inspect -> quality -> bounded repair -> export -> close`。

## 运行时与安全

- Go 原生探测 stdio 和 Streamable HTTP MCP，只执行 initialize、能力协商和 list/read-only discovery。
- 不调用当前 MCP 的写工具，不输出或持久化 bearer token、headers 或 env secret。
- Visio MCP 安装使用组件目录内隔离 `.venv`。
- Codex 配置写入前创建时间戳备份。
- 仅修改带 AiCoding BEGIN/END 标记的托管配置块；同名非托管配置存在时拒绝覆盖。
- uninstall 仅删除精确匹配的托管配置块、组件 `.venv` 和安装状态。
- Diagram IR 默认统一节点尺寸，并验证同层中心、主行/主列和层间距对齐。
- Connector 使用确定性四侧端口；多端口块通过归一化 position 形成语义车道。
- 真实检查读取端点、Glue 数量、route style、path points 和文字 bbox；文字覆盖线、
  框或其他文字均阻断导出。
- 节点文字通过显式 Text Transform 保持水平/垂直居中；同角色框体通过
  `sizeClass` 统一尺寸，每个维度由文字、真实端口密度或容器成员包围盒独立约束。
- 紧凑模式约束页面利用率和同轴间距；connector 首末段必须位于节点外部，固定箭头
  几何的终端净空和节点碰撞属于 Release 阻断项。

## 兼容性

- MCP protocol target：`2025-11-25`。
- 当前配置中的 MCP 在集成前已全部通过只读 initialize/discovery 基线。
- Visio MCP 使用官方 Python MCP SDK 稳定 v1，并固定 `<2` 上限。
- 兼容性 probe 可以执行 `prompts/list`，但受管 Visio MCP 的预期 prompt 数量为零。

## 稳定身份

- component ID、目录、package、module、service、schema、环境变量、示例、测试和代码不编码资产版本。
- 版本仅由 manifest metadata、资产文档、CHANGELOG 或 Tag/Release 权威信息承载。

## 回滚边界

- 删除新增 MCP registry、schema、Go package、CLI 路由和通用 capability 资产。
- 从 `config.toml` 移除精确的 AiCoding 托管块，或恢复安装前备份。
- 删除组件目录内 `.venv` 和 `.aicoding/state/mcp/visio-mcp`。
