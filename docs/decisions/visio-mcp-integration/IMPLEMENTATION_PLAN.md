# Visio MCP 一等控制面实现计划

Plan Status: Approved

## Phase 1：通用 capability 资产

1. 将必要源码、schema、示例、测试、脚本和文档纳入 `CodingKit/tools/windows-automation/visio-mcp`。
2. 排除 `.venv`、cache、dist、测试输出和过时的独立 Go client 草案。
3. 使用官方 Python MCP SDK 替换手写协议循环。
4. 修复 Python 解释器探测、tool annotations、logging 和错误恢复，并删除 capability 对 workflow prompt 的所有权。
5. 使用通用 package/module/environment/schema/example/test 身份，禁止反向观察 AiCoding 或把版本编码进稳定身份。

## Phase 2：Registry 与 Go 控制面

1. 新增 MCP registry、component schema 和 Visio component manifest。
2. 新增 `internal/mcpcontrol`：
   - registry/config discovery；
   - Codex TOML MCP discovery；
   - stdio/Streamable HTTP compatibility probe；
   - Python venv lifecycle；
   - Codex 托管配置块与备份；
   - status/doctor/verify reports。
3. 在 CLI 中增加 `mcp` 路由，不重构无关命令。

## Phase 3：验证与文档

1. Go 单测覆盖 registry、配置保护、stdio/HTTP probe 和 CLI JSON。
2. Python 测试覆盖官方 MCP client lifecycle、零 workflow prompts、invalid JSON recovery、annotations、统一尺寸和对齐质量。
3. 更新 `docs/COMMANDS.md`、MCP 架构/运维文档、三份 README 的工具链入口和 `CHANGELOG.md`。
4. 运行 Go、Kit、DocSync、Markdown、governance、Plan Mode 和 Git diff 门禁。

## 停止条件

- 不覆盖用户现有未提交 CLI、governance 或 issue lifecycle 改动。
- 同名非托管 Codex MCP 配置存在时停止安装。
- 当前 MCP 只读回归出现退化时停止交接。
- 真实 Visio Release 验证留下孤立 `VISIO.EXE` 时停止交接。

## 回滚

所有新增文件按本任务文件列表删除；对现有文件仅反向撤销本任务锚点。Codex 用户配置优先恢复自动备份。
