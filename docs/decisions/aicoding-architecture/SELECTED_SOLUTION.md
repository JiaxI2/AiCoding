# 已选方案：正交深模块 + 仓库控制面 + 静态扩展

Decision Status: Selected and Implemented

## 选择

AiCoding 以 snapshot、plan、runner、adapter、report 和 domain-owned state 六个正交职责为
稳定基础；Go CLI 是仓库唯一产品控制面；Kit、MCP、runtime Skill 通过静态 adapter 和
领域 manifest 扩展。

该选择替代早期“把 capability graph、全域 journal 都放入稳定内核”的设想。当前 manifest
没有两个真实消费者需要通用 provides/requires graph，三个领域也没有共同的原子事务语义，
因此把它们提前放入 Core 会违反稳定边界优先与 No God Core 原则。

## 不变量

- 实现 identity 不编码版本；文档、manifest、Tag/Release 可记录版本。
- 单 Go CLI、单 lifecycle、单 test engine、单 report authority。
- 模块按变化原因分离，通过 immutable values/snapshots/results 连接。
- runner 不理解领域，adapter 不拥有业务策略，state 不进入 global core。
- 所有 write action 可先 plan，并只修改领域登记资产。
- source pin、package、installed state、runtime exposure/discovery 保持分离。
- 不使用 Go dynamic plugin、不预建第二 transport API、不引入无 profile 证据的 C core。

## 实现证据

- `ExecutionPlan` 被 pre-commit 与 lifecycle 两个真实消费者使用。
- `CatalogSnapshot` 将 registry 与 referenced manifests 组合为内容树，Kit/MCP 复用。
- lifecycle 静态 catalog 登记 Kit/MCP/runtime Skill 的 input/state owner/entrypoint/effect。
- lifecycle 顶层 scope switch 已删除，选择结果生成 plan 并串行调度。
- JSON 返回 adapter catalog、domain input 和 plan digest。
- Typed Command Catalog 继续统一 CLI routing/help，与 adapter catalog 保持正交。

## 停止决定

架构闭环满足后冻结。新增 component/Skill 是功能扩展；模块内部性能或错误处理是维护。
只有真实问题、稳定变化点和至少两个消费者同时出现时，才以 ADR 解冻对应模块契约。
