# TODO 0025: 架构图体系（README 总图 + 四张分层图）

Status: Done
Verify: README 总图与 docs/architecture/ 四张分层图全部为 Mermaid 源码、GitHub 原生渲染、lychee 链接全通；每张图节点数 ≤20

> 来源：owner "当前 README 那张图我根本看不懂"。
> 病根：现有图把**执行链**画成一条线，但读者要回答的是
> **"我该用哪个命令、它靠什么支撑、凭什么可信"**——这三问需要分层，不是一条链。

## 一、总图（进 README，替换现有架构图）

设计原则：**一张图回答三个问题**——用户从哪进、能力怎么分层、结论凭什么可信。

```mermaid
graph TB
  subgraph U["① 你 / Agent 从这里进"]
    CLI["aicoding CLI<br/>单一控制面"]
  end

  subgraph P["② Product Workflow —— 可直接执行的完整工作流"]
    PLAN["plan<br/>批准绑定内容"]
    WORK["work<br/>迭代裁决"]
    TEST["test --profile<br/>Smoke/Full/Release"]
    LIFE["lifecycle<br/>声明式收敛"]
    GOV["governance / docsync<br/>治理与文档"]
  end

  subgraph K["③ Kit —— 可安装可分发的能力资产包"]
    K1["c-userstyle-kit"]
    K2["docsync-plus"]
    K3["loop-engineering-kit"]
    K4["… kit list 可查"]
  end

  subgraph D["④ Domain Capability —— 内部实现域"]
    VE["validationevidence<br/>内容寻址 Receipt"]
    TE["testengine<br/>三档验证引擎"]
    LK["loopkit<br/>停止裁决"]
    PL["plan<br/>范围契约"]
  end

  subgraph PR["⑤ Primitive —— 单一职责基础能力"]
    GX["gitx"]
    RP["report"]
    RN["runner"]
    RG["registry"]
  end

  CLI --> PLAN & WORK & TEST & LIFE & GOV
  PLAN --> PL
  WORK --> LK
  TEST --> TE
  LIFE --> K1 & K2 & K3
  GOV  --> RG

  PL & LK & TE --> VE
  VE & TE & LK & PL --> GX & RP & RN & RG

  VE -."内容不变即复用<br/>76s → 0.4s".-> TEST
  VE -."证据即门禁".-> PLAN
```

**图要传达的三件事（配图文字，各一行）：**

```text
① 只有一个入口     所有能力都从 aicoding CLI 进，没有第二控制面
② 能力分五层       上层组合下层，下层永不反向依赖
③ 证据形成闭环     验证结论绑定 Git 内容身份，同一内容零成本复用（虚线回边）
```

## 二、四张分层图（进 docs/architecture/）

每张图独立回答"这一层是什么、怎么用、边界在哪"，**≤20 节点**，
放进各自已有的架构文档，不新建文档。

### 图 1：Primitive 层 → `docs/architecture/PRIMITIVE_CONSTITUTION.md`

画：五个 Primitive 各自的单一职责 + 禁止的反向依赖边（红色虚线标注 forbidden）。
重点是**依赖方向**，因为这层唯一要守的就是方向。

### 图 2：Domain Capability 层 → `docs/architecture/AICODING_CORE_ARCHITECTURE.md`

画：六模块（snapshot/plan/runner/adapter/report/state）与四个域能力的对应关系。
重点是**哪些是冻结的**（加粗/实线）vs **哪些是扩展位**（虚线）。

### 图 3：Kit 层 → `docs/reference/KIT_PLUGIN_VIEW.md`

画：kit-registry → manifest → commands/skills/state → lifecycle 八动词 的投影链。
重点是 **Kit 与 internal 域的区别**（Kit 是交付单元，internal 是实现域）。

### 图 4：Product Workflow 层 → `docs/COMMANDS.md`

画：一次完整开发闭环的命令时序——
`plan check → 改代码 → change verify → plan approve → commit → pre-push gate`。
重点是**每一步谁在裁决、裁决依据是什么**。

## 三、实现约束

1. **全部 Mermaid 源码**，GitHub 原生渲染，零图片维护（0012 已定的规则）。
2. **≤20 节点/图**，超了就是该拆——图的价值在于看得懂，不在于画得全。
3. **总图进 README 的生成区之外**（它是手写的架构表达，不是投影），
   但 **Kit 列表用 0023 的生成区**，两者不混。
