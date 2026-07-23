# TODO 0045: VALIDATION_EVIDENCE_BUDGET 归档瘦身

Status: Done
Verify: bin/aicoding.exe docsync all --json

## 范围

- 把主文档 §9–11 的过程性样本逐字迁移到 `docs/operations/evidence/` 归档。
- 主文档保留结论与归档链接。
- §12、§13、§14、§15（ADR 0014 默认值翻转记录）原样保留。
- 归档不是删除；提交信息明确可用 `git log --follow` 追溯。

## 完成条件

- 原 §9–11 的 `9372` 个字符逐字写入
  `docs/operations/evidence/validation-evidence-phases-3-5-archive.md`；源块与归档块
  SHA-256 均为
  `f483400103d7bd449206e7dfe231ce0df91a7c2bf4e299e55b71b913087334ce`。
- 主文档从 `437` 行收敛为 `266` 行，只保留三期结论与归档链接。
- §12 至文件末尾的 `4792` 个字符在搬运前后 SHA-256 均为
  `8871fd1d745e8c2f6b93f1e2334f787c4d472649cc22a64f39373bb29999f677`；
  §12、§13、§14、§15 与 ADR 0014 翻转记录逐字未变。
- 主文档历史与归档提交可由 `git log --follow` 分别追溯；提交信息显式记录证据搬运。
- 主文档链接、docsync all 与 Markdown fragment 检查全绿。
- 最终 Release summary：`test-results/0046-final-release/summary.json`。
