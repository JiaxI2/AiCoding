# AiCoding Tagging Policy

本文件定义 AiCoding 仓库的 Tag 命名空间。目标是避免平台主版本、Kit / 组件版本和历史里程碑版本混在同一个 `v*` 序列里。

## 1. 平台主版本 Tag

平台主版本只允许使用标准语义化版本：

```text
vMAJOR.MINOR.PATCH
```

示例：

```text
v0.1.0
v0.1.1
v0.1.2
v0.2.0
```

平台主版本表示 AiCoding 平台整体 release，不用于表示某个独立 kit、skill、hook 或历史里程碑。

`vYYYY.MM.DD` 这类裸日期 tag 不是平台主版本；历史上已存在的裸日期 tag 只能作为 legacy historical snapshot 处理。

## 2. Kit / 组件版本 Tag

Kit / 组件版本必须进入 `kit/` 命名空间：

```text
kit/<kit-id>/vMAJOR.MINOR.PATCH
```

示例：

```text
kit/powershell-skill-kit/v1.3.0
kit/system/v2.0.0
kit/agent-dev-kit/v0.11.1
kit/common-control/v0.1.0
```

不要再使用下面这类伪平台版本格式：

```text
v1.3.0-powershell-skill-kit
v2.0.0-kit-system
v0.11.1-agent-dev-kit
```

## 3. 日期里程碑 / 历史快照 Tag

日期里程碑必须进入 `milestone/` 命名空间：

```text
milestone/YYYY.MM.DD-<name>
```

示例：

```text
milestone/2026.07.03-fast-path-v1
milestone/2026.07.03-skill-external-mvp
```

不要再使用：

```text
v2026.07.03-fast-path-v1
v2026.07.03-skill-external-mvp
v2026.06.27
v2026.06.26
```

## 4. Legacy Tag 处理规则

历史上已经创建并绑定 release 的 tag 可能是 immutable，不得假设可以删除或覆盖。

处理原则：

1. 不强制删除 remote tag。
2. 不 force retag。
3. 不复用 immutable release tag。
4. 保留旧 tag，并在 release notes 中标记为 `Legacy component tag` 或 `Historical milestone tag`。
5. 如需要纠偏，创建新的命名空间 tag 指向原 commit。
6. 新 tag push 前必须输出计划并等待确认。

## 5. 推荐纠偏映射

| Legacy tag | 推荐新 tag | 类型 |
|---|---|---|
| `v1.3.0-powershell-skill-kit` | `kit/powershell-skill-kit/v1.3.0` | Kit / component |
| `v2.0.0-kit-system` | `kit/system/v2.0.0` | Kit / component |
| `v0.11.1-agent-dev-kit` | `kit/agent-dev-kit/v0.11.1` | Kit / component |
| `v2026.07.03-fast-path-v1` | `milestone/2026.07.03-fast-path-v1` | Milestone |
| `v2026.07.03-skill-external-mvp` | `milestone/2026.07.03-skill-external-mvp` | Milestone |
| `v2026.06.27` | `milestone/2026.06.27-platform-snapshot` | Historical snapshot |
| `v2026.06.26` | `milestone/2026.06.26-platform-snapshot` | Historical snapshot |

## 6. 发布前检查

发布前至少运行：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Audit -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Plan
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\verify-release-governance-overlay.ps1 -Json
```

如果已安装 Task：

```powershell
task tag:audit
task tag:plan
task tag:verify
```
