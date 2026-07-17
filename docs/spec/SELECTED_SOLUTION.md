# 已选择方案（Selected Solution）：product-convergence

Decision Status: Selected

Selected option: 方案 A，兼容优先的统一控制面。

## 选择理由

- `test --profile Smoke|Full|Release` 成为唯一正式测试入口。
- `lifecycle` 成为唯一正式产品生命周期命名空间。
- `release gate` 复用同一个测试引擎，不再通过 CLI 递归聚合。
- 旧命令保留一个版本，统一输出 `CLI_DEPRECATED`，保护现有 CI。
- 内聚现有 tester、runner 和 report，不新增另一套框架。
- kit、MCP 和 runtime Skill 使用静态 adapter 组合，不引入复杂插件系统。

## 正式入口

```text
aicoding bootstrap
aicoding lifecycle ...
aicoding doctor --all
aicoding verify --profile Smoke|Full|Release
aicoding test --profile Smoke|Full|Release
aicoding release verify|gate
```

## 兼容边界

以下入口兼容一个版本：

```text
smoke
ci
full
test full|release
kit lifecycle
mcp install|update|uninstall
status --all
```

兼容入口必须路由到正式实现，并通过 `report.Result.warnings` 和文本提示输出
`CLI_DEPRECATED: use <canonical command>`。

## 不变约束

- 不修改 `CodingKit/agents/skills`。
- 不新增另一套 CLI、测试框架、Report 或 UI。
- 不自动创建 Release、Tag、PR 或合并。
- 不把所有 PowerShell/Python 改写为 Go。
- 每个 Phase 单独验证和提交，不 squash 分阶段提交。
