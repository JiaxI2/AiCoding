# 已选方案：稳定内核 + 扩展图

Decision Status: Selected

Selected option: 以稳定 plumbing 内核作为唯一基础，通过声明式 capability graph、
静态 adapter 和 porcelain 工作流扩展 AiCoding。

## 决策约束

- 实现目录、文件名、包名、模块名、服务名和稳定 ID 不编码版本。
- README、CHANGELOG、Release、manifest 元数据和说明文档可以记录版本。
- 单 Go 二进制、单 lifecycle、单 test engine、单 report authority 保持不变。
- 不使用 Go 动态插件，不引入微服务。
- 扩展只能依赖同层或更低层，内核不观察具体产品 workflow。
- 所有写操作必须可 plan、可审计，并只回滚自身拥有的状态。
- Codex-Skills source、Marketplace package、installed state、runtime exposure 保持分离。
- 实施分提交进行，但所有提交服务于同一架构，不建立长期双轨。

## 决策证据

- 当前 `internal/runner` 已提供可复用的有界并发与稳定输出。
- 当前 Kit/MCP/runtime Skill 已形成三个真实 adapter，足以证明统一扩展 seam。
- 当前主要债务集中在路径、registry、命令 contract 与报告语义重复，而非缺少框架。
- 编译后二进制的轻路径已在百毫秒内；静态链接应保留。

## 第一批实现证据

- pre-commit 已从 mutable `Plan` 迁移到可摘要的 `ExecutionPlan`。
- Kit/MCP registry loader 已共用 snapshot/digest primitive，不再各自定义摘要算法。
- CLI 顶层路由、alias、namespace help 和全局 help 已由 typed command catalog 统一描述。
- 三类对象都使用稳定 ID 和 SHA-256 digest，不把函数地址、绝对路径或实现代际写入 identity。
