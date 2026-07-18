# 02 Context/Memory 体系（Context Architecture）

Status: Derived View（派生视图；§6 为已确认方向的概述）

> 本文不定义新契约；知识面的权威定义见 [架构手册](ARCHITECTURE_HANDBOOK.md) §7，
> 冲突时以其为准。§6 的 repo-context 是 [07 演进路线](07-roadmap.md) 已立项方向的概述。

## 本篇回答的问题

- Agent 如何理解项目？
- Context 如何组织？
- Memory 如何管理？
- 如何降低 Token 消耗？

## 1. Context 的组织：四个知识进入点

Agent 在本仓库获取"该做什么、怎么做、错了怎么纠正"只有四个入口，
**功能可以无限增长，入口数量保持恒定**：

| # | 进入点 | 承载内容 | 权威位置 |
|---|---|---|---|
| ① | `AGENTS.md` | 仓库总纪律：会话开始必读的边界 + 指向其余知识的索引 | 仓库根目录 |
| ② | SKILL.md 集合 | 多步工作流知识：按 frontmatter description 匹配意图后才加载 | 运行时 Skill 根（经 lifecycle 暴露，见 [03](03-skill-architecture.md)） |
| ③ | CLI 自描述 | 命令知识：`aicoding help` 输出 + JSON 报告的 `schemaVersion`/`ok`/`errorKind` | `internal/cli` catalog 与 `internal/report` |
| ④ | 门禁错误信息 | 纠错知识：失败信息必须指明违反的规则与正确路径 | 各治理/审计检查的 error 文案 |

四个入口互不复制内容：`AGENTS.md` 不展开工作流细节（指向 Skill）；SKILL.md 不复制
命令参考（指向 CLI help）；错误信息不重述完整政策（指向文档）。**重复即漂移的开端。**

## 2. Agent 理解项目的完整路径（具体走一遍）

1. 会话开始：读 `AGENTS.md`，拿到边界（什么不能碰）和索引（细节在哪）。
2. 需要仓库地图：看 `README.md` / `docs/README.md` 里的 REPOSITORY_MAP 生成区块
   （"哪个目录干什么、常见任务从哪进"）。
3. 接到多步任务：意图命中某份 SKILL.md（如升级列车、环境重建），按其步骤走。
4. 执行动作：只调 `aicoding` 命令，读 JSON 报告的 `ok`/`errorKind` 判断成败，
   不解析人类文本。
5. 犯了规：门禁把提交拦下，错误文案直接告诉它违反了哪条规则、正确路径是什么。

## 3. Memory 如何管理

| 记忆类型 | 位置 | 形式 |
|---|---|---|
| 决策记忆（选了什么、为什么） | `.aicoding/memory/DECISIONS.md` | 追加式决策日志，每条带 `Decision Status` |
| 计划记忆（当时怎么规划的） | `docs/decisions/<topic>/`、`docs/spec/` | 计划工件（PRD_OPTIONS、SELECTED_SOLUTION、IMPLEMENTATION_PLAN、TASKS…） |
| 事实记忆（代码何时为何而变） | Git 历史本身 | commit / tag / release |
| 运行证据（哪次跑了什么、结果如何） | `test-results/`、JSON 报告 | 带 digest 的报告文件 |

铁律：**会话中的口头约定不属于记忆**——未持久化到上述位置的知识，对下一次
Agent 会话不存在。

## 4. Repository Index 现状与差距

现状：`config/repository-navigation.json` 是导航配置的唯一源头，生成器把它渲染成
`README.md` 与 `docs/README.md` 中 `AICODING:REPOSITORY_MAP` 标记区（只是导航
生成器，不是第二个运行时门禁）。

差距（这正是 [07](07-roadmap.md) repo-context 立项的现实问题）：

- 手工维护：目录用途变了要人记得去改配置；
- 粒度粗：只到目录级，Agent 进到具体域仍要自己摸；
- 不随提交更新：代码演进后索引可能悄悄过时。

## 5. 如何降低 Token

| 策略 | 做法 | 现状 |
|---|---|---|
| 入口恒定，不堆单体 | 拒绝数千行的单体 AGENTS.md/CLAUDE.md（已知反模式）；`AGENTS.md` 只留纪律与索引 | 已执行 |
| 按需加载 | SKILL.md 靠 frontmatter 意图匹配才进上下文；没触发就是零成本 | 已执行 |
| 机器判读优先 | Agent 读 `ok`/`errorKind`/退出码三个字段，而不是读长报告；命令知识在 `help` 里而不在文档里 | 已执行 |
| 小粒度 scoped context | 每个域一份约 35 行的上下文文件，只在 Agent 碰到该域时激活（`aspenkit/aspens` 已验证的做法） | 未来：repo-context 阶段 2（见 [07](07-roadmap.md) §3） |

## 6. repo-context：上下文层的下一步（概述）

目标：把"Repository Index"从手工配置升级为**从代码自动生成、随提交自动更新**的
受管资产——扫描仓库（目录、语言、依赖图）得到事实快照（snapshot + digest），
生成每域约 35 行的 scoped context 文件，commit 后只增量更新受影响的部分，
`doctor`/`verify` 对账新鲜度（代码 digest vs 生成物 digest，漂移即报）。

它是 lifecycle 受管资产（八动词复用），**绝不覆盖用户手写内容**。
分阶段开发计划、验收门禁与"不并入外部 CLI"的边界见 [07 演进路线](07-roadmap.md) §3。
