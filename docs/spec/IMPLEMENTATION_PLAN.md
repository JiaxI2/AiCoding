# 实现计划（Implementation Plan）：product-convergence

Plan Status: Approved

## 上下文

当前 Go CLI 已是默认控制面，但测试、生命周期、报告和文档仍存在并行入口与重复聚合。
本计划只做收敛，不新增顶层产品能力。

## 已选择架构

方案 A：兼容优先的统一控制面。正式入口使用稳定命名空间，旧入口保留一个版本并
输出 `CLI_DEPRECATED`。现有 global tester 内聚为唯一 `internal/testengine`，生命周期
通过静态 adapter 组合 kit、MCP 和 runtime Skill。

## 约束

- 不新增第二套 CLI、测试框架、Runner、Report 或 UI。
- 不重写全部 PowerShell/Python。
- 不修改无关领域代码或 `CodingKit/agents/skills`。
- 保持现有 CI 在兼容期可运行。
- 每个 Phase 单独验证、提交和保留回滚边界。
- 不创建 Release、Tag 或自动合并 PR。

## Phase 1：CLI 契约

- 修改：`cmd/aicoding/main.go`、`internal/cli`、`internal/report` 和契约测试。
- 目标：统一 help、参数解析、退出码和 `CLI_DEPRECATED`。
- 风险：已有脚本依赖错误退出码或帮助文本。
- 验证：CLI subprocess tests、`go test ./internal/cli ./internal/report`、Smoke。

## Phase 2：唯一测试引擎

- 修改：新增 `internal/testengine`，迁移 `tools/aicoding-global-tester` 和 `internal/cli/test.go`。
- 目标：单一 Registry、Profile、Timeout、Runner、Report、ExitCode。
- 风险：报告字段、用例覆盖率或日志路径漂移。
- 验证：Golden JSON/Markdown、旧用例覆盖对比、Go tests。

## Phase 3：聚合与兼容入口

- 修改：`internal/cli/cli_ext.go`、test engine、`internal/kit/freshclone.go`、Taskfile、CI。
- 目标：删除 Full/Release/CI 递归聚合，旧入口统一路由并提示废弃。
- 风险：遗漏 Release-only gate 或 fresh-clone 形成新递归。
- 验证：Smoke、Full、调用图扫描、每个 test ID 单次执行。

## Phase 4：生命周期

- 修改：新增轻量 `internal/lifecycle` 编排层，调整 `internal/kit`、`internal/mcpcontrol` 和 CLI。
- 目标：统一 plan/install/update/status/doctor/verify/uninstall/rollback。
- 风险：默认写操作影响用户 Codex 配置或 runtime Skill。
- 验证：全部 dry-run、临时 Codex config、runtime Skill audit、rollback tests。

## Phase 5：Doctor、Verify 与 Report

- 修改：`internal/repohealth`、`internal/report`、CLI 和 `config/schemas`。
- 目标：明确诊断、静态验证、测试、发布职责，严格 JSON stdout。
- 风险：现有 JSON 消费者不兼容。
- 验证：Schema、退出码矩阵、JSON-only stdout 回归。

## Phase 6：文档和边界

- 修改：README 三件套、`docs/COMMANDS.md`、Architecture、AGENTS、DocSync、导航和测试文档。
- 目标：一个功能只有一个正式文档入口。
- 风险：链接、生成区或 DocSync policy 漂移。
- 验证：DocSync、Markdown links、governance layout/dependencies。

## Phase 7：清理与全量验收

- 删除只因旧实现存在的 tester、wrapper、静态替代检查和重复文档陈述。
- 运行用户要求的全部验证并确认 worktree、submodule clean。

## 回滚

每个 Phase 通过独立 commit 回滚；不使用 `git reset --hard`，不改写共享历史。
