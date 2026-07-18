# Git 复用边界

Status: Proposed（验收通过后改为 Accepted and Frozen）

## 1. 结论

AiCoding 最大限度复用 Git，不包装 Git porcelain。Git 是整个体系脚下的事实层（L0），
不是第四个 adapter 领域。AiCoding 只处理 Git 无法表达的运行时领域。

判断任何一个功能归属，用三个问题：

1. **这个事实能否由某个 commit 完整重建？** 能 → Git 域。
2. **这个操作的并发保护与回滚语义是否就是 ref 移动？** 是 → Git 域。
3. **是否涉及本机可变资源（junction、venv、Codex 配置、plugin cache、进程）？** 是 → AiCoding 域。

由此得到运行时领域的正式定义：

> AiCoding 运行时领域 = 任何不能由 `git checkout` 重建、也不随 commit 移动的状态。

## 2. 分层位置

```text
L4  调用方        Agent / Skill / CI / Hook             （类比：GUI、IDE 插件）
L3  porcelain    aicoding CLI typed commands            （类比：git add / commit / push）
L2  领域          kit / mcp / runtime-skill              （类比：refs 与 index 的领域规则）
L1  plumbing     snapshot / plan / runner / report      （类比：hash-object / write-tree / update-ref）
L0  事实层        Git 本体：对象库、refs、传输、gitlink    ← 不包装，直接用
```

关键推论：L1 对 L0 是**消费者**关系，不是**适配**关系。Kit/MCP/runtime Skill 各有
adapter，但永远不存在 `git` adapter——adapter 的定义是"把统一 action 翻译到领域"，
而 Git 域的 action 就是 git 命令本身，翻译层是零价值包装。

## 3. 完全交给 Git 的能力

| 能力 | Git 权威 | AiCoding 的角色 |
|---|---|---|
| 工作区、暂存区 | `git status` / `git diff` / `git add` | 只读观察（pre-commit 读 staged files） |
| 历史与分支 | `git commit` / `branch` / `merge` | 无 |
| 对象与内容树 | blob / tree / commit / gitlink | 无 |
| 版本身份 | commit / tree OID | 消费者（见 §6） |
| 并发保护 | expected-old / `update-ref` / push 拒绝 non-fast-forward | 无 |
| 远程同步 | `fetch` / `pull` / `push` | 无 |
| 回滚声明 | `git revert` | 无（见 §7） |
| 子模块 source pin | gitlink + submodule | 维护流程按文档使用裸 git 命令 |

对应的禁令：AiCoding CLI 不提供任何与上表语义重复的命令或别名。这条禁令由
command catalog 契约测试固化（见 §9）。

## 4. AiCoding 保留域

| 保留域 | 为什么 Git 无法表达 |
|---|---|
| install state（`.aicoding/state/**`） | 本机的、随安装动作变化，不随 commit 移动 |
| exposure（junction、plugin cache、`~/.agents/skills`） | 用户目录下的软链与缓存，checkout 无法重建 |
| 外部运行时（venv、package、Codex managed block） | 进程与第三方配置文件，不在仓库对象库里 |
| 生命周期意图与证据（plan/apply、digest 三元组、report JSON） | Git 记录"是什么"，不记录"打算做什么、做成没有" |
| 所有权与验证（ownership 检查、doctor/verify profile） | 需要对比仓库事实与本机现实，Git 只有前者 |

八个统一 action（plan/install/update/uninstall/status/doctor/verify/rollback）是这五类
之上的固化动词表，已由 [Extension Adapter Contract](EXTENSION_ADAPTER_CONTRACT.md) 冻结。

## 5. gitx 宪法

`internal/gitx` 是 AiCoding 与 Git 之间唯一的进程边界，规则：

1. **唯一入口**：生产代码中 git 进程只能由 `internal/gitx` 启动。其他任何
   `internal/*`、`cmd/*` 生产文件不得直接 `exec.Command("git", ...)`。测试文件除外。
2. **薄封装**：gitx 只允许"启动进程 + 解析输出为 Go 值"（如 `StagedFiles`）。
   不允许语义封装——任何"替用户决定 git 语义"的函数（如自动 commit、自动 push）
   都被禁止。判据：删掉 gitx 后，用户用裸 git 命令应能做完全相同的事。
   解析 helper 必须**领域无关**：函数名与返回类型不得出现 kit/mcp/skill/lifecycle
   等消费者语义（`StagedFiles` 合法；`SkillSourceCommit` 违宪）。领域语义的组合
   属于消费者自己的包，gitx 不做任何领域的共享工具箱。
3. **默认只读**：允许的读操作包括 `status`、`diff --cached`、`rev-parse`、
   `submodule status` 等观察类命令。写 git 状态的操作只允许出现在**已登记流程**中：
   - fresh-clone 验证流程的 `clone --recursive`；
   - hook 安装流程的 `config core.hooksPath`；
   - 文档写明的跨仓库 gitlink 维护流程。
4. **零依赖**：gitx 不 import 任何其他 `internal/*` 包，保持纯 L1 工具地位。
5. **importer 白名单**：允许 import gitx 的包由 `config/dependency-governance.json`
   显式登记；新增 importer 必须修改登记并说明用途。

