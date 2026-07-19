# 07 演进路线（Roadmap）

Status: Accepted Direction（已确认方向，非冻结契约）

> 本文按四象限（[00](00-vision.md) §3）组织后续开发计划。方向已确认，但每一项的
> 落地仍须走 [06 扩展规范](06-plugin-sdk.md) 的路径与 ADR 流程；与冻结契约冲突时
> 以契约文档为准。

## 本篇回答的问题

- Repository 如何长期演化？
- 下一步做什么（repo-context），怎么分阶段、怎么验收？
- 生态里已验证的能力（aspens/werkstatt）怎么进来？

## 1. 演化总规则

1. 功能沿"**未知的已知 → 已知的未知 → 地基**"单向沉淀；地基只进不出。
2. 只增不改：新能力以新条目进入，不改冻结语义。
3. 不预建：每项未来功能都带**触发条件**，条件不成立不动工。
4. 每阶段独立提交、独立验证（Smoke/Full 按风险），完成定义见
   [06](06-plugin-sdk.md) §8。

**地基现状**（已知的已知，详见 [01](01-system-architecture.md)）——判定"地基已收敛"
的清单：顶层命令由 typed catalog 唯一登记、不为单个场景加命令；八动词只增不改；
测试仅 Smoke/Full/Release 三档；JSON 报告契约 schemaVersion=1 冻结；Taskfile 纯路由；
PowerShell 专项六类停止增长；已移除的兼容入口不复活（迁移表见 [命令矩阵](../COMMANDS.md)）。

## 2. 已知的未知：既有预留出口（有触发条件才动工）

| 未来功能 | 触发条件 | 落点/出口 | 动内核？ |
|---|---|---|---|
| 流式/交互式执行（进度流、中途审批） | 真实的长任务交互需求 | `report.Result` 传输形态扩展（信封不变，投递方式演进） | 否 |
| 多 Agent 并发写守卫 | 真实并发写场景出现 | expected-digest 守卫：写操作携带"我看到的事实 digest"，事实已变即拒绝 | 否（加守卫不重写） |
| 命令短名（git-alias 式） | 同一长命令组合真实重复出现 | 用户配置展开为既有正式命令，过同一 porcelain 禁用集合 | 否 |
| 不可快照的事实（远程托管等） | 接入此类领域时 | 分类吸收：可快照部分归 input facts，其余归 mutable observation | 否 |
| C/native 性能出口 | 五条件齐备（热点在纯计算内核、Go 已到预算上限、两个真实消费者、同一 golden tests、收益覆盖成本） | 语义之下的物理优化 | 否（语义冻结） |
| 外部集成决策工作流 | 真实集成场景出现 | 按 Draft → RepoLocal 阶梯重建 Skill | 否 |
| Skill 自进化：失败轨迹→技能提炼（`sentient-agi/EvoSkill` 类） | 测试/门禁失败样本积累成规模，且评估证明提炼有效 | Draft 阶梯的自动生成器——产物仍逐级过 Draft → RepoLocal 门禁与同名审计，不直接进运行时 | 否 |
| 会话记忆自动采集（`thedotmack/claude-mem` 类） | 决策库（DECISIONS.md）手工维护成为真实瓶颈 | 决策日志采集器：会话证据 → 追加式条目草稿，人审后入库 | 否 |

## 3. 已知的未知：repo-context 分阶段开发计划（已立项主线）

目标（[00](00-vision.md) §2 三者结合的落地）：把仓库上下文从手工配置升级为
**从代码自动生成、随提交自动更新**的受管资产。参照 `aspenkit/aspens`（MIT）的
已验证做法，在 Go 控制面内实现——**不并入其 npm CLI**（架构禁止第二控制面）。

| 阶段 | 状态 | 做什么 | 产出 | 验收 | 动内核？ |
|---|---|---|---|---|---|
| 0 立项 | ✅ 已完成 | ADR 论证三条件：现实问题=上下文随代码演进漂移、单体指令文件腐化；稳定变化点=代码演进本身；两个真实消费者=本仓库自举 + 受管项目仓库（如 C99 kit 服务的 C 工程）。[ADR 0003](../decisions/0003-repo-context-domain.md)（含 descriptor 与六步准入应答） | ADR + 领域 descriptor | ADR 评审通过 | 否（走路径③，runtime-skill 先例：六模块零修改） |
| 1 扫描 scan | ✅ 已落地 | Go 确定性扫描器（无 LLM）：目录结构、语言/工具链、顶层域 → repo facts snapshot + digest（复用 `internal/registry` 快照原语）。实现于 `internal/repocontext/scan.go` | `repo-context-facts` 事实快照 | 同一仓库两次扫描 digest 稳定（`TestScanIsDeterministic`） | 否 |
| 2 生成 generate | ✅ 已落地 | 从 snapshot 生成每域小粒度 scoped context 文件到受管 owned 根 `.aicoding/repo-context/`；`lifecycle --scope repo-context` 复用 install/update/uninstall/status/doctor/verify。实现于 `internal/repocontext/` + `internal/lifecycle/repo_context.go` | 可按域加载的上下文文件集 + manifest | install/uninstall 往返后**用户手写文件字节不变**（`TestUninstallRemovesOnlyOwnedArtifacts`） | 否 |
| 3 同步 sync | ⏳ 待实现 | commit 驱动增量更新：hook 读本次变更文件 → 映射受影响 context → 只重新生成变了的（aspens `doc sync` 同思路）。与 docsync 分工：docsync **拦**"人写文档没跟上"，repo-context **让**"生成上下文自动跟上" | 提交后上下文自动保鲜 | 改一个文件，只有对应 context 变 | 否 |
| 4 体检 freshness | ⏳ 待实现（域内已具备 status/doctor 对账，待挂入聚合 `doctor --all`/`verify --profile` 与测试 Registry） | 代码事实 digest vs 生成物记录 digest 对账，漂移即报；唯一测试 Registry 登记 leaf gate | 上下文漂移可被机器拦截 | 人为制造漂移能被拦下（`TestStatusReportsFreshThenDriftAfterCodeChange`） | 否 |
| 可选后置 | ⏳ 待评估 | LLM 辅助域发现（aspens 的做法）：默认全确定性，LLM 只作显式可选步骤，产物仍走同一生成器与 digest 对账 | 更好的域切分 | 可对账性不降级 | 否 |

