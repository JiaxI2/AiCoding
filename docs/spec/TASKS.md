# 任务（Tasks）：issue-lifecycle-governance

## Phase 0: 决策 / 计划

- [x] 确认不需要继续选择技术路线；用户已指定建立仓库级 Issue governance。
- [x] 记录仓库级 Issue policy、runtime Skill 不变与 submodule 只读发布边界。

## Phase 1: 实现

- [x] 为 AiCoding 添加 Issue templates、label workflow、Go lint/tests 与文档。
- [x] 保持 `bin/aicoding.exe governance lint --json` 作为平台检查入口。
- [x] 明确本轮不修改 Codex-Skills canonical Skill 或 generated plugin。

## Phase 2: 验证

- [x] 运行 AiCoding Go tests、governance lint、DocSync 和 Smoke/Full/Release 门禁。
- [x] 运行 Issue workflow 静态检查、Plan Mode、PowerShell、hook-equivalent 与 `git diff --check`。
- [x] 验证所消费的 Codex-Skills commit/tag 已发布且 submodule clean。

## Phase 3: 交接

- [x] 将本任务并入用户确认的全仓库发布快照。
- [x] 总结已实现、已验证、未验证与回滚方法。
- [x] 明确未来若提升为 reusable Skill policy，必须走独立 Codex-Skills 实现与发布流程。
