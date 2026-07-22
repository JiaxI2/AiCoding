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

1. 用 `kit init <id>` 或 `mcp init <id> --out config/mcp/components` 生成并评审
   manifest；两者都不会自动启用，`mcp init` 也不会修改 registry。
2. 写 manifest：`config/kits/<id>.json` 或 `config/mcp/components/<id>.json`，
   通过对应的冻结 schema 校验（五个冻结 schema 之一，见
   [FREEZE_AND_ACQUISITION_BOUNDARY](FREEZE_AND_ACQUISITION_BOUNDARY.md)）。
3. 实现真实 capability，并把生成器给出的 disabled entry 经人工评审后登记到
   `config/kit-registry.json` / `config/mcp-registry.json`。
4. 预览并安装：`lifecycle plan --action install --scope kit|mcp …` →
   `lifecycle install …`。
5. 过门禁：`verify --profile Smoke` + `test --profile Smoke`（按风险升 Full）。

MCP 脚手架只生成冻结结构、三个 profile 占位和 registry 建议，不生成业务 runtime，
所以不能把“manifest 合规”误报成“component 已可运行”。完整步骤见
[创作指引](../guides/AUTHORING.md)。

效果：内核与 CLI **零改动**；新组件自动出现在 `list`/`status` JSON 里
（数据变化，不是接口变化）；Agent 无需学任何新东西。

命名约束：可复用能力用平台无关域名；只有真正依赖 AiCoding 行为的资产才允许
`aicoding-*` 前缀；身份里永不编码版本号。

### 2.1 路径①的 pinned 引用注册（外部 Kit，不 vendoring）

外部 Kit 仍属于路径①的 manifest 变体，不需要新领域 adapter。把仓库内 schema v2 manifest
的可选 `source` 写为以下二选一，并保持 `trust.thirdParty:true`、`updatePolicy:pinned`：

```json
{"kind":"git","url":"https://example.invalid/upstream.git","commit":"<40-hex>"}
{"kind":"content","digest":"sha256:<64-hex>"}
```

`kit register --manifest config/kits/<id>.json --prefetch --json` 只登记 manifest 路径和依赖
binding；不复制外部源码。注册阶段可把 Git 网络成本交给后台 prefetch，或用
`kit prefetch --id <id> --json` 显式等待。后续 `lifecycle install|update --scope kit` 只从已验证
的本地内容寻址 cache materialize，网络调用固定为 0；cache 缺失时返回 requiredAction，不能把
“pin 尚未解析”放松成“路径可以不存在”。branch、tag、缩写 SHA 与错误 digest 均 fail-closed。

这与 §6 的 doctrine 一致：外部 source/pin 是显式输入；pin cache 与 materialization 是可重建的
owned 资产；用户配置和外部 checkout 不进入 owned 资产，update/uninstall 不修改它们。设计决策
见 [ADR 0010](../decisions/0010-pinned-reference-registration.md)。

## 3. 路径②：新 external Skill（跨仓库登记）

1. 在 AiCoding 仓库外的可写 Codex-Skills checkout/worktree 创作；可先用
   `skill init <id> --out <external-path>` 生成结构合规但尚未批准的草稿。命令会拒绝
   AiCoding 仓库和只读 `CodingKit/agents/skills` 内的输出。
2. 上游仓库进入 Codex-Skills 的 `external/` **嵌套子模块**并 pin 到发布版本
   （不复制源码；Codex-Skills 侧登记 binding manifest）。
3. AiCoding 侧把运行时名字登记进 full profile 的独立 Skill 清单，并把名字映射到
   含 `SKILL.md` 的具体目录（sourcePaths）。
4. `lifecycle install|update --scope runtime-skill --runtime-profile full …` 暴露；
   运行时审计拒绝同名 active Skill。
5. 升级：解析上游最高稳定 semver tag，评审后前移 pin；卸载只删除目标精确匹配的
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

消费者查看已登记 Kit 的边界与 JSON 契约见
[Kit Plugin View（消费者侧只读投影）](../reference/KIT_PLUGIN_VIEW.md)；该 View 不定义扩展接入规则。
