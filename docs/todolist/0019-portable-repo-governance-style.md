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
| **顶部 banner**（SVG 代码生成 + `#gh-dark-mode-only` 双图，每仓一张、风格同源） | ✅ 通用标准 | owner 明示"有标准意识"；SVG 可 diff 可维护，不依赖设计工具 |
| 徽章 = 技术栈投影（真实、随栈增删、版本与配置一致） | ✅ 通用标准 | 任何仓库适用 |
| **徽章配色分类**（语言=品牌色 / 工具链=中性 / 自研=主色 / 状态=语义色） | ✅ 通用标准 | 一眼区分"外部技术 / 我们的能力 / 当前状态" |
| **Star History 节**（文末，挂链即标准） | ✅ 通用标准 | 零成本、公开后自动生效 |
| **Mermaid 网状架构图**（≤20 节点、分层 subgraph、至少一条回边） | ✅ 通用标准 | GitHub 原生渲染，可 diff，零图片维护 |
| 能力橱窗：核心模块/包一行一句话 + 一个 URL，禁止展开 | ✅ 通用标准 | 任何仓库适用（"能力"因仓库而异：kit/库模块/服务） |
| 30 秒 / 3 分钟 / persona 三层 README 结构 | ✅ 通用标准 | 产品思维不挑仓库 |
| **四象限演进观**（已知的已知=地基…四格 + 各格沉淀资产） | ✅ 通用标准 | owner 明示"所有仓库可以用上"；标准提供表格模板，各仓库填自己的格 |
| **复利仓库叙事**（越用越值钱一节） | 🔴 **不进标准** | owner 明示；它依赖内容寻址证据等 AiCoding 特有机制，别的仓库吹不起 |

## 跨仓库工作切分（子模块边界，踩过的坑不再踩）

标准文档与模板都在 **Codex-Skills 仓库**（AiCoding 里是只读子模块）。工作分两侧：

### A. Codex-Skills 侧（独立 PR，走该仓库流程）

1. `platform/aicoding-git-governance/references/aicoding-git-governance-standard.md`
   （canonical）升版（`version = "2026.08"` 或按其版本策略），新增章节：
   - **README 风格**：三层结构（30s/3min/persona）；徽章政策（技术栈投影三原则：
     全真、随栈增删、版本单源可核对）；能力橱窗（一行一 URL、与能力注册表一致、
     无幽灵行）；
   - **四象限演进节**（推荐而非强制）：README 或 vision 文档含四格表，
     模板给空表 + AiCoding 的填法作为示例链接。
2. `assets/REPOSITORY_GOVERNANCE_TEMPLATE.toml` 同步新增可配置面（示意，
   实际字段名服从标准文档）：

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
   保持向后兼容：老仓库不改 toml 行为不变。

### B. AiCoding 侧（子模块 pin 前移后）

1. 前移 `CodingKit/agents/skills` pin 到含新标准的提交（走既有 pin 前移评审）。
2. `.github/repository-governance.toml` 实例化新字段（AiCoding 选
   `tech-stack-projection` + showcase enabled + quadrants required——它是示范仓）。
3. `governance lint`（Go 侧）若需识别新 required_sections 值：确认其校验逻辑是
   读 toml 泛化校验还是硬编码清单——**泛化则零改动，硬编码则最小扩展**（先查后改）。
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
