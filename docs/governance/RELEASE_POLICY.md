# AiCoding Release Policy

AiCoding 当前 release 分为三条 lane：平台主 release、kit/component release、milestone release。Release 文档只描述当前命名空间和当前门禁。

## Platform Release

平台主 release 使用：

```text
vMAJOR.MINOR.PATCH
```

适用场景：

- 仓库整体架构升级。
- Go CLI 默认控制面行为变化。
- Kit lifecycle 默认语义变化。
- Release gate、install、setup 或 Taskfile 路由入口变化。
- 影响用户 clone 后整体使用方式的变更。

## Kit / Component Release

组件 release 使用：

```text
kit/<kit-id>/vMAJOR.MINOR.PATCH
```

适用场景：

- 单个 kit 的脚本、skill、hook 或模板升级。
- 外部工具包独立升级。
- 不改变 AiCoding 平台整体 release 语义的组件交付。

## Milestone Release

Milestone 使用：

```text
milestone/YYYY.MM.DD-<name>
```

适用场景：

- 方案冻结。
- 架构快照。
- MVP 或阶段性基线。
- 不适合表达为平台语义化版本的状态标记。

## Release Notes 模板

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

## Upgrade / Rollback

- Upgrade:
- Rollback:

## Tag namespace

- Tag: `<tag>`
- Namespace: platform / kit / milestone
```

## 不允许的动作

- 不得把 kit/component 版本打成平台 `v*` tag。
- 不得删除或覆盖 immutable release tag。
- 不得 force push tag。
- 不得在未确认的情况下 push tag 对齐结果。
- 不得让 Full / Release gate 隐式触发硬件动作。
- 不得执行 TI DSS / XDS / reset / halt / run / flash / erase / write-memory 作为 release 治理步骤。