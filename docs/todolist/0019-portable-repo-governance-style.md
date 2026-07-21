# TODO 0019: 可移植仓库治理风格（README 规则 + 四象限入 aicoding-git-governance 标准）

Status: Planned
Verify: Codex-Skills 侧标准与模板更新合并、pin 前移后，AiCoding 的 governance lint 以新 required_sections/规则通过；第二个仓库用模板实例化可复现同风格

> Owner 意图：**"以后我每一个 git 仓库的治理标准风格都差不多。"**
> 载体不新建——`aicoding-git-governance` 已经是干这件事的（canonical 标准文档在
> Codex-Skills，`REPOSITORY_GOVERNANCE_TEMPLATE.toml` 模板实例化到每仓
> `.github/repository-governance.toml`，`governance lint` 校验）。本项是给这个
> 既有标准**升级一个版本**，把 0012 在 AiCoding 验证过的 README 风格规则收编进去。
> **依赖 0012 先落地**——先有验证过的实例，再抽标准（与"先测后改"同一纪律）。

## 通用 vs 个性的边界（本项最重要的一条裁决）

| 规则 | 归属 | 理由 |
|---|---|---|
| **顶部 banner**（SVG 代码生成 + `#gh-dark-mode-only` 双图，风格同源） | ✅ 通用可选 profile | SVG 可 diff 可维护；public/product 仓可启用，不强迫普通私仓做营销图 |
| 徽章 = 技术栈投影（真实、随栈增删、版本与配置一致） | ✅ 通用标准 | 任何仓库适用 |
| **徽章配色分类**（语言=品牌色 / 工具链=中性 / 自研=主色 / 状态=语义色） | ✅ 通用标准 | 一眼区分"外部技术 / 我们的能力 / 当前状态" |
| **Star History 节**（文末） | ✅ public profile 可选 | 公开页面链接优先；匿名 chart 端点可用才嵌入，token/credential/auth 一律禁止 |
| **Mermaid 网状架构图**（≤20 节点、分层 subgraph、需要闭环时至少一条回边） | ✅ 通用可选 profile | GitHub 原生渲染，可 diff，零图片维护 |
| 能力橱窗：核心模块/包一行一句话 + 一个 URL，禁止展开 | ✅ 通用标准 | 任何仓库适用（"能力"因仓库而异：kit/库模块/服务） |
| 30 秒 / 3 分钟 / persona 三层 README 结构 | ✅ 通用标准 | 产品思维不挑仓库 |
| **四象限演进观**（已知的已知=地基…四格 + 各格沉淀资产） | ✅ 通用标准 | owner 明示"所有仓库可以用上"；标准提供表格模板，各仓库填自己的格 |
| **复利仓库叙事**（越用越值钱一节） | 🔴 **不进标准** | owner 明示；它依赖内容寻址证据等 AiCoding 特有机制，别的仓库吹不起 |

## 跨仓库工作切分（子模块边界，踩过的坑不再踩）

标准文档与模板都在 **Codex-Skills 仓库**（AiCoding 里是只读子模块）。工作分两侧：

## AiCoding 侧待外溢清单（2026-07-22；本轮只准备，不执行跨仓变更）

### 当前事实快照

- `CodingKit/agents/skills` 当前 gitlink 为
  `7c605788baf2ba7301a454f12d8bef089cbf7e36`，本轮 `git diff --submodule` 为空；
- 子模块 canonical 文档写 `Standard version: 2026.07`，其模板也写 `2026.07`；
  AiCoding `.github/repository-governance.toml` 与 `internal/governance/issues.go` 当前却固定
  `2026.07.16`。上游 PR 必须先决定并统一下一版本，不能在本仓先猜 `2026.08`；
- AiCoding 的 Go `governance lint` **没有读取** `[readme].required_sections`，目前仍以
  `internal/governance/governance.go` 的 AiCoding 专属 token/正则硬校验 README；只有 Release
  Notes Python validator 会读取 release 的 `required_sections`。因此 B 侧不是“泛化则零改动”，
  而是必须在上游标准明确字段语义后，选择一个最小的本仓 adapter，不能误报已有通用实现；
- 0012 的当前实例已验证 6 个 enabled Kit 橱窗、三行 Quick Start、四条 persona、双主题
  banner、版本 badge 权威和 Mermaid 节点/命令门禁。`README.md` 的 Star History 使用公开页面
  链接；三份 README 均未发现 `sealed_token`、`token=`、`credential=` 或 `auth=`。

### A 侧 PR 必须带出的规则包

