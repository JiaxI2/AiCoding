# 任务（Tasks）：product-convergence

## Phase 0: 决策 / 计划

- [x] 创建独立 worktree 并初始化 submodule。
- [x] 完成产品能力树、问题分级和方案评审。
- [x] 选择方案 A 并记录正式/兼容/内部边界。

## Phase 1: CLI 契约

- [ ] 统一全局和子命令 help。
- [ ] 统一参数错误退出码 `2`。
- [ ] 增加 `CLI_DEPRECATED` 兼容提示。
- [ ] 保持 `report.Result` 兼容字段。

## Phase 2: 唯一测试引擎

- [ ] 内聚 global tester 为 `internal/testengine`。
- [ ] 统一 Smoke/Full/Release Registry、Profile、Timeout、Report、ExitCode。
- [ ] 保持 JSON/Markdown 报告兼容。

## Phase 3: 聚合与兼容

- [ ] 删除 Full→Full、Release→Full、CI→Smoke 等递归调用。
- [ ] `test --profile` 成为正式测试入口。
- [ ] 旧入口保留一个版本并统一路由。

## Phase 4: 生命周期

- [ ] 建立静态 lifecycle adapter。
- [ ] 收敛 kit/MCP/runtime Skill plan、apply、status、doctor、verify。
- [ ] 保持所有写操作显式且可回滚。

## Phase 5: Doctor / Verify / Report

- [ ] 建立产品级 doctor 和 verify 边界。
- [ ] 统一报告 Schema 和严格 JSON stdout。
- [ ] 补充帮助、退出码和兼容回归。

## Phase 6: 文档

- [ ] 收敛 README 三件套和命令文档。
- [ ] 更新 Architecture、AGENTS、DocSync 和 navigation。
- [ ] 旧入口仅出现在兼容表和历史决策中。

## Phase 7: 验收

- [ ] 执行全部用户要求的验证。
- [ ] 分阶段提交完整、未 squash。
- [ ] 工作区和 submodule clean。
