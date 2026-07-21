# TODO 0012: README 产品化重写（用户 30 秒看懂，3 分钟跑通）

Status: Done
Verify: 新 README 通过 .github/repository-governance.toml 的 required_sections 校验，且"快速开始"三条命令在干净 clone 上逐条可执行

> 产品经理视角的病根：现在的 README 是**维护者写给自己看的**。第一句
> "平台集成、安装、治理和 CodingKit 资产仓库"对新用户是零信息量；
> 9 个徽章占据首屏；"项目边界/当前架构"在讲内部分工，没讲**用户能得到什么**。

## 重写原则（自上而下贯彻架构思想到产品层）

架构文档已有分层阅读路径（0007），README 是它的**产品层投影**——同一思想再往上贯彻一层：

```text
30 秒   这是什么、解决我什么问题        （一段话 + 一张图，零术语）
3 分钟  跑起来看到第一个绿灯            （快速开始，复制即用）
按角色  我是谁 → 我读哪条线            （三条 persona 路径）
```

## 实现计划

1. **首屏重构**（30 秒层，对标 spec-kit / 高星仓库首屏公式：
   banner → 徽章 → 一句话 → 图 → 快速开始）：
   - **顶部 banner 图**（owner 决策）：`docs/assets/aicoding-banner.{svg,png}`，
     居中展示项目名 + 一句话 slogan。用 SVG 手绘或代码生成（可控、可 diff、不依赖
     设计工具）；深浅色模式用 GitHub 的 `#gh-dark-mode-only` 双图技巧。
     **banner 属通用治理标准**（0019 收编：每仓一张，风格同源）。
   - 一句话定位改为用户结果导向，例：
     "AiCoding 让 AI 编码工作流可验证、可复用、可审计：一次 Release 验证 76s，
     内容不变时复用只要 0.4s；每个结论都绑定 Git 内容身份。"
   - **架构图升级为网状图（Mermaid graph，不再用线性 ASCII 链）**——
     GRAPH_FIRST.md 是本仓库的冻结架构文档，README 的架构图必须体现同一网状思维。
     GitHub 原生渲染 Mermaid，零图片维护成本。骨架（落地时按实况微调）：

     ```mermaid
     graph TB
       subgraph 你或 Agent
         U[User / AI Agent]
       end
       subgraph 控制面
         CLI[aicoding CLI]
       end
       subgraph 裁决层
         PLAN[plan 批准绑定内容]
         WORK[work 迭代裁决]
         GATE[hook 门禁 毫秒级]
       end
       subgraph 证据层
         VE[validation evidence<br/>内容寻址 Receipt]
         TE[testengine 三档验证]
       end
       subgraph 能力层
         KITS[6+ Kits 自由组合]
         SKILLS[Skills 子模块]
       end
       U --> CLI
       CLI --> PLAN & WORK & GATE
       PLAN --> VE
       WORK --> VE
       GATE --> VE
       CLI --> TE --> VE
       CLI --> KITS
       KITS --> SKILLS
       VE -.证据复利.-> GATE
     ```

     要求：≤20 个节点；分层 subgraph；至少一条回边体现"证据复利"闭环
     ——网状不是把箭头画多，是让读者看见能力互相咬合。
   - **徽章墙保留并按技术栈如实展示**（owner 决策：技术栈广度是信心信号，
     且后续还会扩展）。约束改为两条，不砍数量：
     a) 每个徽章必须**真实反映当前技术栈**——版本号与 go.mod / CI / 配置实际一致，
        由 docsync 语义漂移检查覆盖（徽章版本漂移 = linkDrift/policyDrift）；
        新增技术栈（如未来 Obsidian kit）随落地加徽章，退役的（如 ps1 面收敛完成后）
        同步撤下——**徽章是技术栈清单的投影，不是装饰**；
     b) **配色分类（owner 决策：不要每个技术都一个颜色）**——按语义分组配色，
        规则进 0019 标准：

        | 组 | 配色规则 | 现有徽章 |
        |---|---|---|
        | 语言/运行时 | 各自官方品牌色（保持现状，本来就对） | Go #00ADD8、PowerShell #5391FE、Python #3776AB |
        | 工具链/格式化 | **统一中性灰蓝**（不再各自一色） | clang-format、Taskfile |
        | 自研 Kit | **统一主色**（与 banner 主色一致，标识"这是我们的东西"） | C UserStyle Kit |
        | 状态类 | 语义色：绿=通过、蓝=版本、灰=许可 | Release、License、**CI status（新增）** |

        **一眼看去应能区分"外部技术 / 我们的能力 / 当前状态"三类**，
        而不是一排彩虹。徽章数量不设上限（技术栈有多少就多少）。
     c) **Star History（owner 决策：现在没 star 也要挂，做成标准）**——
        `star-history.com` 自动 SVG 放**文末独立小节**，不占首屏；
        首屏徽章区只放 shields 类。理由：挂链本身是标准的一部分，
        增长曲线是仓库自己的事，且公开后零改动即生效。
     d) 徽章之后紧跟一句话定位——徽章负责"这仓库有多能打"，
        首句负责"这仓库对你有什么用"，两者各司其职、互不挤占。
2. **3 分钟层**：快速开始压缩到 **3 步**（高星共识铁律：Quick Start ≤3 步、复制即跑）：
   `provision` → `verify --profile Smoke` → `test --profile Smoke`，
   每步加一行"你会看到什么"（期望输出摘要），失败时第一句指向 `doctor --all`。
   其余命令一律链 `docs/COMMANDS.md`，不在 README 展开。
