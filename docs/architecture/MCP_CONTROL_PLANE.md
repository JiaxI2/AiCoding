# MCP Control Plane

AiCoding 的 MCP 控制面属于上层 platform/integration。它负责组件登记、安装状态、Codex 配置、兼容性回归和生命周期；通用 MCP capability 只提供领域工具与资源，不观察 AiCoding。

## 依赖方向

```text
AiCoding platform
-> MCP registry / Go control plane
-> generic MCP capability
-> Python / COM / operating-system runtime
```

依赖只允许由高层指向同层或低层：

- `config/mcp-registry.json` 和 `config/mcp/components/*.json` 可以绑定 `visio-mcp`；
- `internal/mcpcontrol` 可以读取组件 manifest、管理运行环境并探测 Codex 中已配置的 MCP；
- `visio-mcp` 的 package、module、service、schema、环境变量、示例和测试保持领域命名，不包含 `aicoding-*`、`AICODING_*` 或其他上层产品身份；
- 上层 Skill 可以编排 MCP tools，但 MCP 不注册工作流 prompt，也不依赖 Skill 名称。

可执行边界由 `config/dependency-governance.json` 定义，并通过以下命令验证：

```powershell
bin\aicoding.exe governance dependencies --json
```

## 组成

| 层 | 权威入口 | 职责 |
|---|---|---|
| Registry | `config/mcp-registry.json` | 登记启用的 MCP component 和 manifest 路径 |
| Component manifest | `config/mcp/components/*.json` | 声明运行时、Codex server、doctor、验证 profile、安全边界和输出 |
| Schema | `config/schemas/mcp-*.schema.json` | 约束 registry 与 component manifest |
| Go package | `internal/mcpcontrol` | inventory、status、doctor、lifecycle 和 protocol probe |
| CLI | `bin\aicoding.exe mcp ...` | 对用户暴露稳定的 Go 控制面 |
| Capability | `CodingKit/tools/windows-automation/visio-mcp` | 提供通用 Visio tools、Diagram IR schema 和渲染实现 |

## 控制面与工作流的分离

`aicoding mcp` 不负责“如何画图”。它只负责确保 MCP component 可安装、可发现、可诊断、可验证和可回滚。

具体画图过程属于上层通用 `visio-diagram` Skill：

```text
validate -> plan -> render/open -> snapshot/inspect
-> quality check -> bounded repair -> export -> close
```

该 Skill 调用 `diagram_*` tools，并对视觉结果进行人工确认。`visio-mcp` 只暴露 tools 和 Diagram IR schema，不维护 prompt 目录，不注册工作流 prompt。

通用 capability 的质量事实包括节点中心与内容/端口所需尺寸、四侧/多端口 endpoint、
双端 glue、正交/直线路由、实际 connector path points、箭头样式与净空、紧凑度，
以及文字 bbox 与线段/框体/其他文字的碰撞结果。具体选择工程图、泳道图、流程图或
状态机规则仍由上层 Skill 按需加载。

`visio-diagram` 的权威源码属于 Codex-Skills。AiCoding 只能在该 Skill 已提交、发布并由
`CodingKit/agents/skills` 锁定到已批准 commit 后增加运行时映射；不得把未提交的相邻源码仓库
直接登记到 `config/codex-kit.json`，也不得复制 Skill 源码。

## 生命周期

Go 控制面按 component manifest 执行以下流程：

1. 将规范化 registry 与全部 referenced component manifest digest 组合为 `mcp-catalog`，
   选择、plan/apply/status/doctor/verify 共用同一批 detached component values；
2. 检查 component root、Python 下限和可选的通用运行时覆盖变量；
3. 在 component root 内维护隔离 `.venv`，先安装 requirements，再按 `packageInstall` 安装组件自身；
4. 将 manifest 声明的 command、args、cwd、timeout 和环境变量写入 Codex MCP 配置；
5. 修改配置前创建时间戳备份；
6. 写入 `.aicoding/state/mcp/<component-id>/install-state.json`；
7. update 只更新受管块；uninstall 先将 `.venv` 原子暂存为同目录临时名，再移除受管配置和安装状态。

`mcp list --json` 同时返回只覆盖 registry 的 `registryDigest` 与覆盖 registry + manifests 的
`catalogDigest`；正式 lifecycle 另返回 adapter catalog、domain input 与 plan digest。

同名但没有受管标记的 Codex MCP 配置视为用户所有，install、update 和 uninstall 都拒绝覆盖或删除。
若活跃 MCP 进程锁定 `.venv`，暂存步骤会在修改 Codex 配置前失败，避免出现半卸载状态。

## 兼容性回归

`mcp verify --configured` 对 Codex 当前配置执行无副作用 probe：

- 支持 stdio 和 Streamable HTTP；
- 执行 initialize、能力协商以及 `tools/list`、`resources/list`、`prompts/list`；
- 不调用业务 tool，不执行写操作；
- 不把 bearer token、headers 或环境变量 secret 写入报告；
- `--all` 在验证所有已启用受管 component 的同时自动包含当前配置的 MCP probe。

`prompts/list` 只用于验证第三方 MCP 的协议兼容性，不表示 Visio capability 应拥有工作流 prompt。受管 `visio-mcp` 的预期 prompt 数量为零。

## 验证 Profile

| Profile | Visio component 范围 | 副作用边界 |
|---|---|---|
| Smoke | Python tests、mock renderer、协议与布局质量回归 | 不要求 Visio，不启动可见 COM |
| Full | Smoke 加 mock benchmark | 不要求 Visio，不启动可见 COM |
| Release | Full 加真实可见 Visio COM smoke 与 VSDX/PNG/SVG/PDF 导出 | 仅 Windows 桌面环境，显式执行 |

Release 必须确认导出文件存在、质量/检查结果有效，并且没有遗留孤立 `VISIO.EXE`。

## 框图质量契约

Diagram IR 的布局按图类型保持框图整洁：

- 简单同质图可使用 `uniformNodeSize=true`；
- 复杂图使用 `sizeClass`，并按文字安全区、实际端口密度和容器成员包围盒分别约束宽高；
- 同轴节点在共同尺寸不会造成过大时共享该尺寸；
- 同层节点中心对齐；
- 相同顺序节点保持主行或主列对齐；
- 相邻层间距保持一致；
- 紧凑模式检查页面利用率和同轴边界间距；
- 质量检查阻断端点向内、箭头覆盖节点、尺寸无依据放大和文字/几何碰撞。

自动 repair 只能作为上层 Skill 控制的有限循环。最终 VSDX、PNG、SVG 和 PDF 仍需快照检查与人工视觉确认。

## 稳定身份与版本

component ID、目录、package、module、service、schema ID、环境变量、示例、测试和运行时代码都不得编码版本。版本只允许出现在：

- component/asset manifest metadata；
- 资产说明文档；
- `CHANGELOG.md`；
- Tag/Release 权威 URL；
- 三份根 README 顶部、指向准确权威的相同版本 badge。

相关运维命令见 [MCP Components](../operations/MCP_COMPONENTS.md) 和 [Commands](../COMMANDS.md)。
