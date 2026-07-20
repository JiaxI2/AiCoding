# Plan Mode 产物目录

每个计划使用独立的 `docs/spec/<plan-id>/` 目录，`plan-id` 必须为小写 kebab-case。

- `PLAN.md`：唯一必需文件，头部 frontmatter 遵循
  `config/schemas/plan-spec.schema.json`。
- `OPTIONS.md`：可选，多方案对比。
- `DECISION.md`：可选，用户决策；存在 `OPTIONS.md` 时必须存在。
- `TASKS.md`：可选，任务拆解。

使用 `bin\aicoding.exe plan verify --json` 校验全部计划，使用
`bin\aicoding.exe plan status --all --json` 查看确定性状态投影。历史归档可以保留额外附件，
但新会话不得再创建根目录单槽文件。
