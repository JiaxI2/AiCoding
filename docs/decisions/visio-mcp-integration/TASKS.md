# Visio MCP 一等控制面任务

## Phase 0：决策

- [x] 用户选择 Option A。
- [x] 记录所有权、安全边界和回滚。

## Phase 1：资产与协议

- [x] 纳入平台无感的通用 Visio MCP capability。
- [x] 替换为官方 Python MCP SDK。
- [x] 修复安装器与协议缺陷，移除 workflow prompt 所有权。
- [x] 增加兼容性、错误恢复、统一尺寸和对齐质量测试。
- [x] 验证稳定身份不编码版本或 AiCoding namespace。

## Phase 2：Go 控制面

- [x] 新增 MCP registry 和 schemas。
- [x] 新增 `internal/mcpcontrol`。
- [x] 新增 `aicoding mcp` CLI 与测试。
- [x] 实现 Codex 配置备份和托管块保护。

## Phase 3：验证

- [x] 当前 MCP initialize/discovery 回归。
- [x] Visio mock/official SDK/COM doctor。
- [x] Go tests、build、Smoke/Full/Release。
- [x] VSDX/PNG/SVG/PDF 视觉对齐与无孤立 `VISIO.EXE`。
- [x] DocSync、Markdown、governance、Plan Mode、Git diff。

## Phase 4：交接

- [x] 汇总已实现、已验证、未验证和回滚。
