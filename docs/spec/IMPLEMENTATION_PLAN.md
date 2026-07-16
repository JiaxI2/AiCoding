# 实现计划（Implementation Plan）：issue-lifecycle-governance

Plan Status: Approved

## 上下文

将 Issue 创建、分类、流转与关闭标准落地为 AiCoding 仓库级 Git governance policy，并保持 Skill source/submodule 只读边界

## 已选择架构

复用现有 `aicoding-git-governance` 的通用 Git/Release 标准，在 AiCoding 内新增仓库级 Issue policy、结构化表单、label manifest、workflow、Go lint/tests 和平台文档。本轮不修改、复制或重新生成 Codex-Skills Skill source。

## 约束

- 不引入第二个 Issue Skill 名称，也不宣称本地仓库策略已经成为 runtime Skill 能力。
- `CodingKit/agents/skills` 保持只读且 clean；generated plugin skill 不手改。
- Issue 自动化只同步 label 和规范单值轴；不自动关闭 stale Issue。
- 关闭必须具备 resolution，且保留人工确认关闭理由、结果摘要与证据链接。
- 不直接修改 Codex plugin cache；Skill gitlink 只指向已发布、远端可解析的提交。

## 验证计划

- Codex-Skills：只验证所消费的已发布 submodule commit/tag 与 clean 状态，不运行不存在的 Issue renderer/validator。
- AiCoding：Go governance 单元测试、`go test ./...`、governance lint、DocSync、Markdown links、Smoke/Full 适用门禁、workflow 静态检查与 `git diff --check`。
- 静态核对 Issue Forms、label manifest、workflow event/permission/action 版本。

## 回滚

回滚只针对本任务文件：AiCoding 新文件可在用户确认后移除，现有文件只还原本任务 hunks。不得使用 `git reset --hard` 或清理用户未提交改动。
