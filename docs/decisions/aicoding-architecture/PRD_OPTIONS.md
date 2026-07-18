# AiCoding 架构路线选项

Decision Status: Selected and refined by implementation evidence

## 目标

在保持单 Go CLI、依赖方向和 source/distribution/runtime 边界的前提下，提高扩展性、
可玩性、性能与局部可验证性，并避免 God Core 或平行控制面。

## 方案 A：正交深模块 + 静态扩展（已选择）

- snapshot、plan、runner、adapter、report、domain state 按变化原因分离。
- Kit/MCP/runtime Skill 用领域 manifest 与静态 adapter 扩展。
- Agent/Skill/Hook/CI 通过唯一 CLI/JSON 调用。
- Registry/manifest 形成内容树；plan/input/result 都可追踪。
- 新组件优先修改领域模块和契约测试，不扩大 Core。

这是早期“稳定内核 + 扩展图”方案的实证收敛：保留稳定 plumbing 与静态扩展，但删除
没有真实依赖数据的 capability graph 和没有共同事务语义的 global journal。

## 方案 B：中央能力图 + 全域生命周期引擎（拒绝）

- 把所有 provides/requires、状态、事务和 rollback 放入一个 control core。
- 表面统一，但要求 Core 理解 Kit/MCP/Skill 业务，形成 God Core。
- 当前没有第二个稳定消费者证明 capability graph，也没有可诚实实现的跨域原子事务。

## 方案 C：只修文档与重复 loader（拒绝）

- 不建立可摘要的 catalog、adapter contract 与 lifecycle plan。
- 变更小，但 Agent 无法确认输入/意图，新增领域继续修改 scope switch。
- 不能满足闭环、可维护与局部测试目标。

## 方案 D：动态 plugin 或远程微内核（拒绝）

- 第三方代码动态进程内加载，或预建 HTTP/gRPC/MCP control service。
- 增加 Windows ABI、供应链、授权、调试和第二控制面复杂度。
- 当前本地 Agent/Skill/CI 都能由 process + JSON 满足，不存在第二 transport consumer。

## 方案 E：C/native 基础层（暂不采用）

- 只有 profile 证明纯计算热点、Go 优化不足、稳定 ABI 有两个消费者且 fallback/golden tests
  完整时才成立。
- 当前主要成本是进程、文件、JSON 和外部 runtime；native Core 不会解决真实瓶颈。

## 选择依据

Git 的先进性来自正交对象/ref/index/transport 与稳定 plumbing，而不是一个理解所有命令的
中心对象。用户新增的“稳定边界优先于无限优化”和 Orthogonal Architecture Design Kit
进一步要求：状态归领域、修改影响半径可局部验证、架构在闭环后停止。因此选择方案 A。