2b. **发展路线露出**（修复"一眼没有发展前景"的病根——**不是没有 roadmap，是没露出来**）：
   快速开始之后两行：链 [07-roadmap](docs/architecture/07-roadmap.md)，
   并写明"活的 roadmap 可机器查询：`aicoding todolist --json`"——
   可执行的 roadmap 强于静态文字，这个差异点要说出来。
3. **Persona 路径**（按角色分流，链接到既有权威文档，不复制内容）：

   | 我是谁 | 我要什么 | 入口 |
   |---|---|---|
   | 新用户 | 跑通并理解价值 | 快速开始 → `docs/COMMANDS.md` |
   | Agent/自动化 | 稳定 JSON 契约 | `kit describe` / `validation check` / report schema |
   | 贡献者 | 不踩红线地改代码 | `docs/architecture/README.md` 必读四篇（755 行） |
   | Kit 作者 | 快速扩展 | `kit init`（0010）→ KIT_MANAGEMENT_STANDARD（0009） |

4. **Kit 能力橱窗**（owner 决策：核心 Kit 层要在 README 露脸，允许适度吹，
   但细节全部外链——README 只当橱窗，不当说明书）：
   - 新增"内核与 Kit"一节：先用 2–3 行吹内核（六模块冻结内核 + 内容寻址验证证据 +
     裁决式 loop——这三样是真的能吹的），再放 Kit 表格，**每个 kit 一行**：

     | Kit | 一句话核心能力 | 详情 |
     |---|---|---|
     | 验证证据 | Release 结论绑定 Git 内容身份，重复验证 0.4s 复用 | → docs/... |
     | c-userstyle-kit | C99 风格裁决：fmt/check/verify 一条链 | → docs/... |
     | docsync-plus | 文档与代码语义漂移评分门禁 | → docs/... |
     | …（全部 enabled kit） | | |

   - **一句话 + 一个 URL，禁止展开**：README 中每个 kit 不超过一行；
     "一句话"直接取 manifest `description`（0009 的门禁保证它面向用户结果），
     URL 指向该 kit 的权威说明（架构文档或 `docs/reference/kits/<id>.md`，
     0008/0009 落地后逐步齐全；暂缺的先指 `docs/COMMANDS.md` 对应节）。
   - 表格与 kit-registry 的一致性纳入验收：enabled kit 必须全部出现、
     无幽灵行（已 disable/移除的 kit 不得残留）——手查即可，
     将来 0009 的 quickstart 投影稳定后可改为生成。
5. **删除/下沉**：Git Governance Standard 全节下沉到 `docs/governance/` 链接
   （README 只留一行）；"当前架构"节压缩为 3 行 + 链接。
6. **README_CN / README_EN 同步**：结构相同，由 docsync 的 readme 规则约束
   （`.github/repository-governance.toml` 已有 required_sections =
   ["status","quick-start","repository-navigation","git-workflow"]——重写必须保住这四节，
   若节名调整则同步改 governance toml 并过 `governance lint`）。
7. **复利仓库一节**（owner 决策：AiCoding 专属吹点，**不进通用治理标准**）：
   在 Kit 橱窗之后加"为什么这个仓库越用越值钱"短节（≤8 行），吹的是已有事实：
   - 地基只进不出（[00-vision §3](docs/architecture/00-vision.md) 四象限：已知的已知=冻结内核，
     一切新能力站在它上面，从不推倒重来）；
   - 证据复利（Receipt 内容寻址：同一内容验证一次，此后 0.4s 复用，跨 worktree 共享）；
   - 能力复利（每个新 Kit 与既有 Primitive 自由组合——loop 复用 evidence，plan 复用 gitx，
     kit init 生成即合规）；
   - 知识复利（四象限每格都有沉淀资产库，见 00-vision §3.1）。
   全节每一句都必须能链到机器事实或权威文档，吹牛不许脱锚。
8. **结构规则可移植**：本项的 README 结构规则（徽章=技术栈投影、能力橱窗一行一 URL、
   30 秒/3 分钟/persona、四象限演进节）在 AiCoding 落地验证后，由 0019 抽取进
   aicoding-git-governance 通用标准；**复利节除外**（仓库个性，不进标准）。
   本项先落 AiCoding 实例，不阻塞在标准升级上。
9. **可执行性验收**：在临时目录 fresh clone 后逐条执行快速开始命令，贴输出。

## 明确不做

- 不在 README 写架构细节/治理细节（链接权威文档，防漂移——docsync 语义漂移
  评分的 docTargets 已覆盖 README）。
- 不加动图/截图（收敛阶段裁决：过期的演示是负资产）。**唯一预留路线**：将来公开
  推广阶段若要加，只允许 `vhs` tape 脚本生成（脚本入库、GIF 可再生），禁止手录。
- 不为 Obsidian 知识体系等未来扩展预留章节（届时随 kit 落地再加）。

## 自测（可信任方式）

```powershell
bin\aicoding.exe governance lint --json          # readme required_sections 校验
bin\aicoding.exe docsync all --json
bin\aicoding.exe verify repo-text --json
# fresh clone 可执行性：
$tmp = Join-Path $env:TEMP "readme-test-$(Get-Random)"
git clone <repo> $tmp ; cd $tmp
# 逐条执行 README 快速开始，贴每条输出；任何一条失败即本项不通过
```

通过判据：governance lint 绿；三条 persona 链接全部可达（lychee，含每个 kit 的详情 URL）；
快速开始在 fresh clone 上零修改跑通；徽章逐个核对版本与 go.mod/CI/配置一致
（贴核对清单）；Kit 表格与 kit-registry enabled 集合一致（无缺行、无幽灵行，
每行一句话 + 一个 URL、无展开段落）；首屏正文（徽章区之后的前 25 行）无一个内部术语
（CodingKit/kit registry/控制面 等词不得出现在首屏正文）。
