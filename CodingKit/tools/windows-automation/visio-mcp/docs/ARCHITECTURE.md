# Visio MCP Architecture

```text
User / agent / diagram workflow
        |
        v
MCP client or host
  lifecycle / discovery / approval
        |
        | MCP stdio, JSON-RPC 2.0
        v
Visio MCP Server
  tools + schema/style resources
        |
        +--> Diagram IR schema
        +--> style-profile schema + active JSON profiles
        +--> renderer-effective field registry
        +--> deterministic compact style + layout
        +--> row / column / layer alignment
        +--> text / connector / arrow quality
        +--> bounded repair
        |
        v
Renderer interface
  +--> Mock renderer (CI)
  +--> Visio COM renderer (Windows)
        |
        v
Visible Visio document
  inspect / edit / snapshot / export
```

## 状态和副作用

- `validate`、`plan`、`inspect`、`quality_check`：只读。
- `render`、`open_visible`：创建会话和文件。
- `edit`、`auto_repair(apply=true)`：修改活动文档。
- `snapshot`、`export`：写输出文件。
- `close`：保存并关闭 COM 会话。

COM 会话只在单个 MCP Server 进程内有效，不跨进程恢复。

MCP 不拥有或注册绘图工作流 Prompt，也不依赖任何上层产品、Skill 名称、registry 或安装状态。

## 样式与质量合同

- `document.styleProfile` 选择平台无关 JSON 样式；`document.typography`、
  `document.appearance` 和节点/连接线字段按层覆盖。
- 默认 profile 只控制字体组、默认字号、文字安全区、共享线宽和圆角，不做隐式页面缩放。
- 默认视觉基线为宋体 `10 pt`、`0.75 pt` 黑线、白底和 `0.12 in` 小圆角；
  标版层级由节点、caption 或连接线的显式字号表达。
- planner、mock renderer、COM renderer 和 live inspection 使用同一套已解析字体、
  字号、颜色、线宽、圆角、80% 文字安全区、caption/label 锚点和绝对同轴间距合同。
- MCP 只报告 renderer-effective 字段与可测量质量结果；视觉迭代和 no-op 工作流归上层 Skill。
