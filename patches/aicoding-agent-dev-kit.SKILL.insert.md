## Plan Mode 中文优先插入片段

面向用户的流程说明必须中文优先。英文术语可以保留，但应写成中文 + 英文括号，例如计划模式（Plan Mode）、门禁（gate）、注册表（registry）。

### 模糊需求处理

1. 模糊需求不要直接实现。
2. 先创建或更新 `spec/PRD_OPTIONS.md`。
3. 给出 2-5 个技术路线选项，说明优点、缺点、风险、验证方式、工作量和推荐条件。
4. 请求用户选择技术路线。
5. 用户选择前，不修改架构敏感文件。
6. 用户选择后，写入 `spec/SELECTED_SOLUTION.md` 和 `.agent-memory/DECISIONS.md`。
7. 再生成或更新 `spec/IMPLEMENTATION_PLAN.md`、`spec/TASKS.md`、`spec/TRACEABILITY.md`、`spec/CHECKLIST.md`。

### 权限摘要要求

请求执行命令时，必须用中文说明目的，例如：读取 Plan Mode registry，用于验证前检查。
