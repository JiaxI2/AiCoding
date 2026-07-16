# Visio MCP 集成方案记录

Decision Status: Resolved - Option A Selected

## 目标

将待集成的 Visio MCP Kit 源仓库验证并纳入 AiCoding，同时对当前 Codex 配置中的 MCP 工具建立无副作用兼容性回归。

## 已确认基线

- 源目录不是 Git 仓库，不能直接作为可追踪 submodule 接入。
- `KIT_MANIFEST.json` 的 41 个文件哈希全部一致。
- Kit 自带 9 个测试全部通过，30 次 mock benchmark 通过。
- Microsoft Visio COM 16.0 doctor 通过。
- 当前 7 个 MCP 均通过 `2025-11-25` initialize 与只读发现回归。
- 集成前必须修复安装器、下层 workflow prompt 所有权、logging、错误恢复、错误分类和 tool annotations 问题。

## Option A：一等 MCP 控制面（推荐）

将 Visio MCP 作为 AiCoding 首方 CodingKit 工具资产，并新增统一 MCP component registry 与 Go CLI：

```text
CodingKit/tools/windows-automation/visio-mcp/
config/mcp/components/*.json
config/schemas/mcp-component.schema.json
internal/mcpcontrol/
aicoding mcp list|status|doctor|verify|install|update|uninstall
```

范围：

- 复制源码时排除 `.venv`、cache、dist 和测试输出；AiCoding 接管后续源码所有权。
- Visio 的安装、doctor、Smoke/Full/Release、回滚与卸载通过 Go 控制面编排。
- 现有 MCP 回归只执行 initialize、能力协商和 list/read-only discovery，不调用写工具，不持久化凭据。
- 修复已发现的协议和安装缺陷，并增加官方 MCP SDK compatibility tests。

优点：满足当前和未来 MCP 的统一治理、回归和可观测性要求。

代价：新增 registry、schema、Go package 和 CLI，改动面最大。

## Option B：复用现有 Kit Registry 的最小集成

将源码纳入 `CodingKit/tools/windows-automation/visio-mcp/`，增加 `config/kits/visio-mcp.json`，复用现有 `kit` 与 `lifecycle` 命令。

范围：

- 修复 Visio Kit 自身问题并增加 compatibility tests。
- 通过现有 kit manifest 管理 install/status/verify/export。
- 当前 7 个 MCP 的回归保留为测试资产，不新增 `aicoding mcp` 命令族。

优点：改动较小，复用现有 schema 和生命周期引擎。

代价：MCP 被作为通用 Kit 管理，缺少面向 MCP transport、capability 和 tool annotations 的一等模型；现有 MCP 的持续回归入口不够直接。

## Option C：保持外部源码，仅做路径/包集成

不复制源码，通过环境变量、Git submodule 或已发布 Python 包定位外部 Kit。

优点：AiCoding 不接管 Visio MCP 源码。

代价：当前目录没有 Git 元数据、远程 URL 或已发布包，无法形成可复现的新机器安装与更新链；在补齐外部发布源之前不能完成集成。

## 推荐

选择 Option A。用户明确要求当前 MCP 工具兼容性回归，一等 MCP 控制面能把 Visio 与现有 MCP 的协议验证纳入同一个稳定入口，而不是把回归逻辑藏在单个 Kit 的测试中。

## 决策结果

Option A 已选择并完成实现。当前权威实现与验收状态见 `SELECTED_SOLUTION.md`、`TASKS.md` 和 `TRACEABILITY.md`；本文件保留三个历史候选方案及其取舍。

## 所有方案共同验收条件

- 不修改 Codex plugin cache，不复制 Skill 源码，不污染 `CodingKit/agents/skills` submodule。
- 当前 7 个 MCP 的 initialize/discovery 回归继续通过。
- Visio MCP 的官方 SDK stdio 回归、mock tests、真实 Visio doctor 通过。
- 安装器不依赖 `py.exe`，并明确 Python 版本选择。
- 协议错误码、tool execution error、logging capability 和 tool annotations 符合 MCP `2025-11-25`；Visio capability 不注册 workflow prompt。
- 执行 Go tests、kit/schema gates、DocSync、Markdown links、governance lint 与 `git diff --check`。

## 回滚

回滚按 `SELECTED_SOLUTION.md` 中的资产边界移除 MCP registry/component binding、Go 控制面和受管 capability 文件，并恢复发布前的 Codex 配置备份；不得修改 plugin cache 或 Skill source。
