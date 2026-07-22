---
id: capability-usage-closure
status: approved
scope:
  - config/internal-capabilities.json
  - config/schemas/internal-capabilities.schema.json
  - internal/capability/**
  - internal/cli/capability_test.go
  - internal/docsync/**
  - internal/testengine/**
  - docs/architecture/LOOP_ENGINEERING_ARCHITECTURE.md
  - docs/CAPABILITIES.md
  - docs/COMMANDS.md
  - README.md
  - CHANGELOG.md
  - docs/todolist/0027-capability-usage-closure.md
approvedTree: "46ed2ff852a078beedebd23b34df8ac70afa0a57"
gates:
  - profile: full
---

# 能力使用闭环计划

## 目标

在不新增注册表、命令域或激活流程的前提下，让有公共入口的 capability 从同一 registry
投影出 quickstart 与 activation，并让 `capability describe`、README 生成区、能力索引和
Loop Engineering 架构图共同回答“是什么、怎么用、怎么进 Agent、怎么验证、当前状态”。

## 边界

- `quickstart` 与 `activation` 仅作为现有 capability registry 的可选字段，保持旧条目可解析。
- CLI-entry 能力直接由 Agent 调现有 typed command，不要求 install；`kit-install` 只保留为
  有界 activation 类型，不把 Loop Engineering 错写成待安装能力。
- README 与 `docs/CAPABILITIES.md` 继续由 `capability index --write` 唯一生成。
- DOCS-006 扩展到 Loop Engineering 的两张 Mermaid，但每图仍不超过 20 节点，图中命令仍
  必须来自 typed command catalog。

## 验证

完成前运行 capability/governance/docsync/testengine 局部测试、三条真实负例、按 describe
quickstart 执行 `work validate/next`，再运行 Full profile、链接检查与 `git diff --check`。