4. 每张图配 **≤3 行文字**说明"这张图回答什么问题"，不写解说词。
5. 图中出现的每个命令必须在 typed command catalog 中真实存在
   （新增 static 用例 `DOCS-006` 校验：扫描架构文档中的 `aicoding xxx`
   代码片段，断言命令名 ⊆ catalog）。

## 四、明确不做

- 不用 SVG/PNG 画架构图（只有 banner 用 SVG，0012 已定）。
- 不画时序图/类图/ER 图（当前无消费者）。
- 不为每个 internal 包画图（0023 已裁决：无公共入口的域零新增文档）。
- 不做图的自动生成（依赖关系图有 `governance dependencies` 的 JSON，
  想看的人可以查；渲染成图无第二消费者）。

## 五、自测

```powershell
bin\aicoding.exe docsync all --json          # 架构文档 Status 与链接
lychee --config lychee.toml README.md docs/  # 图中链接全通
bin\aicoding.exe test --profile Full --json  # 含 DOCS-006
#   DOCS-006 负例：在架构文档里写 `aicoding nonexistent` → 必须红 → 撤销后转绿
#   节点数断言：脚本统计每个 mermaid 块的节点数 ≤20
```

通过判据：五张图全部 GitHub 渲染正常（贴截图或渲染确认）；
DOCS-006 负例被抓；每图 ≤20 节点；README 总图与四张分层图无内容重复
（总图只画层与闭环，细节全在分层图）。

## 六、执行裁决与证据（2026-07-21）

执行时按真实运行条件修正了草案时序：`plan approve` 只接受 clean HEAD，因此批准必须发生在
编辑之前；TODO 0021 的 `change verify` 尚未实现，本项不提前虚构命令，使用现有
`verify --profile Smoke`。五张图均为手写 Mermaid 源码，没有新增 SVG/PNG 或图生成器。

| 验证 | 实际结果 |
|---|---|
| 五图职责 | README 只画单一入口、五层与证据回边；Primitive 图画依赖方向；Core 图区分六个冻结模块与四个扩展域；Kit 图区分交付投影与 `internal/kit`；Commands 图按 clean approve、staged gate、commit、Release Receipt、pre-push exact OID 排列。 |
| DOCS-006 | 唯一 test engine 新增 required static case；直接用 Go AST 读取 `internal/cli/catalog.go` 的 `CommandDescriptor` name/alias，不复制第二命令表，也不导入 `internal/cli`。每个载体必须恰好一个 Mermaid block、显式节点数为 1..20、图内 `aicoding xxx` 顶层命令必须存在。 |
| 单测 | `go test ./internal/testengine -count=1` 通过；包含 typed catalog 正例、未知命令文件/行号、21 节点超限、真实仓库集成与 Registry 登记测试。 |
| 未知命令负例 | 临时将 Commands 图改为 `aicoding nonexistent` 后，`TestRepositoryArchitectureDiagrams` 按预期失败：`docs/COMMANDS.md:53: diagram command "aicoding nonexistent" is absent from internal/cli typed catalog`；恢复后转绿。 |
| 节点预算 | README / Primitive / Core / Kit / Commands 分别为 `15 / 6 / 10 / 9 / 11` 个显式节点，五个载体均恰好一个 Mermaid block。 |
| 真实渲染 | `@mermaid-js/mermaid-cli 11.12.0` 使用本机 Edge 150 分别渲染五个 Markdown；每个输入恰好产生一个 SVG，大小依次为 `44168 / 22548 / 34035 / 28760 / 23756` bytes，产物仅在系统临时目录，仓库零图片新增。 |
| 链接审计 | 首轮暴露 TODO 0012 两条既有相对路径错误及匿名 Star History 图片端点 500；修正路径并保留页面链接、移除失效图片后，`lychee --config lychee.toml README.md docs/` 为 `286 OK / 0 Errors`。 |
| DocSync | `bin\aicoding.exe docsync all --json` 返回 `ok=true`、`warnings=[]`、`errors=[]`。 |
| Full | `bin\aicoding.exe test --profile Full --reuse off --allow-dirty --json` 为 `66 total / 63 pass / 0 fail / 0 warn / 3 skip`，`duration_ms=181172`；DOCS-006 单项 PASS（4ms）。dirty subject 明确不可复用，未伪装 Receipt。 |

本项实现、负例、节点预算、真实渲染、链接、DocSync 与 Full 均已通过，状态翻为 Done。
