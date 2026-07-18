# MCP Components

本页说明 AiCoding 上层 MCP registry 与 Go 生命周期控制面的日常操作。MCP capability 本身保持通用；平台绑定只存在于 registry、component manifest、Go 控制面和受管安装状态中。

## 前置条件

- Windows 上使用 PowerShell 7 或更高版本；
- component manifest 指定的 Python 最低版本可用；
- `visio-mcp` 与 `ppt-mcp` 默认可通过 `python.exe` 或本机 Python 安装目录发现解释器，不要求存在 `py.exe`；
- 需要指定解释器时分别使用 component manifest 声明的 `VISIO_MCP_PYTHON` 或 `PPT_MCP_PYTHON`；
- 两个组件的 Smoke 和 Full 都不启动可见 Office COM；
- Release 需要对应的 Windows 桌面版 Microsoft Visio 或 PowerPoint，并会显式打开可见 COM 会话。

## Inventory 与状态

```powershell
bin\aicoding.exe mcp list --json
bin\aicoding.exe mcp status visio-mcp --json
bin\aicoding.exe mcp doctor visio-mcp --json
bin\aicoding.exe mcp status ppt-mcp --json
bin\aicoding.exe mcp doctor ppt-mcp --json
```

- `list` 合并显示 registry 中的受管 components 与 Codex 当前配置中的 MCP；
- `registryDigest` 只标识规范化 registry，`catalogDigest`/外层 `inputDigest` 标识 registry
  与全部 referenced component manifests；
- `status` 检查 component root、`.venv`、安装状态、受管配置块和同名非受管冲突；
- `doctor` 在 component 隔离环境中运行 manifest 声明的诊断命令。

需要检查其他 Codex 配置文件时使用 `--codex-config <PATH>`。

## 安装、更新与卸载

先执行 dry-run：

```powershell
bin\aicoding.exe lifecycle plan --action install --scope mcp --component visio-mcp --json
bin\aicoding.exe lifecycle plan --action update --scope mcp --component visio-mcp --json
bin\aicoding.exe lifecycle plan --action uninstall --scope mcp --component visio-mcp --json
bin\aicoding.exe lifecycle plan --action install --scope mcp --component ppt-mcp --json
```

确认后执行：

```powershell
bin\aicoding.exe lifecycle install --scope mcp --component visio-mcp --json
bin\aicoding.exe lifecycle update --scope mcp --component visio-mcp --json
bin\aicoding.exe lifecycle uninstall --scope mcp --component visio-mcp --json
bin\aicoding.exe lifecycle install --scope mcp --component ppt-mcp --json
```

也可以使用 `--scope mcp --all`，处理 registry 中全部启用的 components。旧
`mcp install|update|uninstall` 已移除，写操作统一由 lifecycle 承载。

生命周期写操作遵守以下所有权规则：

- Codex 配置写入前创建 `config.toml.bak-<timestamp>`；
- component `.venv` 同时安装 requirements 与 manifest 声明的组件 package，保证 fresh install 可直接启动 module；
- 只维护 `# BEGIN AICODING MCP <server>` 与对应 END 标记之间的上层受管块；
- component 环境变量来自 manifest，并将 `${componentRoot}` 展开为实际路径；
- 同名非受管 `[mcp_servers.<name>]` 存在时拒绝覆盖；
- uninstall 先在 component root 内暂存精确命名的 `.venv`；活跃进程锁定时会在改写配置前失败；
- 暂存成功后才删除受管配置、`.venv` 和安装状态，不删除未知目录或用户 MCP。

## 验证

受管 component：

```powershell
bin\aicoding.exe mcp verify visio-mcp --profile Smoke --json
bin\aicoding.exe mcp verify visio-mcp --profile Full --json
bin\aicoding.exe mcp verify visio-mcp --profile Release --json
bin\aicoding.exe mcp verify ppt-mcp --profile Smoke --json
bin\aicoding.exe mcp verify ppt-mcp --profile Full --json
bin\aicoding.exe mcp verify ppt-mcp --profile Release --json
```

当前 Codex MCP 兼容性：

```powershell
bin\aicoding.exe mcp verify visio-mcp --profile Smoke --configured --json
bin\aicoding.exe mcp verify --all --profile Smoke --json
```

`--configured` 或 `--all` 会对当前配置的 stdio/Streamable HTTP servers 执行只读 initialize/discovery。报告包含 tools、resources 和 prompts 的发现数量，但不调用业务 tool。

### Profile 选择

- Visio Smoke：覆盖协议、Diagram IR、mock renderer、统一尺寸和对齐质量，不触发真实 Visio；
- Visio Full：在 Smoke 基础上增加 benchmark；
- Visio Release：在 Full 基础上运行可见 Visio COM smoke，并验证 VSDX、PNG、SVG、PDF、quality 和 inspection 输出；
- PowerPoint Smoke：运行 vendored 上游 pytest 回归，使用 mock，不启动可见 PowerPoint；
- PowerPoint Full：当前无独立 benchmark，执行与 Smoke 相同的完整 pytest 集；
- PowerPoint Release：在 pytest 后显式运行可见 PowerPoint COM round-trip，生成并验证 PPTX。

Release 属于显式慢路径。执行后检查没有遗留 `VISIO.EXE` 或 `POWERPNT.EXE`。

## 框图对齐验收

输入 Diagram IR 建议显式保留：

