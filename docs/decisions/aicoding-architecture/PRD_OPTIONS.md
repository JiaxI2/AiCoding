# AiCoding 架构路线选项

Decision Status: Selected

## 目标

在保持单 Go CLI、现有依赖方向和 source/distribution/runtime 边界的前提下，
同时提升可扩展性、可玩性和性能，并避免再造平行控制面。

## 方案 A：稳定内核 + 扩展图（已选择）

- 把 root/path、manifest snapshot、capability graph、plan/runner、report、
  state/journal 定为最小稳定 plumbing。
- Kit、MCP、runtime Skill、checks 和工具用声明式 descriptor + 静态 adapter 扩展。
- 用户 CLI、Skills、profiles、hooks 和 CI 作为 porcelain 组合基础能力。
- 保留单二进制与静态链接；外部工具使用有界子进程。
- 适合当前三个真实 lifecycle adapter，能渐进迁移且不牺牲启动性能。

## 方案 B：只修重复路径与文档

- 修正 Plan Mode root/path、合并部分 registry loader，不建立 capability graph。
- 风险最低，但每新增领域仍需修改 CLI、lifecycle switch 和多份 contract。
- 无法解决长期扩展成本，拒绝作为长期架构。

## 方案 C：动态插件微内核

- 第三方能力以动态 Go plugin 或任意进程内模块加载。
- 表面可玩性最高，但 Windows ABI、供应链、安全、调试和缓存复杂度显著增加。
- 与单二进制、高性能、受治理分发和当前真实需求冲突，拒绝。

## 用户选择依据

用户明确要求以 Git 为参照：基础功能必须足够稳定，扩展功能建立在基础能力之上；
同时要求实现路径和稳定 identity 不出现版本信息，并尽量一次确定长期方案。因此选择方案 A。
