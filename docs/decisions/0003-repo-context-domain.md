# ADR 0003: 立项 repo-context 领域 adapter（仓库上下文自动生成与保鲜）

## Status

Accepted（阶段 0–4 全部实现落地：扫描 + 生成 + commit 增量同步 + 聚合门禁——
`internal/repocontext` + `internal/lifecycle` 的 `repo-context` adapter + `hook post-commit`
+ `doctor.repo-context`/`verify.repo-context` 聚合 check + 测试 Registry `RC-001/RC-002`）

## Context

现实问题（不是"未来可能需要"）：

- Agent 所需的仓库上下文随代码演进漂移：`config/repository-navigation.json` →
  REPOSITORY_MAP 生成区块靠人记得去改，目录用途变了索引可能悄悄过时；
- 索引粒度只到目录级，Agent 进入具体域仍要现场摸索，重复消耗 Token；
- 生态已证明替代形态可行（`aspenkit/aspens`，MIT）：从代码派生每域约 35 行的
  scoped context，commit 后增量更新——单体指令文件（数千行 AGENTS.md/CLAUDE.md）
  腐化是公认反模式。

三条件检验（内核/边界变更准入，[架构手册](../architecture/ARCHITECTURE_HANDBOOK.md) §5.3）：

1. **现实问题**：如上。
2. **稳定变化点**：代码演进本身——事实每次提交都在变，这个变化点永不消失。
3. **两个真实消费者**：AiCoding 仓库自身（自举）+ 受管项目仓库（如 C99 kit
   服务的 C 工程）。

为什么现有三个领域无法表达（对抗性追问的答案，
[EXTENSION_ADAPTER_CONTRACT](../architecture/EXTENSION_ADAPTER_CONTRACT.md) §10）：

- kit / mcp / runtime-skill 管的都是**外部获取后登记的资产**，输入是
  registry/manifest（`kit-catalog`、`mcp-catalog`、`runtime-skill-registry`）；
- repo-context 管的是**从仓库事实派生的生成物**，输入是仓库扫描快照，
  没有 manifest 可写——它不是任何现有领域的 manifest 变体；
- 生成物的状态所有权（"哪些文件是我生成的、可以收敛/删除"）不属于任何现有
  StateOwner。

## Decision

新增第四个领域 adapter（扩展路径③），descriptor 草案严格对齐
[EXTENSION_ADAPTER_CONTRACT](../architecture/EXTENSION_ADAPTER_CONTRACT.md) §2：

```text
ID:         repo-context
InputKind:  repo-context-facts   规范化仓库事实快照：目录树、语言/工具链识别、
                                 依赖边（import/include）。相对路径；绝对路径
                                 不进入 digest（同 runtime-skill 先例）。
StateOwner: repo-context          owned = 生成的 scoped context 文件 + 其清单与
                                 digest 记录。不拥有：用户手写文件、其他领域资产。
Entrypoint: go-static             纯 Go 实现，无外部进程依赖。
Actions:    install  (write)      从事实快照首次生成 scoped context 资产
            update   (write)      重扫描并收敛生成物到当前代码事实（含增量路径）
            uninstall(write)      只删除清单登记且 digest 匹配的生成物
            status   (read)       对比"代码事实 digest"与"生成物记录的 digest"
            doctor   (read)       诊断生成物缺失/漂移/越界，只报告不修复
            verify   (read)       校验生成物结构与新鲜度契约
```

- **暂不设 `rollback`**：update 即"重新收敛到事实"，天然可恢复；真实回滚需求
  出现后按"只增不改"追加动作（不预建）。
- 顶层 `plan` 沿用契约：`lifecycle plan --action X --scope repo-context` =
  X + dryRun，plan/apply 共用同一领域路径。
- **不新增 CLI 顶层命令**：全部经 `lifecycle --scope repo-context` 进入；
  `--scope all` 的稳定执行顺序在 catalog 中追加于 runtime-skill 之后。

阶段划分、产出与验收照抄 [07 演进路线](../architecture/07-roadmap.md) §3
（阶段 1 扫描 → 2 生成 → 3 commit 增量同步 → 4 新鲜度门禁；可选后置 LLM 域发现
默认关闭且产物走同一 digest 对账）。

六步准入义务（契约 §10）的应答：

1. 先定义具体领域模块 `internal/repocontext`（暂名），不先定义通用接口；
2. input facts / state owner / actions+effects / entrypoint 如上；
3. descriptor + 静态函数进 `internal/lifecycle` catalog（一行），无 init 期
   隐式注册、无全局可变表；
4. 请求翻译为 typed domain values，返回 domain result（`AdapterResult` 复用）；
5. 测试：catalog contract、领域模块契约测试、lifecycle consumer 回归、
   CLI JSON contract；影响半径按契约 §11 表执行；
6. **可删除性证明义务**：删除该 adapter 只需移除 catalog 一行与领域包，
   `internal/registry`（snapshot）、`internal/runner`、`internal/report` 零改动
   ——阶段 1 落地时以此为验收项之一。

非目标（明确不做）：

- 不并入 `aspenkit/aspens` 的 npm CLI（禁止第二控制面）；概念参照，Go 重实现；
- 默认零 LLM：扫描与生成全确定性，同一仓库两次扫描 digest 稳定；
- 永不覆盖用户手写内容：update/uninstall 往返后用户文件字节不变
  （[架构手册](../architecture/ARCHITECTURE_HANDBOOK.md) §6.1 定制铁律）；
- 不加第四测试档、不改 JSON 报告契约、不新增知识进入点类型
  （生成物是可被 `lifecycle status` 枚举的受管资产数据）。

## Consequences

Positive：

- 仓库上下文随提交自动保鲜，Agent 理解项目不再依赖过时索引或现场摸索；
- 每域约 35 行的 scoped context 按需加载，Token 成本下降；
- 复用八动词与 JSON 契约，Agent 零新增学习成本（知识进入点数量不变）；
- "已知的已知"资产（上下文库）持续变厚而调用成本不升——四象限复利指标的
  直接落地。

Trade-offs：

- 新增一个领域模块的长期维护成本（扫描器需随语言/工具链场景演进）；
- 生成物所有权纪律要求严格（清单 + digest 记录必须与事实同步，否则 uninstall
  不敢删）；
- 阶段 3 的 commit 驱动增量同步给 hook 链增加一个挂点，需守住"hook 是薄壳、
  秒级完成"的原则（慢路径放 update 全量收敛）。

## Rollback

阶段 0 仅文档，回滚 =

```text
docs/decisions/0003-repo-context-domain.md
CHANGELOG.md 中对应的 docs(plan) 条目
docs/architecture/07-roadmap.md §3 中指向本 ADR 的一句引用
```

后续实现阶段各有其回滚：catalog 行与 `internal/repocontext` 包整体可删
（见"可删除性证明义务"），生成物经 `lifecycle uninstall --scope repo-context` 清除。
