# Selected Solution: Dependency Direction And Stable Identity Governance

Plan Status: Approved

用户选择将“下层不得依赖或感知上层”提升为 AiCoding 仓库级治理，并追加“资产稳定身份与运行代码不得观察自身版本”的规则。

Selected architecture:

```text
platform -> integration -> capability -> runtime
```

AiCoding 作为 composition root 通过 registry/manifest 组合下层能力。通用 Kit、standalone Skill、MCP 和模块保持平台无关；产品 namespace、安装状态与 lifecycle 只存在于上层。

版本只在 manifest metadata、资产文档、CHANGELOG、Tag/Release URL 和 README badge 权威面可见。
