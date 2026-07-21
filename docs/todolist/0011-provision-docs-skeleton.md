# TODO 0011: provision 建立 SDD 文档骨架（git-native 仓库初始化）

Status: Planned
Verify: 在空目录执行 bin/aicoding.exe provision --json 后，docs 骨架就位、二次执行零变更、git status 只含骨架文件

> 依赖关系：独立可做；与 0004–0006（Plan Mode）配合最佳——骨架为 plan/spec/todolist 预留位置。
> 你的直觉是对的：**`git init` 之于 .git/，就该有 provision 之于 docs/ ——
> 仓库出生时 SDD（Spec-Driven Development）流程的地基就该在。**
> 开源先例：`git init` 的 template dir（`~/.git-templates`）、`cargo new` 生成
> src+tests+Cargo.toml、Yeoman 脚手架。git-native 体现在两点：状态记在 git config
> （repoinit 已有的 aicoding.* 标记面），结构由 git 追踪而非旁路数据库。

## 背景

`aicoding provision`（ADR 0005，`internal/repoinit`）已做：git init 幂等、hooks 接线、
`aicoding.*` git-config 标记、`.aicoding` home。**没做**：docs 结构 —— 新仓库要用
docsync/plan-mode/todolist 流程，得手工照抄 AiCoding 的目录约定，抄错了门禁才告诉你。
这违反"约定优于配置"：约定应该被**放置**，不是被**描述**。

## 实现计划

1. **`internal/repoinit` 扩展 docs 骨架步骤**（保持编排定位——只组合，不含业务逻辑）：

   ```text
   docs/
   ├── README.md            hub 骨架（含 REPOSITORY_MAP 标记对，docsync 可接管）
   ├── architecture/README.md   分层阅读路径骨架（0007 模板的空壳）
   ├── decisions/           空目录 + .gitkeep 或 README（ADR 落点）
   ├── spec/                Plan Mode per-plan 目录落点（0005 约定）
   └── todolist/README.md   todolist Primitive 的目录说明（照抄 AiCoding 现有）
   ```

   原则：
   - **幂等且不覆盖**：任一路径已存在即跳过（Actions 里记 "kept"），绝不动已有内容；
   - **最小骨架**：每个 README 不超过 15 行，只写"这个目录归谁管 + 权威文档链接"，
     内容模板放 `config/templates/provision/`（与 0010 同一模板层）；
   - git-config 追加标记 `aicoding.docsSkeleton = 1`（沿用既有 markers 机制），
     `repoinit.Status` 一并汇报。
2. **`provision` 报告扩展**：`Report.Actions` 逐条列出 created/kept；新增
   `DocsSkeleton []string` 字段（omitempty，向后兼容）。
3. **与门禁的关系想清楚再接**：骨架 README 必须**天生通过** governance layout 与
   docsync（在 AiCoding 仓库自测；目标仓库无门禁时骨架只是普通文件）。
   若 layout 的 `allowedRootMarkdownFiles`/hub 规则与骨架冲突 —— **改骨架适配规则**，
   不改规则适配骨架。
4. **不越界**：provision 不写 plan-policy.json、不写 kit-registry、不装任何 kit ——
   那些属于"采用 AiCoding 平台"的 lifecycle 动作，不属于"初始化一个仓库"。
   骨架只准备**目录约定**，让后续 docsync/plan/todolist 有地方落脚。

## 明确不做

- 不做交互式选择（哪些目录要/不要 —— 全要，这是约定不是配置）。
- 不复制 AiCoding 的架构文档内容进目标仓库（骨架是空位，不是拷贝）。
- 不新增独立的 `docs init` 命令（provision 是唯一初始化入口，Do One Thing 的"one"
  是"让仓库就绪"，docs 骨架属于就绪的一部分）。
- 不做骨架版本升级/迁移机制（骨架是一次性放置，之后归仓库自己演进）。

## 自测（可信任方式）

```powershell
go test ./internal/repoinit/...
# 端到端（临时目录）：
$tmp = Join-Path $env:TEMP "provision-test-$(Get-Random)"
New-Item -ItemType Directory $tmp | Out-Null
bin\aicoding.exe provision --repo-root $tmp --json        # 首次：git init + 骨架，Actions 全 created
git -C $tmp config --get aicoding.docsSkeleton            # 期望 1
bin\aicoding.exe provision --repo-root $tmp --json        # 二次：幂等，Actions 全 kept，零变更
git -C $tmp status --porcelain                            # 只含骨架文件（首次后未提交状态）
# 不覆盖验证：改 $tmp/docs/README.md 内容 → 三次 provision → 内容未被动
Remove-Item -Recurse -Force $tmp

# 在 AiCoding 仓库本身跑（已有 docs）：provision 必须全 kept，git status 干净
bin\aicoding.exe provision --json ; git status --porcelain
bin\aicoding.exe governance layout --json ; bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Smoke --json
```

通过判据：空目录首次 created / 二次 kept 的幂等对（贴两次 Actions）；已有内容
绝不被覆盖（改后重跑验证）；在 AiCoding 自身跑零变更；骨架文件在 AiCoding 门禁下全绿。
