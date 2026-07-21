# TODO 0002: PowerShell 专项脚本收敛（不含 ai-debug-repair-kit / jtag / ccsdebug）

Status: Done
Verify: bin/aicoding.exe doctor pwsh-budget --json 保持绿、PWSH-003 阻断默认 PowerShell 编排，且 22 个保留脚本均可归入冻结专项类别或已登记的兼容退役窗口

## 2026-07-22 重新评估与裁决

本项立项时的“脚本数量必须下降”是收敛手段，不是产品保证。此后
`POWERSHELL_BOUNDARY.md` 已进入 **Accepted and Frozen**，0004/0006 已把 Plan Mode
强制语义迁入 Go 并保留兼容薄壳，0022 又把退役计数纳入 `doctor pwsh`，明确“只报数、
不设门禁；冻结面自然减少，不为归零重写”。因此本次以当前可执行职责重新盘点，而不是继续
沿用旧数量目标。

当前 `tools/specialty/**/*.ps1` 共 22 个，逐文件归属如下：

| 冻结类别 / 过渡状态 | 文件 | 数量 | 裁决 |
|---|---|---:|---|
| tag planning | `aicoding-tag-governance.ps1` | 1 | 保留非破坏性 tag 审计/计划 |
| release overlay compatibility | `verify-release-governance-overlay.ps1` | 1 | 保留显式 release 慢路径 |
| PowerShell quality / 专项 Kit 资产验证 | `status/test/verify-aicoding-agent-dev-kit.ps1`、`status/test/verify-codex-agent-powershell-skill-kit.ps1`、`verify-agent-engineering-foundation.ps1` | 7 | 只由人工专项流程调用；不进入默认 profile |
| Plan Mode helpers | `confirm-agent-decision.ps1`、`invoke-aicoding-agent-hook.ps1`、`new-agent-plan-mode-session.ps1`、`hooks/aef/plan-mode-gate.ps1`、`hooks/aef/spec-artifact-gate.ps1` | 5 | 交互/兼容 helper；强制裁决已归 Go |
| external skill workflows | `aicoding-skill.ps1`、`audit-runtime-skills.ps1`、`set-codex-skill-profile.ps1` | 3 | 保留跨仓下载、安装、profile、junction 与审计状态边界 |
| safety / hardware / toolchain | `status/test/verify-ai-debug-repair-kit.ps1` | 3 | 保留 DSS/XDS/flash 独立安全边界，本项不触碰 |
| 已登记兼容退役窗口 | `verify-agent-dev-kit-plan-mode.ps1`、`verify-codex-kit.ps1` | 2 | 不拥有新语义；分别服从 ADR 0009 与独立 Retirement Plan 的 release 节奏 |

`doctor pwsh` 对顶层 20 个脚本报告 `remainingScripts=20 / thinShells=2 /
deprecated=2`；嵌套的两个 AEF Hook 薄壳使物理文件总数为 22。这里的 2 个兼容壳不是
第二控制面：`verify-codex-kit.ps1` 的 Phase 2 明确要求 Phase 1 至少落地一个发布版本后再删，
Plan Mode 壳也按 ADR 0009 的 release 阶段退役。现在抢先删除会违反各自的独立计划和验证，
不是“收敛”。

默认路径检查同时成立：Taskfile 的 doctor/verify/Smoke/Full/Release 全部直达
`bin/aicoding.exe`；CI 的 `shell: pwsh` 只用于核对 Go toolchain，实际测试 profile 由 Go CLI
执行；PWSH-003 是所有 profile 的 Required leaf gate。`doctor pwsh-budget` 实测
`hot-path=0 / slow-path=1 / fallback=6 / documentation-only=44` 且退出 0；slow/fallback
全部对应上表的显式专项或兼容调用。

**最终裁决：PowerShell 专项边界已经达成，0002 完成。** 专项面继续“只减不增”；后续只在
各自 release 退役条件成熟时删除薄壳，不为追求脚本数归零重写合法的专项流程，也不把
外部 Skill、Plan Mode 交互或硬件安全语义搬进 Go 内核。

## 背景（Graph First：找重复路径与万能节点）

`tools/specialty/` 24 个脚本里存在两处可收敛结构，`POWERSHELL_BOUNDARY.md` 本身也把方向
定为"专项面停止增长、新能力进 Go"。**明确排除硬件/安全链**：
`status/test/verify-ai-debug-repair-kit.ps1`（jtag / ccsdebug / DSS/XDS / flash）本轮不动。

**重复路径 A —— 每个 kit 的 status/test/verify 三件套**（并行的外围节点）：
- `aicoding-agent-dev-kit`（status/test/verify + `verify-agent-dev-kit-plan-mode` + `verify-agent-engineering-foundation`）
- `codex-agent-powershell-skill-kit`（status/test/verify + `verify-codex-kit.ps1`）
这是同一形状（每 kit status/test/verify）复制多份。`verify-codex-kit.ps1` 退役已 Active
（Phase 0/1 完成，见 `docs/decisions/verify-codex-kit-retirement/`），是本收敛的模板。

