# TODO 0002: PowerShell 专项脚本收敛（不含 ai-debug-repair-kit / jtag / ccsdebug）

Status: Planned
Verify: bin/aicoding.exe doctor pwsh-budget --json 保持绿 且被收敛脚本的结构检查已作为 Go leaf gate 登记

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

## 收敛计划（把并行 ps1 节点收进中心节点，分阶段、逐脚本单独验证）

**阶段 1（低风险，先做）——完成已在途的退役**：按退役计划落地 `verify-codex-kit.ps1`
Phase 2；把它的结构检查确认已由 Go `test --profile Full` / leaf gate 覆盖后移除脚本。

**阶段 2 —— 三件套 → Go 单一测试引擎**：对 `aicoding-agent-dev-kit` /
`codex-agent-powershell-skill-kit` 的 status/test/verify，凡属**确定性结构检查**的，登记为
`internal/testengine` 的 Go leaf gate（如 `ADK-00x`），像 `RC-001` 一样，然后按"单独计划+验证"
逐个退役对应 ps1。Plan Mode 交互类（`new-agent-plan-mode-session`、`confirm-agent-decision`、
`invoke-aicoding-agent-hook`、`hooks/aef/*`）保留为专项（非结构门禁）。

**阶段 3 —— 去重 skill 工具链**：把 `aicoding-skill.ps1` 的通用 helper 上移到
`lib/CodexKit.psm1` / `lib/AiCoding.SkillAudit.psm1`；skill-audit 逻辑统一到单一实现
（`audit-runtime-skills.ps1` 复用模块，不各自再实现）；瘦身 1192 行的万能脚本。

## 完成定义（绿灯）

- `bin/aicoding.exe doctor pwsh-budget --json` 保持绿（调用仍在保留类别内、且专项面未增长）；
- 被收敛脚本的结构检查已作为 Go leaf gate 登记并在 `test --profile Full` 绿；
- `tools/specialty/` 非硬件脚本数量下降、无重复 helper 实现；
- 每个被移除脚本都有单独的退役记录（遵守 `POWERSHELL_BOUNDARY.md` 第 45 条）；
- 本项 Status 改为 Done。

## 明确不做

- **不动** `*-ai-debug-repair-kit.ps1`（jtag/ccsdebug/DSS/XDS/flash 安全链），及其独立安全边界。
- 不新增专项脚本、不新增保留类别（边界第 41 条）。
- 不为收敛而改内核六模块——收敛发生在外围 ps1 → Go 中心节点，中心节点零改动。

## Graph First 对齐

把"每 kit 一套 ps1 检查"的并行外围节点，收进"唯一测试引擎"这个中心节点——减少节点/边、
提高复用；万能脚本拆分 + helper 上移到 lib 模块 = 去重实现。收敛方向与
[GRAPH_FIRST](../architecture/GRAPH_FIRST.md) §5、`POWERSHELL_BOUNDARY.md` 一致。
