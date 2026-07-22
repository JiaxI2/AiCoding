# TODO 0033: verify-codex-kit Phase 2 退役

Status: Done
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
dependencies、lint 全绿。最终 Full/Release 使用下述固定 summary 路径验收。

## 明确不做

- 不删除任何其他 PowerShell 脚本。
- 不修改 test engine、report、CLI 命令或 PowerShell 保留类别。
- 不触碰只读 Codex-Skills 子模块中的历史/上游引用。

## 完成证据（2026-07-22）

- Phase 2 实现提交：`ff4948148f9d28b9a42873cbe46179e383f76853`。
- `v1.1.0` commit 为 `112e111f78f5731d3e570bcd1354e9a59996ba24`；
  `merge-base --is-ancestor 2a8b49af12386787eb8db112da66cf736882cb84 v1.1.0`
  返回 `0`。
- 仅删除 `tools/specialty/verify-codex-kit.ps1`，并移除 Full 当前说明与
  `aicoding-platform` export manifest 中只匹配该脚本的活跃引用；历史记录和子模块零改动。
- 删除后实测 `doctor pwsh` 为 `remainingScripts=19 / thinShells=1 / deprecated=1 /
  unspecified=0`；PWSH-002、dependencies、lint、DocSync `840/0/0` 全绿。
- 首次 Full 在 `test-results/aicoding-global-test-20260722-170557/summary.json` 真实抓到
  `EXP-002` export include 残留；精确撤销该引用后，Full 在
  `test-results/aicoding-global-test-20260722-171429/summary.json` 以
  `71 total / 67 pass / 0 fail / 0 warn / 4 skip` 通过。
- 本轮最终 Full：`test-results/0032-final-full/summary.json`；最终 Release：
  `test-results/0032-final-release/summary.json`。二者对包含 A/B/C 归档的同一 staged tree
  真跑并作为最终验收。
