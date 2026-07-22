# TODO 0026: 内容钉死的引用注册（不 vendoring 只登记 + 快速导入）

Status: Planned
Verify: 用一个 pinned git 引用注册一个外部 skill/kit（不复制其源码），kit list 立即可见（<300ms），lifecycle install 本地 materialization <10s，且导入内容可被 validation evidence 绑定

> 来源：owner 的两阶段模型 —— "注册（仓库知道有这么个东西，不改源码、快）→
> 导入（仓库把它交给 agent，快）"。
> **评审结论：模型正确，架构 80% 已就位，唯一缺口是"不 vendoring 只登记任意外部能力"。**
> 本项补这一块，是迭代支持 initiative 的第一块（先于 overlay 层）。

## 一、现状确认（实测，2026-07-22）

| owner 要的 | 现状 |
|---|---|
| 两阶段分离（注册 / 导入） | ✅ 已有：`kit-registry.json`（注册）vs `lifecycle install`（导入） |
| 注册基于仓库不基于 agent | ✅ 已有且正确：registry 是仓库 config，agent 不参与 |
| 注册快 | ✅ `kit list` 100ms；加条目 = JSON 编辑 |
| 导入快 | ✅ `lifecycle plan` 6ms；install 写本地 state |
| skill/mcp 视为统一 kit | ✅ 既有抽象 |
| **不改源码、纯引用注册任意外部能力** | ⚠️ **缺** —— 见下 |

**缺口根因**（`internal/kit/structure.go:241`）：

```go
if !platform.IsFile(platform.RepoPath(repo, skill.Path)) {  // 资产必须已在场
```

今天注册一个 kit，要求资产**要么已 vendored，要么是 pinned submodule**（runtime-skill 路径②）。
"指向外部、登记时不要求在场"不存在。

## 二、为什么不能 naive 地做"纯路径引用"（本项的核心约束）

**如果注册只记一个会变的路径、不 pin 内容**，则：
- validation evidence 无法绑定它的内容（无 Tree OID / digest）→ Receipt 体系对它失效；
- 单一权威被破坏（引用指向可变外部 = 没有唯一权威源）；
- 门禁无从判断"这次导入的和上次注册的是否同一物"。

**纯引用 = 快，但把内容身份纪律整个架空 —— 正是本仓库一路拒绝的"图快牺牲正确性"。**

## 三、正确设计：content-pinned reference（既快又保住不变量）

注册的不是路径，是**被内容钉死的引用**。**复用既有 pin 机制，不重造**：
`internal/lifecycle/runtime_skill.go` 的 `runtimeSkillInputDigest` 已经用
`git rev-parse HEAD` 把源钉到 commit —— 本项把它上升为 kit manifest 的一等能力。

```text
注册（register）
  manifest 增加可选 source 字段（内容钉死）：
    "source": { "kind": "git", "url": "...", "commit": "<40-hex-sha>" }
    或
    "source": { "kind": "content", "digest": "sha256:..." }
  仓库知道"有这么个东西"且知道其确切内容身份 —— 不 vendoring、不要求在场。
  代价：JSON 编辑 + 一次 pin 校验（commit 是否为不可变引用）。~ms 级。

导入（import to agent）
  = 既有 lifecycle install 的 materialization（junction / archive / copy）
  从 pin 物化到 agent 可见位置。因 pin 是内容钉死的，
  物化内容可被 validation evidence 绑定 —— 快、可验、可回滚。
```

## 四、性能设计：网络成本移到注册时（保证导入永远 <10s）

**pin 在注册时后台预取到本地 content-addressed 缓存，import 只做纯本地 materialization。**
网络成本花在"注册后"这个用户不等待的时刻；导入永远是本地操作 → 永远 <10s。

```text
register → 后台 prefetch pin 到 <git-common-dir>/aicoding/pins/<digest>/
import   → 从本地 pin 缓存 materialization（零网络）
若 import 时 pin 未预取完成 → 返回 category=evidence-missing +
          requiredAction（提示等待预取或显式 prefetch），
          绝不在 import 路径里静默 network fetch（那会破坏 <10s 承诺）
```

## 五、实现计划

