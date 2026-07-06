# AiCoding Release Policy

AiCoding 仓库 release 分为三条 lane：平台主 release、Kit / 组件 release、历史里程碑 release。

## 1. Platform Release

平台主 release 使用：

```text
vMAJOR.MINOR.PATCH
```

适用场景：

- 仓库整体架构升级
- Fast Path 行为变化
- Kit lifecycle 默认语义变化
- Release gate / install / setup 入口变化
- 影响用户 clone 后整体使用方式的变更

示例标题：

```text
AiCoding v0.1.2: release namespace policy and task entry
```

`vYYYY.MM.DD` 和 `vYYYY.MM.DD-<name>` 是历史日期式命名，不再作为平台主 release tag。

## 2. Kit / Component Release

组件 release 使用：

```text
kit/<kit-id>/vMAJOR.MINOR.PATCH
```

适用场景：

- 单个 kit 的脚本、skill、hook 或模板升级
- PowerShell Skill Kit 单独升级
- Agent Dev Kit 单独升级
- Common Control Kit 单独升级

示例标题：

```text
PowerShell Skill Kit v1.3.0
```

## 3. Milestone / Historical Snapshot

历史里程碑使用：

```text
milestone/YYYY.MM.DD-<name>
```

适用场景：

- 方案冻结
- 架构快照
- MVP 标记
- 不适合表达为平台语义化版本的历史状态

## 4. Release Notes 模板

```markdown
# <Release title>

## Scope

- Platform / Kit / Milestone:
- Affected paths:
- Compatibility:

## What changed

- ...

## Validation

```powershell
# commands here
```

## Migration / Rollback

- Migration:
- Rollback:

## Tag namespace

- Tag: `<tag>`
- Namespace: platform / kit / milestone
```

## 5. 不允许的动作

- 不得把 kit/component 版本打成平台 `v*` tag。
- 不得删除或覆盖 immutable release tag。
- 不得 force push tag。
- 不得在未确认的情况下 push 纠偏 tag。
- 不得让 Full / Release 默认隐式触发。
- 不得执行 TI DSS / XDS / reset / halt / run / flash / erase / write-memory 作为 release 治理步骤。

## 6. 建议的下一次平台版本

如果本次只修复 release namespace、Taskfile 入口和性能闭环文档，建议平台版本继续使用：

```text
v0.1.2
```

不要使用：

```text
v1.3.1
v2.0.1
```

这些版本号应保留给 kit/component 命名空间。
