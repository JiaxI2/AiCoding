# 可追溯性（Traceability）：issue-lifecycle-governance

| 需求 / 决策 | 计划章节 | 任务 | 验证 |
|---|---|---|---|
| Issue 创建必须结构化 | AiCoding repository Issue policy + Issue Forms | AiCoding `.github/ISSUE_TEMPLATE` | governance lint；workflow static checks |
| Issue 必须分类与流转 | label axes + managed lifecycle profile | label manifest；issue workflow | Go unit tests；workflow static checks |
| Issue 关闭必须有依据 | resolution/summary/evidence gate | repository policy；closed/reopened normalization | Go unit tests；workflow static checks |
| 不新增运行时 Skill | repository policy boundary | 不修改 `aicoding-git-governance`，不新增 Issue Skill | submodule diff；runtime Skill audit |
| 保持 kit 生命周期边界 | released Skill dependency -> read-only submodule | AiCoding 只消费已发布 gitlink | remote tag check；submodule clean check |
