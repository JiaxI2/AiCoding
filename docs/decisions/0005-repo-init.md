# ADR 0005: repoinit —— `aicoding provision`（本地 AI-coding 环境初始化）

PrimitiveReview: required

## Status

Accepted。实现于 `internal/repoinit` + CLI `aicoding provision`。

## TL;DR

对标 `git init` 建 `.git/`（本地文件 + 后续命令靠它快速判断状态），`aicoding provision`
在仓库里一次性建立**本地 AI-coding 环境**：确保 git 仓库、接线 `.githooks`、把 AI-coding
标记写进 **git 自己的 config（`.git/config` 的 `aicoding.*` 命名空间）**、并确保 `.aicoding`
本地状态根存在。它不新增业务逻辑，只**组合**已有 Primitive（gitx、platform），幂等可重跑。

## Context

- 用户诉求：像 `git init` 那样在初始化时落一些本地文件/标记，后续命令用它们**快速判断
  或加速**（如 git 用 `.git/index`、`.git/config`）。
- 现状：hooks 接线（`core.hooksPath`）只在 kit 安装时发生（`internal/kit` 的
  `configurePlatformRepository`），bare clone + bootstrap 不接线；无任何 `aicoding.*`
  本地状态存储；`.aicoding/` 子目录按需 MkdirAll，无统一入口。

## Decision

新增读命令 `aicoding provision`，其领域 Primitive 为 `internal/repoinit.Init`：

1. **确保 git 仓库**：`.git` 缺失则 `git init`（经 gitx；幂等）。
2. **接线 hooks**：`git config core.hooksPath .githooks`——激活 pre-commit/commit-msg/
   post-commit（用 git 自带机制，不重造 hook 发现）。
3. **写 AI-coding 标记进 `.git/config`**：`aicoding.initialized/home/schemaVersion`。这就是
   "混在 git 文件夹下"的**安全实现**——`.git/config` 是 git 自己的本地、per-clone、不提交的
   配置文件；后续命令用 `git config --get aicoding.*` **瞬时**读取，无需扫描工作树。
4. **确保 `.aicoding` 本地状态根**存在（其版本化子树按需创建且 gitignore，provision 只保证根）。

配套只读 helper `repoinit.Status(repo)`：从 git config 读标记，供后续命令快速判断
"是否已 provision"而不扫描。

**关键设计取舍**：

- **不往 `.git/` 里塞松散文件**（`.git/objects` 等由 git 管理，塞文件脆弱、会被 gc/忽略）。
  本地 KV 存储用 `.git/config` 的 `aicoding.*` 命名空间——git 原生、安全、可 `git config` 读写。
- **命令名不能叫 `init`**：`init` 是 git porcelain 保留动词，[GIT_REUSE_BOUNDARY](../architecture/GIT_REUSE_BOUNDARY.md)
  §9 + `catalog_test.go` 禁止 CLI 用 git 动词名（避免遮蔽 git）。故命令名 `provision`；
  内部仍可经 gitx 跑 `git init`（正如 `fresh-clone` 内部跑 `git clone` 却不叫 `clone`）。
- **组合而非合并**：`provision` 不建二进制（那是 `bootstrap` 的单一职责）；两者组合使用。
  `bootstrap` 按边界不是 git 调用方，故 git 相关全在 `repoinit`（已加入
  `gitProcessBoundary.allowedImporters`）。

## §12 Checklist 自评（Primitive 宪法）

**架构**
- 单一职责？是——`repoinit.Init` 只做"建立本地 AI-coding 环境"这一件事的编排。
- 可继续拆分？否——四步都是幂等的最小 git/fs 调用。
- 能被直接复用？是——`Init`/`Status` 是纯函数，任何命令可调用（Status 供后续命令判断）。
- 存在重复实现？否——hooks 接线逻辑与 kit 的 `configurePlatformRepository` 都用同一 git
  机制；未来若第二处也需接线，可把该行抽为共享 helper（当前两处足够简单，不预抽）。
- 真的需要新 Primitive？是——现无"本地环境初始化 + git-config 状态存储"能力。

**性能**
- Fast Path？是——纯 git config/init 调用，**零工作树扫描**；Status 读 git config 瞬时。
- 无关扫描 / 重复 IO / 重复计算？无。
- 最小输入/输出？是——输入 `repo`，输出标准 `Report`。
- 减少进程启动？provision 是一次性初始化；后续命令用 `git config --get` 单次读取判断状态。

**质量**
- 确定性？幂等——重跑不改变结果（`TestInitIsIdempotentAndGitNative`）。
- 接口稳定？新增命令与包，不改任何既有接口；命令名避开 git porcelain 保留字。
- 独立测试？是——`internal/repoinit` 全套单测（幂等、git-native 读写、未初始化态）。
- 自由组合？是——与 `bootstrap`（构建）、`doctor`（可读 Status 判断）组合。

## Rollback

删除 `internal/repoinit`、CLI catalog 的 `provision` 行 + `runProvision`、
`dependency-governance.json` 中 repoinit 的 gitx 允许项；内核零改动。
`.git/config` 的 `aicoding.*` 与 `core.hooksPath` 可用 `git config --unset` 清除。