阶段 1–2 的可删除性证明已兑现：删除 `internal/repocontext` 包与 catalog 中一行 descriptor
即可移除本领域，`internal/registry`、`internal/runner`、`internal/report` 零改动。

## 4. 未知的已知：生态吸收计划

| 来源 | 怎么进来 | 边界 |
|---|---|---|
| `Bollwerkio/werkstatt`（MIT，Superpowers 分支） | 先审计其技能与现有 SDD/BDD/TDD/计划模式技能的重叠；**只吸收缺失项**，走路径②（external 子模块 pin + 登记），同名审计防冲突 | 不整包并入；Superpowers 系保持"可选加速"定位（`AGENTS.md` 既有立场） |
| `aspenkit/aspens`（MIT） | **概念重实现**：扫描/scoped context/增量同步的做法进 §3 的 Go 领域实现 | 不并入 npm CLI（不引入第二控制面）；语言覆盖各取所长 |
| 其他外部 Skill | 一律走获取/激活分离四步边界（[FREEZE_AND_ACQUISITION_BOUNDARY](FREEZE_AND_ACQUISITION_BOUNDARY.md) §3.2） | 不复制源码、不带 URL 的激活 manifest |
| 用户 Skill | Draft → RepoLocal → Kit 收编阶梯持续运行（现有实例：升级列车、环境重建） | 准入门禁 + 同名审计 |

### 4.1 对照项目清单（参考学习，不并入）

生态调研核实的参照项目（纯文本坐标 owner/repo；只学做法，不并入源码或 CLI）：

| 类别 | 参照项目 | 学什么 | 落点 |
|---|---|---|---|
| Skill 组织 / AGENTS.md | `mxyhi/ok-skills`、`anthropics/skills`（官方规范事实标准）、`VoltAgent/awesome-agent-skills` | 技能粒度切分、SKILL.md 格式对齐、playbook 组织 | [03](03-skill-architecture.md) |
| 工作流纪律 / 质量门禁 | `obra/superpowers`（上游原版）、`Bollwerkio/werkstatt`、`nizos/tdd-guard` | hook 强制 TDD 的拦截与"失败信息指路"文案、评审工作流 | [04](04-workflow-architecture.md)、[05](05-governance.md) |
| 仓库上下文生成 | `aspenkit/aspens`、`yamadashy/repomix`（全量打包对照路线）、aider 的 PageRank repo-map 机制、`context-hub/generator` | scoped 与全量两条路线取舍、依赖图→上下文选择、增量同步 | 本篇 §3 各阶段 |
| 记忆 / 自进化 | `sentient-agi/EvoSkill`（失败轨迹→技能）、`thedotmack/claude-mem`、`rohitg00/agentmemory` | 失败样本提炼流程、会话记忆采集与压缩注入 | [02](02-context-architecture.md) §3、本篇 §2 新增两行 |
| Skill 安装分发 | `numman-ali/openskills`（通用加载器 + AGENTS.md 同步）、`aiskillstore/marketplace`（带安全审计的市场） | 跨运行时安装形态、市场准入审计清单 | [03](03-skill-architecture.md)、[06](06-plugin-sdk.md) |
| 规则标准与同步 | `agentsmd/agents.md`（AGENTS.md 开放标准本体）、`intellectronica/ruler` | 一份规则源分发到多 agent 工具的 target 模型 | [02](02-context-architecture.md) §1 |

## 5. 未知的未知：兜底机制（不预测，只兜底）

1. **吸收层设计**：新工具、新协议、新分发形态、模型更迭——变化落在登记层 /
   adapter 层 / 调用方层吸收，内核零改动（runtime-skill 域进入时六模块零修改
   即构造性证明）。
2. **解冻规则**：ADR + 现实问题 + 稳定变化点 + 两个真实消费者，缺一不开闸。
3. **拒绝清单**：第二 CLI / 第二测试引擎 / 第二报告体系、动态插件、跨领域
   SystemManager、第四测试档、PowerShell 专项增长——见
   [核心架构](AICODING_CORE_ARCHITECTURE.md) §11 与 [06](06-plugin-sdk.md) §7。

## 6. 节奏

先 §3 阶段 0（ADR），其余阶段依验收逐个推进；§2 各项等触发条件；§4 的 werkstatt
重叠审计可与 §3 并行。任何一步的"完成"以 [06](06-plugin-sdk.md) §8 四项知识检查 +
对应门禁绿为准，不以口头声明为准。
