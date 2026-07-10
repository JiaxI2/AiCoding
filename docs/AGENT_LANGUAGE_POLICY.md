# AiCoding 语言策略

AiCoding 默认中文优先。面向用户的文字应先用中文说明，再按需保留英文术语括注。

## 必须中文优先的内容

- 权限摘要和命令目的说明。
- 执行计划、工作进度、验证结果和失败原因。
- hook/script message、warning、error、summary、recommendation 等人读字段。
- 风险说明、rollback/handoff 说明和下一步建议。
- 用户选择技术路线、门禁阻塞、规格同步和发布说明。

## 不翻译的内容

- JSON key、schema id、code 枚举和机器读取字段。
- 文件名、路径、命令、参数、tag、commit SHA。
- 约定术语可保留英文，但应写为中文 + 英文括号，例如计划模式（Plan Mode）、注册表（registry）、门禁（gate）、子模块（hook module）。

## 示例

- 不推荐：英文权限摘要，例如“英文读取 registry 授权摘要”。。
- 推荐：读取 Plan Mode registry，用于验证前检查。
- 不推荐：英文验证摘要，例如“Validation-passed.”。
- 推荐：验证通过。
- 不推荐：英文阻塞摘要，例如“Implementation-blocked because NEEDS_USER_DECISION exists.”。
- 推荐：检测到 `docs/spec/NEEDS_USER_DECISION.md`，用户尚未选择技术路线，禁止继续实现。

## Codex 权限请求要求

当 Codex 需要请求用户授权执行命令时，必须用中文说明命令目的。例如：

- 读取 Plan Mode registry，用于验证前检查。
- 运行 Plan Mode 验证脚本，确认规格、计划、任务和决策记录完整。
- 读取规格文档，用于判断是否需要用户选择技术路线。
