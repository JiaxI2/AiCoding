# 实施计划：AiCoding 内核与扩展图架构

Plan Status: Approved

## 架构确定

- 建立 `docs/architecture/AICODING_CORE_ARCHITECTURE.md` 为权威架构。
- 更新总览、维护入口和 Agent 必读文档。
- 记录选项、用户选择、性能基线和拒绝项。

## 已实现的第一批核心对象

1. `ExecutionPlan`：稳定 descriptor、不可变选择、snapshot、digest，并迁移 pre-commit 真实消费者。
2. Registry Snapshot + Digest：建立通用 snapshot 对象，迁移 Kit/MCP loader，MCP inventory 暴露 digest。
3. Typed Command Catalog：用 typed ID 绑定 handler、alias、namespace 和 help，并删除 CLI 顶层 switch 与手写 help。

## 后续代码迁移

1. 统一 root/path 与 manifest snapshot。
2. 引入 capability graph 和静态 adapter factory。
3. 为 plan 增加 read/write 与 journal 语义。
4. 用 typed command catalog 校验 Taskfile、命令文档和兼容命令。
5. 统一内部 result/check，并保留外部兼容。
6. 接入 digest cache 与 benchmark gate。
7. 删除旧路径并执行 Full/Release。

## 实施纪律

- 每个提交迁移真实消费者并删除对应旧路径。
- 不先搭空框架，不保留长期双轨。
- 不修改子模块源、生成插件或 plugin cache。
- 不 push、PR 或 release，除非用户另行授权。
