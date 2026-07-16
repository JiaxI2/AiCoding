# Visio MCP 一等控制面可追溯性

| 需求 / 决策 | 实现 | 测试 / 门禁 |
|---|---|---|
| 一等 MCP registry | `config/mcp-registry.json`、component manifests/schema | registry/schema 单测、Kit structure |
| `aicoding mcp` Go 控制面 | `internal/mcpcontrol`、CLI 路由 | CLI JSON tests、Go tests |
| 当前 MCP 兼容性 | Codex config discovery、stdio/HTTP probe | initialize、tools/resources/prompts list |
| 下层不能观察上层 | generic `visio-mcp` package/module/env/schema/example/test | dependency governance、namespace scan |
| MCP/Skill 职责分离 | MCP tools/resource；`visio-diagram` Skill 编排工作流 | prompt count = 0、Skill validation |
| 稳定身份版本不可见 | manifest metadata/文档/CHANGELOG 承载版本 | dependency governance、README badge authority |
| Visio MCP capability | `CodingKit/tools/windows-automation/visio-mcp` | Python tests、benchmark、COM doctor |
| 框图尺寸、同轴一致性与收敛 | `sizeClass`、内容/端口尺寸包络、compact layout | under/oversize、axis size、page utilization、same-axis gap tests |
| 连接端点、多端口车道与箭头净空 | side ports、absolute lane、calibrated arrow geometry | endpoint intrusion、paired lane、terminal clearance、COM path inspection |
| 文字不覆盖线或框 | planned/live label bbox collision gate | text-line、text-shape、text-text regression |
| 框内文字与尺寸族 | explicit text transform、`sizeClass`、content ratio | text center error、size class、84% safe area |
| 连接语义完整 | 两端 GlueToPos、live Connects count | `fullyGluedRatio = 1.0` |
| 协议合规 | 官方 Python MCP SDK、annotations、tool errors | 官方 client regression、invalid JSON recovery |
| 安全安装与卸载 | 时间戳备份、托管配置块、精确 `.venv` ownership | lifecycle tests、dry-run、rollback |
| 文档同步 | README、COMMANDS、架构/运维、CHANGELOG | DocSync、Markdown links |