| 规则域 | canonical 必须说清的通用契约 | 模板 / lint 最小落点 | 强度 |
|---|---|---|---|
| 中文优先与文件级双语 | `README.md` 中文优先；顶部显式链 `README_CN.md` / `README_EN.md`；三文件的身份、命令、支持边界和 release 链接不得冲突 | 复用现有 `primary_language`、language files 与 GitHub About 字段 | baseline required |
| 可再生 banner | 每仓可选一组同源 light/dark SVG，使用 `#gh-light-mode-only` / `#gh-dark-mode-only`；源必须可 diff、无未解析占位；不能要求所有私仓都做营销图 | 新增可选 `[readme.banner]`；模板只给路径/模式，不复制 AiCoding 图 | profile controlled |
| badge = 技术栈投影 | 只展示真实默认栈；随栈增删；显示版本绑定单一权威；三份 README badge block 一致；语言用品牌色、工具链用中性色、自研能力用主色、状态用语义色；徽章数量不设人为上限 | `[readme.badges] policy/verify_versions/color_policy`；lint 检查 HTTPS、版本源与文件间一致性 | enabled 时 required |
| 30 秒 / 3 分钟 / persona | 首屏一句用户结果；Quick Start 最多 3 步、可复制、写预期结果与 doctor 失败入口；按角色只链接权威文档，不在 README 展开细节 | `required_sections` 增 `value-proposition`、`quick-start`、`personas` 的稳定语义，不绑定中文标题字面值 | profile controlled |
| Mermaid 网状图 | 使用 GitHub 原生 Mermaid；每图不超过 20 个显式节点；分层 subgraph；需要表达闭环时至少一条回边；图中命令必须是真实命令 | 可选 `[readme.architecture_graph]`；lint/adapter 校验节点预算，命令目录由各仓注入 | enabled 时 required |
| 能力橱窗 | 每个启用能力恰好一行、一句用户结果、一个详情 URL；无缺行、无幽灵行；标准只定义投影契约，不规定 registry 形状 | `[readme.capability_showcase]` 声明 enabled、source adapter、每项一行和 URL 要求 | profile controlled |
| 四象限演进观 | 提供“已知的已知 / 已知的未知 / 未知的已知 / 未知的未知”空表模板，仓库填写自己的沉淀资产；README 或 vision 二选一 | `[readme.evolution] quadrants_section = recommended|required|off` | 默认 recommended |
| Star History | public 仓库可在文末提供公开页面链接；匿名 chart 只有端点实测可用时才可选嵌入；任何 `sealed_token`、token、credential、auth 参数一律 fail-closed；private/off 不强求 | `[readme.star_history] mode = page-link|chart|off`，默认不携带凭据 | public profile 可选/可配置 |
| 仓库个性 | “为什么这个仓库越用越值钱”等复利叙事只有事实可追溯的仓库自行写，绝不进入 baseline 模板或 required section | 无字段、无模板 | 明确排除 |

上游 PR 不能只改 prose。必须同步：

1. `references/aicoding-git-governance-standard.md` 与版本号；
2. `assets/REPOSITORY_GOVERNANCE_TEMPLATE.toml` 的可选字段、默认值和注释；
3. `assets/lint-git-governance.ps1`、必要的 renderer/validator 及测试夹具；
4. 旧 TOML 不启用新字段时行为字节级/结论级不变的兼容测试；
5. 一个非 AiCoding fixture，证明 source adapter 可替换而不是把 `kit-registry` 写死进标准。

### 上游合并后 B 侧的精确落点

1. 只把 `CodingKit/agents/skills` gitlink 前移到已合并 commit；不在挂载目录直接改源码，
   记录上游 PR URL、merge SHA、canonical version 与 pin SHA。
2. `.github/repository-governance.toml` 采用正式字段：AiCoding 选择 badge projection、banner、
   graph、showcase enabled、quadrants required、Star History `page-link`；版本必须与 pin 内容一致。
3. `internal/governance/issues.go` 当前硬编码 `2026.07.16`，需随版本最小更新；同步
   `internal/governance/commit_test.go`、`internal/cli/cli_test.go` 等 fixture，不能只改生产字符串。
4. `internal/governance/governance.go` 当前是 AiCoding 专属 lint，不解析 README
   `required_sections`。按上游最终字段增加最小 config decode/adapter；通用标准校验仍由
   Codex-Skills lint 拥有，本仓不复制第二份规则引擎。
5. 复用 `config/dependency-governance.json#versionVisibility.readmeBadges` 作为 AiCoding badge
   版本权威，复用 Kit registry / capability registry 的现有投影门禁；不新增 README registry。
6. 三份 README 统一 Star History 模式，避免一份 page link、两份匿名 chart 的长期漂移；
   保留 AiCoding 专属复利叙事，但不把它写回标准。
7. 新增“新仓库如何采用”指引，与 provision 文档互链；只说明复制模板、填项目/分支与运行
   lint 的最短路径，不建立多仓批量迁移器。

### 必须真跑的兼容与负例矩阵

| 场景 | 期望 |
|---|---|
| 旧 TOML 完全没有新字段 | 与升级前结论一致，不被新 required section 反向阻断 |
| badge 声称错误版本、三份 README 不一致或目标不是 HTTPS | lint 非零退出并指出具体 badge / 文件 |
| showcase enabled 但少一个 enabled capability，或多一个幽灵行 | adapter/lint 非零退出；`off` 时不强求橱窗 |
| quadrants=`required` 但 README/vision 均无四格；`recommended` 时缺失 | required 非零；recommended 只给可判读 warning，不伪装 error |
| Star History URL 含 `sealed_token` / token / credential / auth | 无论链接是否可达都非零退出；`page-link` 不要求匿名 chart API 可用 |
| Mermaid 超过 20 节点或引用不存在的命令 | 对应仓库门禁非零退出 |
| 第二个非 AiCoding 仓库实例化 | 不需要 AiCoding Kit registry，也能得到同类首屏、Quick Start、persona 与四象限结构 |

