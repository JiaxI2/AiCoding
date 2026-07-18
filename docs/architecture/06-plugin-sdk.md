# 06 扩展规范：Skill / MCP / CLI（Plugin SDK）

Status: Derived View（派生视图）

> 本文不定义新契约；adapter 契约见
> [EXTENSION_ADAPTER_CONTRACT](EXTENSION_ADAPTER_CONTRACT.md)，
> 拒绝清单见 [核心架构](AICODING_CORE_ARCHITECTURE.md) §11，冲突时以其为准。

## 本篇回答的问题

- 新能力从哪条路进来？各条路的成本是多少？
- 核心与插件的机制边界在哪里（什么时候必须走 ADR）？
- 一个扩展做到什么程度才算"完成"？

## 1. 总原则

从便宜到昂贵四条路径，**95% 的需求应停在路径①②**。遇到看似要走③④的需求，
先对抗性追问：它真的不能表达为现有领域的 manifest 变体吗？

```text
① 新 Kit / MCP component     登记即扩展（不碰内核，不碰 CLI）
② 新 external Skill          跨仓库 pin + 登记（不复制源码）
③ 新领域 adapter             新领域模块 + catalog 一行（需 ADR）
④ 内核契约修改               ADR + 三条件（极少发生）
```

## 2. 路径①：新 Kit / MCP component（成本最低）

具体步骤：

1. 写 manifest：`config/kits/<id>.json` 或 `config/mcp/components/<id>.json`，
   通过对应的冻结 schema 校验（五个冻结 schema 之一，见
   [FREEZE_AND_ACQUISITION_BOUNDARY](FREEZE_AND_ACQUISITION_BOUNDARY.md)）。
2. 登记 registry：`config/kit-registry.json` / `config/mcp-registry.json` 增一条。
3. 预览并安装：`lifecycle plan --action install --scope kit|mcp …` →
   `lifecycle install …`。
4. 过门禁：`verify --profile Smoke` + `test --profile Smoke`（按风险升 Full）。

效果：内核与 CLI **零改动**；新组件自动出现在 `list`/`status` JSON 里
（数据变化，不是接口变化）；Agent 无需学任何新东西。

命名约束：可复用能力用平台无关域名；只有真正依赖 AiCoding 行为的资产才允许
`aicoding-*` 前缀；身份里永不编码版本号。

## 3. 路径②：新 external Skill（跨仓库登记）

1. 上游仓库进入 Codex-Skills 的 `external/` **嵌套子模块**并 pin 到发布版本
   （不复制源码；Codex-Skills 侧登记 binding manifest）。
2. AiCoding 侧把运行时名字登记进 full profile 的独立 Skill 清单，并把名字映射到
   含 `SKILL.md` 的具体目录（sourcePaths）。
3. `lifecycle install|update --scope runtime-skill --runtime-profile full …` 暴露；
   运行时审计拒绝同名 active Skill。
4. 升级：解析上游最高稳定 semver tag，评审后前移 pin；卸载只删除目标精确匹配的
   junction；解除集成必须同时清掉两侧登记，不留孤儿链接。

## 4. 路径③：新领域 adapter（需 ADR）

先例：runtime-skill 域进入时**内核六模块零修改**。步骤：

1. ADR 论证：为什么不能表达为现有领域的 manifest 变体。
2. 新领域模块（Go）实现领域业务与自己的 state/rollback 规则。
3. `internal/lifecycle` 静态 catalog 增一行 descriptor（input kind、state owner、
   entrypoint、read/write effect）——`--scope` 多一个取值，八动词与 JSON 契约复用。
4. Full 门禁 + 治理更新。

## 5. 路径④：内核契约修改（极少发生）

ADR + 三条件缺一不可：现实问题（不是"未来可能需要"）、稳定变化点、
至少两个真实消费者；验证半径 Full/Release。八动词表只增不改；新动词先问
"是不是某个动词的领域内部步骤"。

## 6. 用户定制阶梯（与扩展同构，另一条梯子）

```text
① 调用参数层   单次生效：CLI flag、tool 参数
② 用户配置层   持久偏好：有 schema 的配置文件（.gitconfig 的对应物）
③ 组合层       新工作流：用户 Skill 编排正式命令
④ 登记层       新能力：新 registry entry
⑤ 源码层       改能力行为：上游改 → 更新 pin → install/update
```

铁律：**用户定制永远流经输入（参数、配置、IR），永远不进入 owned 资产**
（plugin cache、junction、内核代码）。可验收不变量：对任何组件执行
update/uninstall，②层用户配置与③层用户 Skill 必须字节不变。

## 7. 明确不提供的扩展点

动态 Go plugin、第二 CLI / 第二 lifecycle / 第二测试引擎 / 第二报告体系、
capability graph、全域事务、远程控制 API、跨领域 SystemManager。
完整拒绝清单见 [核心架构](AICODING_CORE_ARCHITECTURE.md) §11。

## 8. 完成定义（DoD：扩展做到什么程度才算完）

四项知识检查（[架构手册](ARCHITECTURE_HANDBOOK.md) §7.4）全部满足，再加上
与验证半径匹配的门禁绿：

1. **可发现**：命令进 typed catalog / 组件进 registry，`help`、`list`、`status` 可见；
2. **可判读**：JSON 契约齐全，Agent 用 `ok`/`errorKind` 即可程序化判断；
3. **可遵循**：涉及多步编排的功能有对应 SKILL.md；
4. **可纠错**：门禁失败信息指明规则与正确路径。

四项均不满足的功能对 Agent 不存在；部分满足的功能会以支持成本持续收税。
