# 03 Skill 生命周期与组织（Skill Architecture）

Status: Derived View（派生视图）

> 本文不定义新契约；权威边界见 `AGENTS.md`（Runtime Skill Exposure Policy、
> External GitHub Skill Policy）、[CODEX_KIT_ARCHITECTURE](CODEX_KIT_ARCHITECTURE.md)
> 与 [架构手册](ARCHITECTURE_HANDBOOK.md) §7.3，冲突时以其为准。

## 本篇回答的问题

- Skill 如何调用（怎么被触发、怎么执行）？
- 新增一个 Skill 应该放在哪一层？
- Skill 的来源、所有权和生命周期是怎么管的？

## 1. Skill 是什么

一份 `SKILL.md`：frontmatter 声明名字、description（触发条件）与适用/不适用场景，
正文是多步工作流的步骤、验证命令和安全边界。**Skill 承载知识，不承载执行权**——
所有动作转调 `aicoding` 命令或标准工具，结果按 JSON 契约判读。

## 2. 四类来源与所有权

| 类别 | 位置 | 命名 | 所有权 |
|---|---|---|---|
| Plugin 内嵌平台 Skill | 经 AiCoding plugin 安装（源码在 `CodingKit/agents/skills` 子模块） | `aicoding-*`（依赖平台行为才允许用此前缀） | 上游 Codex-Skills 仓库；本仓库只读 |
| 独立标准 Skill | 同一子模块的 canonical 源 | 平台无关域名（不得用 `aicoding-*`） | 上游 Codex-Skills 仓库 |
| 外部 Skill | 上游 GitHub 仓库，经 Codex-Skills `external/` 嵌套子模块 pin 进来 | 保留其域名 | 外部作者；本仓库只登记映射，**不复制源码** |
| RepoLocal 用户 Skill | `.agents/skills/`（本仓库版本管理） | 平台无关域名 | 本仓库 |

关键边界：本仓库**不拥有任何嵌入式 Skill 源码**；`CodingKit/agents/skills`
是只读发布依赖，一切修改到上游提交后更新 pin。

## 3. 调用机制（Skill 如何被触发）

1. **暴露**：`lifecycle install|update --scope runtime-skill --runtime-profile
   runtime|full|skill-development` 把选定 Skill 集合接到 Agent 运行时的 skill 根；
   `runtime` 是常规模式（只暴露 plugin 内嵌 Skill），`full` 增加登记的独立/外部
   Skill，`skill-development` 供上游开发调试。
2. **触发**：Agent 运行时按 frontmatter description 匹配用户意图，命中才把该
   SKILL.md 载入上下文（未命中零 Token 成本）。
3. **执行**：Skill 步骤逐条转调 CLI；每步用 `ok`/`errorKind` 判定，失败按 Skill
   声明的回滚/停止规则处理。
4. **审计**：同名 active Skill 会被运行时审计拒绝——任一工作流主题只有一份生效
   知识（`lifecycle status --scope runtime-skill` 可盘点当前暴露了什么、来自哪里）。

## 4. 成熟度阶梯：Draft → RepoLocal → Kit

```text
Draft（.aicoding/user-skills/）     个人试用；不进版本管理的正式运行时
  → RepoLocal（.agents/skills/）    团队复用；进版本管理，Agent 可发现
    → Kit（上游 Codex-Skills 收编）  正式资产；随 plugin 分发
```

- **准入门禁**：进入任何正式阶段前，`aicoding-skill.ps1 verify` 校验 frontmatter、
  when-to-use / when-not-to-use、验证命令与安全边界完整性；不合格不得进入运行时。
- **现存实例**：`aicoding-upgrade-train`（升级列车）与 `aicoding-environment-rebuild`
  （环境重建）均经门禁后进入 RepoLocal；收编进 Kit 留待真实使用反馈。
- 任一阶段可卸载；知识是有身份、可验证、可卸载的资产，不是散落的经验。

## 5. 新增一个 Skill 应该放在哪一层（判定流程）

依次问：

1. **它是多步工作流知识吗？** 否 → 不要写 Skill：命令知识进 CLI help，
   政策进治理文档，单步操作直接用命令。
2. **它依赖 AiCoding 平台行为吗？** 是 → plugin 内嵌 `aicoding-*` Skill
   （上游源码仓库开发）；否 → 平台无关域名。
3. **谁用？** 只有自己试用 → Draft；本仓库团队复用 → RepoLocal；
   跨仓库/正式分发 → 走上游收编成 Kit。
4. **来源是外部 GitHub 吗？** 是 → 必须走 `external/` 子模块 pin + binding 登记
   （[06](06-plugin-sdk.md) 路径②），永不复制源码进仓库。
5. **同名冲突？** 先查运行时审计；一个名字只能有一个 active 来源。

## 6. 生命周期动词复用

Skill 资产与 kit、MCP 一样用同一套动词管理：`plan / install / update / status /
doctor / verify / uninstall`（`--scope runtime-skill`）。知识本身按包管理，
不设第二套流程——这就是"知识即资产"的执行形式。
