# 任务：AiCoding 内核与扩展图架构

## 架构基线

- [x] 审计仓库分层、命令、registry、adapter、runner 和 report。
- [x] 对照 Git、GitHub 与 mattpocock/skills。
- [x] 建立编译后二进制性能基线。
- [x] 记录可选路线与用户选择。
- [x] 写入权威架构、入口和维护边界。

## 代码迁移

- [x] 建立 `ExecutionPlan` descriptor、不可变选择、snapshot 和 digest，并迁移真实消费者。
- [x] 建立 Registry Snapshot + Digest，并替换 Kit/MCP registry loader。
- [x] 建立 typed command catalog，接管顶层 handler、namespace contract 与全局 help。
- [x] 用单元/contract 测试冻结以上对象与现有 CLI 行为。
- [ ] 把性能基线固化为 benchmark gate。
- [ ] 统一 root/path 和 manifest snapshot。
- [ ] 建立 capability graph 与静态 adapter factory。
- [ ] 建立 read/write execution 和 state/journal。
- [ ] 让 typed command catalog 校验 Taskfile、命令文档和兼容命令。
- [ ] 统一内部 check/result。
- [ ] 接入 digest cache 和性能门禁。
- [ ] 删除重复实现并完成 Full/Release。
