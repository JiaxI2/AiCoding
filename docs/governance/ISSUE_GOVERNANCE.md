# Issue 生命周期治理 / Issue Lifecycle Governance

## 归属与权威源

Issue 创建、分类、状态流转、重开和关闭属于 AiCoding 的仓库级 Git governance policy。本版本不创建独立 Issue Skill，也不修改 Codex-Skills 中的 canonical Skill source。

通用 Git/Release 标准仍由 Codex-Skills 提供：

- `platform/aicoding-git-governance/references/aicoding-git-governance-standard.md`

Issue 专项规则的权威实例位于本仓库：

- `.github/repository-governance.toml`
- `.github/ISSUE_TEMPLATE/`
- `.github/issue-labels.json`
- `.github/workflows/issue-governance.yml`
- `docs/governance/ISSUE_GOVERNANCE.md`

这些资产由 AiCoding 直接拥有并由 Go governance lint 验证，不宣称来自尚未发布的 Skill renderer。

## 创建门禁

普通贡献者必须选择以下结构化表单，`blank_issues_enabled` 固定为 `false`：

- Bug：当前/预期行为、最小复现、影响、环境、证据和完成条件。
- Feature：问题、目标结果、范围、验收条件、替代方案和可追溯信息。
- Governance：当前缺口、建议规则、生命周期影响、验证、兼容性和回滚。

安全漏洞使用 `SECURITY.md` 指定的私密渠道，不能创建公开 Issue。

## 分类标准

完成 triage 后，每个开放 Issue 必须满足：

| 轴 | 基数 | AiCoding 值 |
|---|---:|---|
| `type:*` | 1 | `bug`、`feature`、`governance` |
| `area:*` | 1..N | `platform`、`kit`、`plugin`、`skill`、`docs`、`ci` |
| `priority:*` | 1 | `p0`、`p1`、`p2`、`p3` |
| `status:*` | 1 | `needs-triage`、`needs-info`、`ready`、`in-progress`、`blocked` |
| `resolution:*` | 0 | 开放 Issue 禁止保留 resolution |

优先级含义：

- `p0`：安全紧急事件、生产事故或发布阻塞；
- `p1`：关键功能不可用或高影响回归；
- `p2`：正常计划工作；
- `p3`：低紧急度或 backlog 候选。

## 状态流转

```text
status:needs-triage
-> status:needs-info | status:ready
-> status:in-progress | status:blocked
-> closed + resolution:*
```

进入 `status:ready` 前必须记录 owner、验收条件、依赖和追踪目标。`status:blocked` 必须写明 blocker 与解除条件。同一时刻只能保留一个 `status:*`。

## 关闭标准

关闭前必须同时具备：

1. 恰好一个 `resolution:*`；
2. 与 resolution 一致的 GitHub close reason；
3. 简洁的结果摘要；
4. PR、commit、release、decision、验证报告或 canonical duplicate Issue 等证据链接。

Resolution 规则：

- `resolution:completed`：验收条件已满足，链接实现与验证；
- `resolution:duplicate`：链接 canonical Issue，并迁移独有证据；
- `resolution:not-planned`：写明产品或治理理由；
- `resolution:invalid`：解释无效假设、不支持范围或不可复现结论。

重开 Issue 时移除所有 `resolution:*` 并恢复 `status:needs-triage`。stale 时间只能触发提醒，不能单独作为关闭依据。

## 自动化边界

`.github/workflows/issue-governance.yml` 使用最小权限 `contents: read` 与 `issues: write`，负责：

- 根据 `.github/issue-labels.json` 创建或更新声明的 label，不删除未知或用户管理的 label；
- 打开/重开时恢复 `status:needs-triage`；
- label 应用时规范 `type`、`priority`、`status`、`resolution` 单值轴；
- 关闭时移除开放状态，并在没有更具体 resolution 时按 GitHub close reason 补充默认 resolution；
- 重开时移除 resolution。

workflow 不自动关闭 Issue、不指定 owner、不编造关闭理由或验证证据。远程启用后的首次 label 同步和实际 Issue 事件仍需在 GitHub Actions 中观察确认。

## 本地验证

```powershell
bin\aicoding.exe governance lint --json
go test ./internal/governance
```

## 发布边界

```text
AiCoding repository policy
-> Go governance lint/tests
-> GitHub Issue Forms + label workflow
-> AiCoding commit, tag and release
```

`CodingKit/agents/skills` 仍保持只读；本仓库 Issue policy 不要求修改或重新生成该 submodule。若未来将本策略提升为可复用 canonical Skill 能力，必须在 Codex-Skills 独立实现、验证、发布后再更新 gitlink。