**万能节点 B —— `aicoding-skill.ps1`（1192 行）**：内含大量通用 helper（路径解析、JSON 读写、
result 信封、skill-id 校验、sources 读取）与 skill install/verify/audit/status/profile 多职责；
其 audit 逻辑与 `audit-runtime-skills.ps1`（287）、`lib/AiCoding.SkillAudit.psm1`（331）重叠。

## 历史收敛计划及本次处置

**阶段 1（独立退役计划继续）**：`verify-codex-kit.ps1` 的结构检查已由 Go
`test --profile Full` / leaf gate 覆盖，但其 Retirement Plan 明确要求 Phase 1 至少落地一个
发布版本后才能执行 Phase 2。本项不合并或绕过该时间条件。

**阶段 2（不再以数量为目标强制执行）**：对 `aicoding-agent-dev-kit` /
`codex-agent-powershell-skill-kit` 的 status/test/verify，凡属**确定性结构检查**的，登记为
`internal/testengine` 的 Go leaf gate（如 `ADK-00x`），像 `RC-001` 一样，然后按"单独计划+验证"
逐个退役对应 ps1。重新盘点确认现有脚本验证的是打包产物中的 Python/PowerShell 专项工具、
AST/安全规则和 runtime mirror，不是默认产品结构门禁；没有现实依据把整套显式专项行为复制进
Go。Plan Mode 交互类继续保留为专项（非结构门禁）。

**阶段 3（不做无收益重构）**：原计划把 `aicoding-skill.ps1` 的通用 helper 上移到
`lib/CodexKit.psm1` / `lib/AiCoding.SkillAudit.psm1`；skill-audit 逻辑统一到单一实现
（`audit-runtime-skills.ps1` 复用模块，不各自再实现）；瘦身 1192 行的万能脚本。当前脚本拥有
外部下载缓存、安装日志、user-created Skill 与跨仓安全边界；runtime audit/profile 又是
lifecycle 的显式 specialty adapter。没有新增缺陷或第二默认权威，单为行数搬 helper 会扩大
耦合和回归半径，故不执行。

## 完成定义（按冻结边界裁决后的绿灯）

- `bin/aicoding.exe doctor pwsh-budget --json` 保持绿（调用仍在保留类别内、且专项面未增长）；
- PWSH-003 作为 Go leaf gate 阻断默认入口回退到 PowerShell，并在 Full 绿；
- 22 个现存脚本全部属于冻结专项类别或有独立退役记录的兼容薄壳；
- 本项未移除脚本；未来移除仍须独立退役记录并遵守 release 窗口；
- 本项 Status 改为 Done。

## 明确不做

- **不动** `*-ai-debug-repair-kit.ps1`（jtag/ccsdebug/DSS/XDS/flash 安全链），及其独立安全边界。
- 不新增专项脚本、不新增保留类别（边界第 41 条）。
- 不为收敛而改内核六模块——收敛发生在外围 ps1 → Go 中心节点，中心节点零改动。

## Graph First 对齐

重新评估后，默认结构门禁已经收进唯一测试引擎；现存 status/test/verify 是显式专项 Kit
资产检查，不再与默认 profile 构成平行控制面。继续把领域状态、外部下载、AST 或硬件安全
行为搬进中心节点，反而会让中心理解所有领域。保留有界 specialty leaves、由 Go catalog/
lifecycle 只持路径和调用边界，更符合 [GRAPH_FIRST](../architecture/GRAPH_FIRST.md) §5 与
`POWERSHELL_BOUNDARY.md` 的“中心稳定、外围按需组合”。

## 实测证据（2026-07-22）

```text
go test ./internal/repohealth ./internal/testengine                      PASS
doctor pwsh                                                             exit 0, ok=true
retirement                                                              remaining=20, thin=2, deprecated=2
doctor pwsh-budget                                                      exit 0, ok=true
budget                                                                  hot=0, slow=1, fallback=6, docs=44
powershell regex-lint --path tools/specialty                            exit 0, ok=true
docsync all / governance layout / verify repo-text                      PASS
test --profile Smoke --reuse off                                        45 PASS / 22 SKIP / 0 FAIL
PWSH-003（REQUIRED）                                                     PASS, 1 ms
todolist                                                                24 Done / 1 Planned（仅 0019）
```

负例使用临时测试夹具把 doctor/verify/Smoke/Full/Release 五个 Taskfile 默认任务全部改成
`pwsh -File ...`，真实调用既有 `checkTaskfileGoRoutes`。探针退出 0 是因为“预期拒绝”断言通过，
门禁原文为：

```text
PWSH-003 rejected PowerShell-only defaults: taskfile missing Go-native default routes: doctor, verify, smoke, full, release
```

临时探针文件已删除，未进入本项提交；该负例证明冻结面不是文档约定，默认入口一旦回退到
PowerShell 会被 Required leaf gate 阻断。
