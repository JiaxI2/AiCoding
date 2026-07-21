# TODO 0020: 仓库自带创作模板（skill / mcp / hook 脚手架，复用 kit init 机制）

Status: Planned
Verify: bin/aicoding.exe skill init demo-skill --dry-run --json 与 mcp init demo-mcp --dry-run --json 输出合规骨架；生成物零编辑过对应 verify 门禁

> **本项是对 owner"仓库自带 skill/plugin kit"想法的审核结论**：方向对，但要拆成
> **可采纳的一半**和**必须拒绝的一半**，且**排在收敛期之后**。
> 依赖 0010（`kit init` 模板机制先落地，本项只是加模板家族，不新建机制）。

## 审核：想法拆成三块

| 你的想法 | 裁决 | 理由 |
|---|---|---|
| **建立新的符合仓库的 skill/mcp 更方便**（脚手架） | ✅ **采纳** | 这是 0010 `kit init` 的同类需求。加模板家族 ≠ 加新系统 |
| **方便自定义修改**（skill 源在只读子模块里改不动） | ⚠️ **部分采纳，走 overlay 不走复制** | 见下"为什么不能自带第二套 skill" |
| **仓库自带一批 skill/plugin kit**（内置能力包） | 🔴 **拒绝** | 违反单一 skill 权威；且与当前收敛目标相反 |

### 为什么不能"自带第二套 skill"

`AGENTS.md` 与 `docs/ARCHITECTURE_OVERVIEW.md` 都明写：

> AiCoding **不拥有嵌入式 skill 源码**。权威 skill/plugin 源位于
> `CodingKit/agents/skills` 子模块（Codex-Skills）。

仓库内再放一套 skill，立刻产生：**两个 skill 源 → 同名冲突 → runtime 审计拒绝 →
"改哪份才生效"无人说得清**。这正是 Plan Mode 那两套平行产物目录踩过的坑
（0005 刚花力气收敛掉）。

**"方便修改"的正解不是复制，是 overlay**：Codex-Skills 侧改源 → pin 前移；
本仓库只做**登记与投影**（既有路径①②，见 `06-plugin-sdk.md`）。
真要本地试改，用 worktree 直接在子模块里开分支，改完走上游 PR ——
这条路已经通，只是没写进文档（本项补一页指引即可）。

### 时机判断（比技术判断更重要）

你自己说的："现在还是收敛仓库、固化核心能力、展示产品神力的环节。"
**我同意，并建议本项整体排在 0009–0019 之后。** 理由：

- 脚手架的价值随"有多少人要造新 skill"增长，当前是单人 + 收敛期，收益接近零；
- 收敛期加创作入口，等于一边收口一边开口；
- 但**现在就把它立项写下来**是对的——避免将来临时起意时又发明一套新机制。

## 实现计划（0010 落地后再启动）

1. **模板家族扩展**（`config/templates/` 下与 kit 并列，同一机制）：

   ```text
   config/templates/
   ├── kit/          （0010 已建）
   ├── skill/        SKILL.md frontmatter 骨架 + 章节契约（When to use / Workflow /
   │                 Verification / Constraints——PLUGIN_STANDARD §5 的稳定章节）
   ├── mcp/          MCP component manifest 骨架（对齐 config/schemas/mcp-component.schema.json）
   └── hook/         hook 薄壳骨架（禁跑测试/禁构建/fail-closed 三条内建）
   ```

2. **命令按需增加，不一次全上**：
   - `aicoding skill init <id> --dry-run|--json`：生成 SKILL.md 骨架到
     **目标路径由参数指定**（默认打印到 stdout，不落盘到子模块——
     避免误写只读子模块，这是硬约束）；
   - `aicoding mcp init <id>`：生成 component manifest + registry 条目建议
     （**只在出现第二个真实 MCP 组件需求时才实现**，否则纯属为一个场景造轮子）；
   - `hook init` 暂不做（4 个 hook 已覆盖 git 全部触发点，无新增需求）。
3. **一页创作指引** `docs/guides/AUTHORING.md`：
   - 改既有 skill → 在子模块开分支 → 上游 PR → pin 前移（三步，配命令）；
   - 造新 skill → `skill init` → 放 Codex-Skills → 登记 → pin 前移；
   - 造新 kit → `kit init`（0010）；
   - **一张表说清"改什么去哪改"**，终结"skill 到底在哪"的困惑。
4. 门禁：生成物必须过对应既有校验（skill 过 `skill verify`，
   mcp 过 `mcp verify`），沿用 0010 的"生成即合规"承诺与模板-schema 同步测试。

## 明确不做

- **不在 AiCoding 仓库内放任何 skill 源码**（单一权威铁律）。
- 不做 skill 市场/远程模板/模板版本管理。
- 不做 `hook init`（无需求）。
- `mcp init` 在第二个真实组件出现前不实现。
- 不改 `CodingKit/agents/skills` 子模块的只读定位。

## 自测（可信任方式）

```powershell
go test ./internal/kit/... ./internal/cli/...
bin\aicoding.exe skill init demo-skill --dry-run --json     # 骨架含 PLUGIN_STANDARD §5 全部章节
bin\aicoding.exe skill init demo-skill --out $env:TEMP\demo --json
bin\aicoding.exe skill verify --all --profile Smoke --json  # 既有 skill 未受影响
# 硬约束负例：尝试把 skill init 输出写进 CodingKit/agents/skills → 必须拒绝（贴错误）
git status --porcelain                                       # 子模块零变更
bin\aicoding.exe governance layout --json ; bin\aicoding.exe test --profile Full --json
```

通过判据：生成物零编辑过 `skill verify`；**写入只读子模块被拒绝**（核心安全断言）；
`docs/guides/AUTHORING.md` 的"改什么去哪改"表覆盖 skill/kit/mcp/hook 四类；
全仓 skill 源仍只有子模块一处（`find . -name SKILL.md -not -path "./CodingKit/*"` 为空）。
