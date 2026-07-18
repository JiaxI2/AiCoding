# Retirement Plan: verify-codex-kit.ps1

Plan Status: Proposed（Phase 0 已随本轮修复落地；Phase 1/2 待验收后执行）

目标：按 [POWERSHELL_BOUNDARY.md](../../architecture/POWERSHELL_BOUNDARY.md) 的
"不删除专项脚本除非有单独计划和验证"规则，为 `tools/specialty/verify-codex-kit.ps1`
提供该单独计划。不新增 CLI 命令，不修改 test engine 或 report 契约，不触碰其他专项脚本。

## 背景与边界依据

脚本演变事实：

| 时间 | 提交 | 状态 |
|---|---|---|
| 2026-07-03 前后 | `54d5f71` 等 | `scripts/verify-codex-kit.ps1` 是真实的 Smoke 级专项验证器（submodule/plugin/asset 检查） |
| 2026-07-09 | `3813747` | 专项逻辑收编进 Go 控制面，脚本掏空为 `aicoding full` 兼容别名的薄包装 |
| 2026-07-10 | `f656686` | 随目录治理迁至 `tools/specialty/` |
| 2026-07-18 | `bce8282`（v1.0.0） | `full` 兼容路由到期移除，包装脚本随之硬失败（`unknown command: full`，`errorKind=usage`，退出码 2） |

对照 POWERSHELL_BOUNDARY.md 的判定：

- 脚本不属于六个保留类别（tag planning、release overlay compatibility、PowerShell
  quality、Plan Mode helpers、external skill workflows、safety/hardware/toolchain）中的任何一个；
- 其现状恰好是禁止事项"不把 Go 默认入口重新包装成 PowerShell"所指的形态；
- 其原始职责已被 Go 控制面完全覆盖：重门禁 = `bin\aicoding.exe test --profile Full --json`，
  Smoke 级 kit 检查 = `bin\aicoding.exe kit verify --all --profile Smoke --json`。

结论：脚本应退役。但按"单独计划和验证"规则不直接删除，且在引用迁移完成前保持
可用（Phase 0 修复），复用 `full` 别名本身"兼容窗口 → 到期移除"的既有先例（`bce8282`）。

## 引用清单（实现前基线）

仓库内对脚本的活跃引用：

| 位置 | 性质 | 处置 |
|---|---|---|
| `AGENTS.md:141`（升级工作流 step 5）、`AGENTS.md:157`（Required Verification） | 活跃门禁引用 | Phase 1 迁移至正式入口 |
| `CodingKit/README.md:71` | 使用说明 | Phase 1 迁移 |
| `docs/operations/KIT_LIFECYCLE_TEST_PROFILES.md` | 曾错误声称脚本是 Smoke 默认门禁 | Phase 0 已改为如实描述 |
| `.agents/skills/aicoding-agent-patch-kit/SKILL.md:31,37` | 指向已不存在的 `scripts\verify-codex-kit.ps1` 旧路径（`f656686` 起即失效的既有漂移） | Phase 1 一并修正 |
| `docs/decisions/aicoding-architecture/TRACEABILITY.md:49` | 历史验收证据 | 不迁移，历史记录保持原样 |
| `CodingKit/agents/skills/**/SKILL.md`（Codex-Skills submodule 内） | 跨仓引用（同为 `scripts/` 旧路径漂移） | Phase 1 记录为上游事项，随下次 submodule 升级在 Codex-Skills 仓修正 |

## Phase 0：修复包装（本轮已落地）

- 失效调用改为正式入口 `test --profile Full --json --repo-root <repo>`，保留 `go run` 回退；
- 判读改走 JSON 契约：`ok=true` → 退出 0；`errorKind=usage` → 退出 2；其余失败 → 退出 1；
  stdout 非 JSON 时按执行失败处理并透传 CLI 退出码；
- 增加兼容提示（显式写 stderr——子进程 pwsh 会把 `Write-Warning` 渲染到 stdout，
  会破坏 `-Json` 的严格 JSON stdout 透传）指向本计划；
- 同步修正 KIT_LIFECYCLE_TEST_PROFILES.md 的过时声称与 CHANGELOG Unreleased。

验证（全部退出码 0 且 `ok=true`）：

```powershell
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe powershell regex-lint --path tools/specialty/verify-codex-kit.ps1 --json
bin\aicoding.exe test --profile Smoke --json
pwsh -NoProfile -ExecutionPolicy Bypass -File tools\specialty\verify-codex-kit.ps1 -Json   # 端到端 Full
```

- [x] 脚本 `-Json` stdout 是无前缀/后缀、无替换字符的严格 UTF-8 JSON，退出码与 JSON 判读一致；
- [x] 除脚本、本计划、KIT_LIFECYCLE_TEST_PROFILES.md、CHANGELOG 外零改动。

## Phase 1：引用迁移（待验收后执行）

- `AGENTS.md` 两处 `verify-codex-kit` 门禁改为 `bin\aicoding.exe test --profile Full --json`；
- `CodingKit/README.md:71` 同步替换；
- `.agents/skills/aicoding-agent-patch-kit/SKILL.md` 修正旧路径漂移，直接指向正式入口；
- Codex-Skills 上游 SKILL.md 的 `scripts/verify-codex-kit.ps1` 门禁行提交上游修正，
  随下次按 AGENTS.md Cross-Repository Upgrade Workflow 的 submodule 升级带入本仓。

验证：

```powershell
# 期望：仅命中本计划与 TRACEABILITY 历史记录（及未升级前的 submodule 内文件）
git grep -n "verify-codex-kit"
bin\aicoding.exe docsync --json
bin\aicoding.exe test --profile Smoke --json
```

- [ ] 仓库自有文件中不再有指向脚本的活跃门禁引用；
- [ ] docsync 与 Smoke 全绿。

## Phase 2：移除（Phase 1 落地满一个发布版本后）

- 删除 `tools/specialty/verify-codex-kit.ps1`；
- CHANGELOG 记录移除，与 `bce8282` 兼容路由移除条目同风格；
- 本计划状态改为 Completed，作为移除的"单独计划和验证"存档。

验证：

```powershell
git grep -n "verify-codex-kit"        # 期望：仅历史记录与本计划
bin\aicoding.exe doctor pwsh --json
bin\aicoding.exe doctor pwsh-budget --json
bin\aicoding.exe test --profile Smoke --json
```

## 回滚

各 Phase 独立提交，回滚即 `git revert` 对应提交。脚本无状态、无数据迁移；Phase 2
回滚只需恢复文件并撤销 CHANGELOG 条目。
