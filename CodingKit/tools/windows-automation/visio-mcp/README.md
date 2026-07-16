# Visio MCP

通用 Microsoft Visio MCP 组件，提供配置驱动的图生成、可视化编辑、质量检测和有限自动修复。

## 边界

- Diagram IR 是唯一图模型。
- Layered/grid 默认统一节点外框尺寸，并保持主行/主列、同层中心和层间距一致。
- Connector 默认使用正交路由，支持四侧中心与归一化多端口车道。
- 真实检查读取连接端点、Glue 数量、ShapeRouteStyle、实际 path points 和文字块坐标。
- 文字与连接线、框线或其他文字相交属于阻断错误。
- 样式由精简 JSON profile 控制；默认 `engineering-standard` 恢复早期输出的宋体 `10 pt`、黑白配色、`0.75 pt` 线和 `0.12 in` 小圆角。
- profile 只暴露普通/亚洲/公式字体、默认字号、文字安全区、线宽和圆角七个高影响参数，不自动改写整张图的角色层级。
- 需要复刻 `id=0` 一类标版时，才对具体节点/信号/caption 显式使用 `18/16/14/12/10 pt`；页面大小不会暗中改写字号或线宽。
- 默认黑线、黑字、白底。蓝/红/绿只作为稳定语义 style token，不用于装饰。
- 节点文字块默认水平、垂直和段落居中，并使用框体宽高的 80% 作为文本安全区；菱形默认 70%。
- 节点可用 `captionSide`、`captionPosition` 和 `captionOffset` 将外部模块标题绑定到框体上下或左右中心。
- 连接线文字使用 `labelPosition` 锚定到相对中部，并限制切向/法向避让，无法在预算内放置时阻断而不是无限漂移。
- `sizeClass` 约束同角色框体宽高。
- 普通文字和字形测量都不得超出框体宽高的 80% 安全区；宽高分别由已解析字号的文字、实际端口密度或容器成员包围盒确定。
- 紧凑模式检查页面利用率、同轴间距、总连接长度和折点，避免框图无语义地发散。
- 同一主轴、同一 `sizeClass` 的连续节点使用页面绝对边界计算框间距，整组间距差不得超过 `0.03 in`。
- 箭头样式、尺寸和线宽固定；首末段必须从节点外部进出，箭头包围盒不得覆盖任何节点。
- PNG 快照与导出保留 4% 到 8% 的白色外边距，消除 Visio 导出过滤器裁到内容边界造成的贴边结果。
- MCP 使用本地 stdio transport。
- 真实渲染仅支持 Windows 桌面版 Microsoft Visio。
- COM 操作是显式副作用能力，不进入默认仓库 Smoke。
- CI 使用 mock renderer，不要求安装 Visio。
- Server 不执行任意 shell、不加载宏，并限制输出目录。

## 独立开发

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\install.ps1
.venv\Scripts\python.exe -m visio_mcp --renderer mock doctor --json
.venv\Scripts\python.exe -m pytest -q
```

## MCP 能力

- Tools：校验、对齐规划、渲染、打开、快照、检查、编辑、质量检测、修复、导出和关闭。
- Resources：Diagram IR schema、style-profile schema、当前 style profiles 与 renderer-effective 字段清单。
- Prompts：无。工作流编排属于上层 Skill，不属于 MCP capability。

真实 Visio Release 验证：

```powershell
.venv\Scripts\python.exe tools\visio_smoke.py --visible
.venv\Scripts\python.exe tools\visio_smoke.py --input examples\generic-dual-loop-actuator-control.json
```

## Node 与 Connector IR

```json
{
  "document": {
    "styleProfile": "engineering-standard",
    "typography": {
      "fontSizePt": 10,
      "mathFontSizePt": 12
    },
    "appearance": {
      "lineWeightPt": 0.75,
      "nodeCornerRadiusIn": 0.12
    }
  },
  "node": {
    "id": "controller",
    "text": "Current\nController",
    "caption": "Current loop",
    "captionSide": "top",
    "captionPosition": 0.5,
    "captionOffset": 0.1,
    "sizeClass": "multiport-stage",
    "fontRole": "body",
    "textBlockWidthRatio": 0.8,
    "textBlockHeightRatio": 0.8,
    "width": 1.25,
    "height": 1.15
  },
  "edge": {
    "sourcePort": "right",
    "sourcePortPosition": 0.75,
    "targetPort": "left",
    "targetPortPosition": 0.75,
    "routing": "orthogonal",
    "labelSide": "above",
    "labelOffset": 0.22,
    "labelPosition": 0.5,
    "fontRole": "signal",
    "style": "feedback"
  }
}
```

端口位置默认 `0.5`，表示准确侧边中心。left/right 的位置从下到上，top/bottom
的位置从左到右。质量门禁要求 `fullyGluedRatio = 1.0`，端点误差不超过
`0.03 in`，并且所有 text/line、text/shape、text/text collision 计数为零。
线到文字包围框的净距必须至少为 `0.08 in`。
节点实测要求 `nodeTextMisalignedCount = 0`、文字块中心误差不超过 `0.02 in`；
文字块宽高比例误差不超过 `0.02`，Latin/Asian 字体必须与 IR 请求一致；
节点、caption 和连接线标签字号误差不超过 `0.25 pt`，同一
`fontRole + sizeClass` 的字号跨度不超过 `0.25 pt`；线宽误差不超过 `0.10 pt`，
圆角误差不超过 `0.005 in`，字体样式、文字/线/填充颜色 mismatch 计数必须为零；
外部标题与连接线文字锚点误差分别不得超过 `0.02 in`/`0.03 in`。
同一 `sizeClass` 的宽高差不得超过 `0.05 in`。紧凑模式要求页面横向/纵向内容
利用率至少为 75%/65%，同轴连接框边界间距不超过
`max(0.8 in, 1.5 * nodeGap)`。`sourceEndpointIntrusionCount`、
`targetEndpointIntrusionCount`、`arrowTerminalClearanceLowCount`、
`arrowheadNodeOverlapCount` 和 `arrowGeometryUnverifiedCount` 必须全部为零。

内置 profile 位于 `styles/style-profiles.json`。需要快速切换或维护用户配置时，
可复制该文件并设置 `VISIO_MCP_STYLE_PROFILES` 指向新 JSON；字段合同由
`schemas/style-profile.schema.json` 约束。默认配置仅包含 `font`、`text` 和
`line` 三组七个值；修改后重启 MCP 进程。
