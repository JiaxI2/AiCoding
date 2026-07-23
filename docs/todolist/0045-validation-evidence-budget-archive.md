# TODO 0045: VALIDATION_EVIDENCE_BUDGET 归档瘦身

Status: Planned
Verify: bin/aicoding.exe docsync all --json

## 范围

- 把主文档 §9–11 的过程性样本逐字迁移到 `docs/operations/evidence/` 归档。
- 主文档保留结论与归档链接。
- §12、§13、§14、§15（ADR 0014 默认值翻转记录）原样保留。
- 归档不是删除；提交信息明确可用 `git log --follow` 追溯。

## 完成条件

- 搬运内容逐字一致，可由脚本比较证明。
- 主文档链接、docsync all 与 Markdown fragment 检查全绿。
- 最终 Release summary：待 0046 收口提交前回填。