```json
{
  "document": {
    "layout": {
      "uniformNodeSize": true,
      "nodeWidth": 2.4,
      "nodeHeight": 1.0,
      "compact": true
    }
  },
  "edges": [
    {
      "id": "signal",
      "from": "source",
      "to": "target",
      "sourcePort": "right",
      "sourcePortPosition": 0.5,
      "targetPort": "left",
      "targetPortPosition": 0.5,
      "routing": "orthogonal"
    }
  ],
  "nodes": [
    {
      "id": "source",
      "text": "Source",
      "sizeClass": "process"
    }
  ]
}
```

`sourcePortPosition`/`targetPortPosition` 是 `0..1` 的归一化侧边位置，默认
`0.5` 为侧边中心。多端口工程块先确定共同的绝对页面 X/Y，再根据每个框体的
实际宽高反算各自归一化位置；只有框体尺寸相同时，相同百分比才天然代表同一车道。

验收至少包括：

- 矩形框宽高一致；
- 同层中心对齐；
- 主行/主列对齐；
- 层间距一致；
- connector 端点位于选定侧边中心或端口车道，误差不超过 `0.03 in`；
- connector 双端 glue，`fullyGluedRatio = 1.0`；
- 节点文字块、段落和垂直对齐均居中，中心误差不超过 `0.02 in`；
- 同一 `sizeClass` 的框体宽高一致；同轴节点在共同尺寸不会造成过大时共享该尺寸；
- 每个维度分别由文字安全区、实际端口密度或容器成员包围盒确定，普通文字内容不超过框体宽高的 84%；
- 紧凑模式下页面横向/纵向利用率至少为 75%/65%，同轴相邻框边界间距不超过 `max(0.8 in, 1.5 * nodeGap)`；
- 正交/直线路由样式与 IR 一致，且不穿过无关框体；
- connector 首段向源框外部离开、末段从目标框外部进入；箭头终端至少保留 `0.18 in`，箭头包围盒不得覆盖节点；
- `SOURCE_ENDPOINT_INTRUSION`、`TARGET_ENDPOINT_INTRUSION`、
  `ARROW_TERMINAL_CLEARANCE_LOW`、`ARROWHEAD_OVERLAPS_NODE` 和
  `ARROW_GEOMETRY_UNVERIFIED` 为零；
- `TEXT_LINE_OVERLAP`、`TEXT_LINE_CLEARANCE_LOW`、`TEXT_SHAPE_OVERLAP`、
  `TEXT_TEXT_OVERLAP` 为零，线到文字 bbox 的净距至少 `0.08 in`；
- `INCONSISTENT_NODE_SIZE`、`LAYER_MISALIGNED`、`ORDER_MISALIGNED` 和层间距类 finding 为零；
- 导出的 PNG/PDF 与可编辑 VSDX 视觉一致。

连接线文字不能使用 Visio 默认的“在线中心”位置。MCP 会设置独立文字坐标并使用
不透明背景作为视觉兜底，但最终是否合格由实际文字 bbox 与实际 connector path
points 的坐标相交检查决定。

质量 repair 的次数和停止条件由上层 `visio-diagram` Skill 控制；MCP 不通过 prompt 隐式决定工作流。

## 常见故障

### 找不到 Python

确认目标解释器满足 manifest 的最低版本，然后设置：

```powershell
$env:VISIO_MCP_PYTHON = 'C:\path\to\python.exe'
bin\aicoding.exe mcp doctor visio-mcp --json
$env:PPT_MCP_PYTHON = 'C:\path\to\python.exe'
bin\aicoding.exe mcp doctor ppt-mcp --json
```

`py -3.11` 不是前置条件；控制面直接验证解释器路径和实际 major/minor。

### 同名非受管配置

不要让控制面覆盖用户配置。先检查 `mcp list`/`mcp status` 输出，由用户决定重命名、迁移或手工保留现有 server。

收养 `ppt-mcp` 时，历史手动 `[mcp_servers.ppt-mcp]` 必须在用户确认并完成配置备份后移除；控制面随后写入自己的受管块。旧的手动 clone 不属于受管卸载范围，只有用户另行确认后才能删除。

### 卸载提示 `.venv` 被占用

先关闭或重启正在使用该 component 的 MCP host，再重试 uninstall。控制面不会终止未知 Python
进程；`.venv` 无法暂存时，Codex 受管配置和安装状态保持不变。

### Release 无法启动 Visio

先运行 Smoke 和 Full 排除协议、依赖和布局问题，再确认桌面版 Visio COM 可用。Release 失败后关闭已知测试会话，并检查是否存在孤立 `VISIO.EXE`；不要终止用户正在编辑的未知会话。

### Release 无法启动 PowerPoint

先运行 PowerPoint Smoke 和 Full 排除依赖及 mock 回归问题，再确认桌面版 PowerPoint COM 已注册。Release smoke 要求事先关闭现有 PowerPoint 会话，避免附着、修改或关闭用户正在编辑的演示文稿。

## 新增 Component

1. 使用领域名称创建通用 capability，禁止把 AiCoding 名称或版本写入稳定身份；
2. 在 `config/mcp/components/` 增加 manifest，并登记到 `config/mcp-registry.json`；
3. 定义 doctor、Smoke、Full、Release、安全限制和允许输出根；
4. 对 stdio 或 Streamable HTTP lifecycle 增加只读 compatibility test；
5. 确认 capability 只暴露 tools/领域 resources，不注册工作流 prompt；
6. 更新架构、命令、运维和 changelog；
7. 运行：

```powershell
bin\aicoding.exe governance dependencies --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe mcp verify --all --profile Smoke --json
git diff --check
```

架构边界见 [MCP Control Plane](../architecture/MCP_CONTROL_PLANE.md)。