1. **ADR + plan approve（前置，不可跳过）**：
   加 `source` 字段动的是 `config/schemas/kit-manifest.schema.json` ——
   **冻结面**（`additionalProperties:false`，且 config/schemas/** 在 plan-policy 敏感路径）。
   必须先 `plan check --staged` 命中 → 建 plan → `plan approve` 绑 tree → ADR 0010
   论证它与 06-plugin-sdk §6 doctrine（定制流经输入、不进 owned 资产）一致。
   **schema 只增可选字段，向后兼容**（老 manifest 无 source 字段照常有效）。

2. **`internal/kit` 增 pinned source 解析**（复用 runtime-skill 的 pin digest 逻辑）：
   - `source.commit` 必须是 40-hex 不可变 commit（**拒绝 branch/tag 等可变引用**）；
   - `source.digest` 必须是 `sha256:` content hash；
   - pin 校验是纯函数，不 fetch（fetch 归 prefetch）。

3. **`structure.go:241` 的 verify 放松（有界）**：
   skill/asset 路径校验从"必须是本地文件"改为
   **"本地文件存在 OR source pin 已解析且已预取"** ——
   **绝不放松成"路径不需要存在"**（那会破坏身份）。

4. **注册面 CLI**（不新增命令域，扩展既有）：
   - `kit register --manifest <path> [--prefetch] --json`：校验 pin + 写 registry +
     可选后台预取。或直接沿用"编辑 registry + kit verify"，`kit register` 只是便捷封装。
   - `kit prefetch --id X --json`：显式把 pin 拉到本地缓存（网络成本在此，用户可选后台）。

5. **导入面**：`lifecycle install` 的 kit adapter 增加"从 pin 缓存 materialization"分支；
   pin 未预取时 fail-closed 给 requiredAction，不静默 fetch。

6. **pin 缓存纳入 0024 的 cache 治理**：`<git-common-dir>/aicoding/pins/` 作为
   cache 的第六个 scope（内容寻址，可安全删除重取；被 registry 引用的不删）。

7. **同步**：ADR、COMMANDS.md、HelpForm、`docs/architecture/06-plugin-sdk.md`
   （在路径①/②之间补一条"pinned reference 注册"的说明）。

## 六、明确不做

- **不做纯路径引用**（无 pin）—— 破坏内容身份，本项存在的全部意义就是不这么做。
- **不在 import 路径静默 network fetch** —— 网络成本只在 register/prefetch。
- **不放松"路径可以完全不存在"** —— 只放松成"本地在场 OR pin 已解析预取"。
- 不 vendoring 外部源码进仓库（那是本项要替代的旧方式）。
- 不改单一 skill 权威规则 —— pin 就是权威（内容钉死），不是第二个可变源。
- 不为此新建 registry —— 复用 kit-registry + manifest 的 source 字段。
- 不动 runtime-skill 路径②（它是 submodule pin，与本项的 kit-manifest pin 并存，
  两者都是"内容钉死"的合法形态；ADR 说明二者边界）。

## 七、自测（可信任方式，负例必须真跑）

```powershell
go test ./internal/kit/... ./internal/lifecycle/... ./internal/cache/...

# 正例：pinned git 引用注册一个外部 skill（不复制源码）
bin\aicoding.exe kit register --manifest testdata/kit/pinned-external.json --prefetch --json
bin\aicoding.exe kit list --json                    # 立即可见
1..5 | % { (Measure-Command { bin\aicoding.exe kit list --json }).TotalMilliseconds }  # <300ms
bin\aicoding.exe lifecycle install --scope kit --kit pinned-external --json  # <10s（本地）
bin\aicoding.exe validation check --profile Smoke --target HEAD --json       # 导入内容可绑定

# 负例（逐条贴输出）：
#  1) source.commit 写 branch 名（可变引用）→ 必须拒绝
#  2) source.commit 指向不存在的 sha → fail-closed
#  3) 无 pin 的纯路径 source → schema 拒绝
#  4) import 时 pin 未预取 → category=evidence-missing + requiredAction，不静默 fetch
#  5) 老 manifest（无 source 字段）→ 照常有效（向后兼容断言）
#  6) 改 source.commit → validation identity 变 → 旧 Receipt 正确失效

# 性能边界证明：import 全程无 network 调用（进程/网络计数器断言 = 0）
bin\aicoding.exe cache status --json                 # pins scope 出现
bin\aicoding.exe governance dependencies --json ; bin\aicoding.exe test --profile Full --json
```

通过判据：
1. 用 pinned 引用注册**不复制任何外部源码**（`git status` 无新增 vendored 文件）。
2. 注册 <300ms、导入 <10s（本地）、import 路径 network 调用 = 0（负例4断言）。
3. 六条负例全部被抓。
4. 导入内容可被 validation evidence 绑定（Receipt 生成成功）。
5. 改 pin = 新身份 = 旧 Receipt 失效（内容身份纪律完好）。
6. 老 manifest 向后兼容（schema 只增可选字段）。
7. 单一权威未破坏：pin 是唯一内容源，无可变第二源。
8. ADR 论证了与 06-plugin-sdk §6 doctrine 的一致性。
