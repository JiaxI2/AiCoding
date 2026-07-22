# TODO 0033: verify-codex-kit Phase 2 退役

Status: Planned
Verify: 严格按 Retirement Plan Phase 2 删除兼容脚本；Full、docsync、dependencies、lint 全绿且活跃引用归零

## 触发证据

Phase 1 提交 `2a8b49af12386787eb8db112da66cf736882cb84` 已被正式稳定版
`v1.1.0` 包含，`git merge-base --is-ancestor` 返回 0；其后已有 `v1.2.0-rc.1`。
因此 `docs/decisions/verify-codex-kit-retirement/RETIREMENT_PLAN.md` 的 Phase 2
发布窗口已经满足。

## 范围

1. 删除 `tools/specialty/verify-codex-kit.ps1`。
2. 移除仍把该兼容脚本描述为当前入口的活跃文档引用；历史 CHANGELOG、已完成 todo、
   Traceability 与本 Retirement Plan 保留。
3. 将 Retirement Plan 标为 Completed，记录触发 commit/tag 包含关系、提交与验证输出。
4. CHANGELOG 以既有兼容入口退役风格记录删除。

## 验证

```powershell
git grep -n "verify-codex-kit"
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe governance dependencies --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Full --json
```

通过判据：脚本删除；活跃引用归零；允许的命中仅为历史记录与本退役计划；Full、DocSync、
dependencies、lint 全绿。最终 Full/Release summary 路径在本轮最终验收后回填。

## 明确不做

- 不删除任何其他 PowerShell 脚本。
- 不修改 test engine、report、CLI 命令或 PowerShell 保留类别。
- 不触碰只读 Codex-Skills 子模块中的历史/上游引用。