## 6. 版本身份分工

Git OID 与 AiCoding digest 回答不同问题，都保留，不合并：

| 摘要 | 覆盖内容 | 回答的问题 |
|---|---|---|
| Git commit/tree OID | 原始字节 + 历史 | 仓库在某一时刻的确切内容 |
| `catalogDigest` / `inputDigest` / `planDigest` | 规范化后的语义事实 | 对哪组事实执行了什么意图 |

AiCoding digest 覆盖未提交的 worktree 状态且跨机器稳定，因此不能用 OID 替代；
OID 绑定提交历史，因此不能用 digest 替代。"在 run evidence 中记录 HEAD OID"是
已识别的未来出口，在真实消费者出现前不实现。

## 7. 并发与回滚

- **Git 域并发**交给 ref 语义：`update-ref` 的 expected-old、push 的
  non-fast-forward 拒绝。AiCoding 不重复实现。
- **AiCoding state 并发**：`.aicoding/state/**` 是本机 JSON，不受 ref 语义保护。
  在真实跨进程并发场景出现前保持单进程串行假设，不预建 CAS
  （与 [CLI 与 MCP 控制面](CLI_MCP_CONTROL_PLANE.md) §6.1 一致）。
- **两种回滚永不合并**：`git revert` 回滚仓库声明的事实；领域 rollback 回滚本机
  运行时状态。完整回滚 = git revert（事实层）+ 对应领域 rollback（运行时层），
  由调用方（Skill）编排。永远不提供合并的 `rollback --all`。

## 8. 同名不同义的既有命令

以下既有命令与 git 动词同名或相近，但语义属于 AiCoding 领域，不是 git 包装，
予以保留并豁免于 §9 的禁用清单：

| 命令 | 语义 | 与 git 的区别 |
|---|---|---|
| `status` | lifecycle/领域状态对账 | 不是 `git status` 的工作区状态 |
| `tag` | tag 策略审计与对齐计划（specialty） | 不是 `git tag` 的创建/删除 |
| `fresh-clone` | 已登记验证流程，经 gitx 调用 `git clone` | 是流程编排，不是 clone 别名 |

未来任何新的同名命令必须先通过 ADR 论证其语义不与 git 重复。

## 9. 可执行门禁

本边界由三条门禁固化，不依赖人工守约：

1. **git 进程所有权**（governance dependencies）：扫描 `internal/`、`cmd/` 生产
   Go 文件，`internal/gitx` 之外出现以字面量 `"git"` 启动进程即失败。
2. **gitx 依赖边界**（governance dependencies）：gitx 的 forbiddenImports 覆盖全部
   其他 internal 包；import gitx 的包必须在 allowedImporters 登记内。
3. **porcelain 动词禁用**（command catalog 契约测试）：command catalog 的命令名与
   alias 不得出现 git porcelain 动词：
   `add, am, apply, bisect, blame, branch, checkout, cherry-pick, clone, commit,
   diff, fetch, init, log, merge, pull, push, rebase, remote, reset, restore,
   revert, show, stash, submodule, switch, worktree`。

### 9.1 验证半径

与核心架构 §9 同构：局部修改跑局部验证，只有跨模块公开契约变化才触发全量。

| 变化 | 最小必跑 | 扩大条件 |
|---|---|---|
| gitx 内部实现（进程处理、错误格式） | `internal/gitx` tests | `Run`/helper 签名或输出解析契约变化时跑全部 importer 的包测试 |
| gitx 新增领域无关解析 helper | gitx tests + 首个消费者包测试 | 无 |
| gitx importer 增减 | `governance dependencies` + 登记更新 | 无 |
| 门禁 check 实现（ownership/allowlist） | `internal/governance` tests + CLI fixture | check 名或 JSON 报告字段变化时跑 CLI contract + Full |
| porcelain 禁用集合增减 | catalog test + 本文档同步修改 | 集合缩小（放行动词）必须走 ADR + Full |
| 新增已登记写流程（§5.3） | 该流程所属领域测试 + `governance dependencies` | 始终同步更新本文档登记 |

## 10. 明确拒绝

- 任何 git porcelain 命令的 AiCoding 别名或"增强版包装"；
- `git` adapter、GitService、RepoManager 等把 Git 当作领域适配的抽象；
- gitx 中的语义封装（自动 commit/push/merge 决策）；
- 把 install state、junction、venv 提交进 Git 仓库；
- 用 AiCoding digest 替代或重新实现 Git OID，反之亦然；
- 跨进程 state CAS、全域事务、合并式 `rollback --all`；
- 在 run evidence 中预建 HEAD OID 字段（无消费者时）。

## 11. 冻结与解冻

本边界与六模块架构同级冻结。解冻沿用仓库停止规则：只有**现实问题 + 稳定变化点 +
至少两个真实消费者**同时出现，才通过 ADR 修改本文档对应章节。新增 Git 交互
（新的读观察或新的已登记写流程）不属于解冻，按 §5 修改登记并通过门禁即可。
