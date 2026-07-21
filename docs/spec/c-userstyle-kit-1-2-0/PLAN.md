---
id: c-userstyle-kit-1-2-0
status: archived
scope:
  - CodingKit/tools/c-userstyle-kit/**
  - internal/cstyle/**
  - config/kits/c-userstyle-kit.json
  - docs/guides/C99_STANDARD_C_SKILL.md
approvedTree: ""
decision: docs/spec/c-userstyle-kit-1-2-0/DECISION.md
gates:
  - profile: full
  - profile: release
---

# 计划模式会话：C UserStyle Kit 1.2.0 集成

Mode: Plan
Plan Status: Approved
Created: 2026-07-15

## 需求

把当前 C Kit 集成进 AiCoding 的既有架构，完成本地快速功能验证，并在全部门禁通过后发布平台版本。

## 已确认约束

- C Kit 属于 `CodingKit/tools` 外部确定性资产，不进入 skills submodule、生成插件或插件缓存。
- 用户入口继续使用 `aicoding skill c99-standard-c`，不新增平行顶层命令。
- C Kit 保持自包含 Go module；AiCoding Go 控制面只提供有界适配和标准 JSON 报告。
- 用户已明确授权 PDF、规范化 Markdown 与 raw 转换件随公开仓库和 Release 发布。
- registry、CLI、Smoke/Release 行为发生用户可见变化，因此平台版本采用 `v0.8.0`。

## 成功标准

1. fresh clone 中不依赖个人绝对路径即可定位 C Kit 1.2.0。
2. `skill c99-standard-c verify --profile fast --json` 执行真实 lint、GCC C99 和 host test。
3. Kit registry、生命周期计划、全局测试器与文档只有一个 C99 用户入口。
4. skills submodule 保持 clean，插件与缓存零修改。
5. Smoke、Full、Release、DocSync、治理、Hook 和 Git 门禁全部通过后才推送。

未解决的 `[NEEDS CLARIFICATION]`：无。
