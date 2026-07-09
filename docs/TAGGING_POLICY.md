# AiCoding Tagging Policy

本文件定义当前 main 的 Tag 命名空间。当前标准只保留三类可发布 tag：平台、kit/component、milestone。非当前命名只作为审计结果处理，不在本文档中维护额外清单。

## 平台主版本 Tag

平台主版本只允许使用标准语义化版本：

```text
vMAJOR.MINOR.PATCH
```

平台主版本表示 AiCoding 平台整体 release，不用于表示单个 kit、skill、hook 或 milestone。

## Kit / Component Tag

Kit / component 版本必须进入 `kit/` 命名空间：

```text
kit/<kit-id>/vMAJOR.MINOR.PATCH
```

其中 `<kit-id>` 使用 manifest 或 registry 中的当前 id。

## Milestone Tag

Milestone 使用日期命名空间：

```text
milestone/YYYY.MM.DD-<name>
```

Milestone 只用于方案冻结、架构快照和阶段性基线，不替代平台语义化版本。

## 发布前检查

```powershell
bin\aicoding.exe tag audit --json
bin\aicoding.exe release verify --json
bin\aicoding.exe release gate --json
```

专项 tag 对齐计划仅在需要人工审计 tag 命名时运行：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Audit -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts\aicoding-tag-governance.ps1 -Action Plan
```

## 不允许的动作

- 不把 kit/component 版本放入平台 `v*` 命名空间。
- 不把日期式 tag 作为平台 release tag。
- 不删除、覆盖或复用已经绑定 release 的 immutable tag。
- 不 force push tag。
- 不在未确认的情况下 push 对齐 tag。