### 本轮停止条件

本轮只提交上述清单并保持 `Status: Planned`。在 Codex-Skills 独立 PR 合并、gitlink pin 前移、
AiCoding 新字段落地、旧 TOML 兼容负例和第二仓实例化全部有真实证据前，**不得**把 0019
翻为 Done，也不得把本清单描述成已交付标准。

本轮准备验证：

```text
governance lint                                                        PASS
docsync all                                                            checked=805, errors=0, warnings=0
verify repo-text / plan verify                                         PASS
rg sealed_token|token=|credential=|auth=（三份 README / docs / .github） 0 hits
git diff/status -- CodingKit/agents/skills                             empty
todolist                                                               24 Done / 1 Planned（0019）
```

未创建 Codex-Skills 分支或 PR，未改 standard/template/lint，未更新 AiCoding local TOML，
也未用第二个仓库冒充可移植性验收；这些都明确保留为跨仓后续证据。

### A. Codex-Skills 侧（独立 PR，走该仓库流程）

1. `platform/aicoding-git-governance/references/aicoding-git-governance-standard.md`
   （canonical）按其正式版本策略升版（不能由 AiCoding 侧预猜版本号），新增章节：
   - **README 风格**：三层结构（30s/3min/persona）；徽章政策（技术栈投影三原则：
     全真、随栈增删、版本单源可核对）；能力橱窗（一行一 URL、与能力注册表一致、
     无幽灵行）；
   - **四象限演进节**（推荐而非强制）：README 或 vision 文档含四格表，
     模板给空表 + AiCoding 的填法作为示例链接。
2. `assets/REPOSITORY_GOVERNANCE_TEMPLATE.toml` 同步新增可配置面（示意，
   实际字段名、默认值和关闭语义服从上面的待外溢清单与上游评审）：

   ```toml
   [readme.badges]
   policy = "tech-stack-projection"   # 或 "minimal" —— 仓库可选
   verify_versions = true

   [readme.capability_showcase]
   enabled = true
   max_lines_per_item = 1
   require_detail_url = true

   [readme.evolution]
   quadrants_section = "recommended"  # recommended | required | off
   ```

3. `required_sections` 的可选值扩展（如增 `capabilities`、`evolution`），
   保持向后兼容：老仓库不改 TOML 行为不变；稳定值表达语义，不绑定某一种语言的标题字面值。

### B. AiCoding 侧（子模块 pin 前移后）

1. 前移 `CodingKit/agents/skills` pin 到含新标准的提交（走既有 pin 前移评审）。
2. `.github/repository-governance.toml` 实例化新字段（AiCoding 选
   `tech-stack-projection` + showcase enabled + quadrants required——它是示范仓）。
3. `governance lint`（Go 侧）当前已确认没有读取 README `required_sections`，而是
   AiCoding 专属硬编码检查；上游字段确定后增加最小 config decode/adapter，并同步版本与 fixture，
   不在本仓复制 canonical lint 规则。
4. `docs/guides/` 增一页"新仓库如何采用"：三步（复制模板 → 填 [project]/[branch] →
   provision + governance lint 绿），与 0011 的 provision 骨架互链——
   **provision 放目录约定，governance 模板放规则约定，两者合起来就是
   "新仓库出生即有统一风格"**。

## 明确不做

- 不新建第二套治理标准/不 fork 标准文档进 AiCoding（canonical 只有 Codex-Skills 一份）。
- 复利叙事不进标准（见边界表）。
- 不做多仓库批量迁移工具（第二个仓库手工实例化即可；出现第三个仓库再谈自动化）。
- 不把 AiCoding 的 kit-registry 机制抽象进标准（能力注册表因仓库而异，
  标准只要求"橱窗与注册表一致"，注册表长什么样各仓自定）。

## 自测（可信任方式）

```powershell
# A 侧合并、B 侧 pin 前移后：
cd F:\Study\AI\worktrees\AiCoding
git -C CodingKit/agents/skills log --oneline -1        # pin 已含标准新版
bin\aicoding.exe governance lint --json                # 新 required_sections 下绿
bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Smoke --json

# 可移植性实证（核心验收）：拿一个非 AiCoding 仓库（如 Git-Agent-Kit）实例化：
#   复制 REPOSITORY_GOVERNANCE_TEMPLATE.toml → 填 project/branch → 按标准写 README 骨架
#   → 该仓库的 README 结构与 AiCoding 同风格（徽章投影/橱窗/三层/四象限表）
#   贴两仓 README 首屏对照截图或文本对比
```

通过判据：canonical 标准升版且 AiCoding toml 引用新版本号；governance lint 在新规则下绿；
第二个仓库实例化后风格一致（对照贴出）；老 toml（未启用新字段）行为不变（向后兼容负例）；
AiCoding 子模块仅 pin 前移、无内容直改（`git status` 证明）。
