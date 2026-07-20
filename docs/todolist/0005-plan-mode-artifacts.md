# TODO 0005: Plan Mode 重构 II —— 产物标准化（per-plan 目录 + frontmatter + plan verify）

Status: Planned
Verify: bin/aicoding.exe plan verify --json 全绿，且两个历史会话已迁入 docs/spec/<id>/ 归档

> 依赖 0004（internal/plan 包已存在）。
> 现状实锤：`docs/spec/PLAN_MODE.md` 躺着 "product-convergence" 会话，
> `docs/decisions/plan-mode-overlay/PLAN_MODE.md` 躺着 "C UserStyle 1.2.0" 会话 ——
> **产物是全局单槽固定路径，每开新会话覆盖上一个，两套目录互为平行事实源。**

## 实现计划

1. 目录标准（一个 plan 一个目录，永不互相覆盖）：

   ```text
   docs/spec/<plan-id>/
   ├── PLAN.md        唯一必需，frontmatter 机器可读
   ├── OPTIONS.md     可选：多方案对比（原 PRD_OPTIONS）
   ├── DECISION.md    可选：用户裁决记录（原 SELECTED_SOLUTION）
   └── TASKS.md       可选：任务拆解
   ```

2. `PLAN.md` frontmatter schema（新增 `config/schemas/plan-spec.schema.json`）：

   ```yaml
   ---
   id: <kebab-case，与目录名一致>
   status: draft | needs-decision | approved | implemented | archived
   scope:
     - internal/loopkit/**
   approvedTree: ""            # approve 时由 CLI 写入（0006），人不手填
   decision: docs/spec/<id>/DECISION.md   # OPTIONS 存在时必填
   gates: [{ profile: full }]  # 完成判据，指向 validationevidence profile
   ---
   ```

   frontmatter 读取**复用 `internal/todolist` 的头部解析模式**（同一 Primitive 思路的
   第二个消费者：只读一个目录、只解析头部、确定性排序）——但不共享代码强扭；
   若抽公共函数则放 `internal/plan`，todolist 不动。
3. `internal/plan` 增加：
   - `ListSpecs(repo)`：枚举 `docs/spec/*/PLAN.md`，解析 frontmatter，按 id 排序。
   - `VerifySpecs(repo)`：schema 校验 + 约束检查（id==目录名 / status 合法 /
     scope 非空且 glob 可解析 / OPTIONS 存在时 DECISION 必须存在 /
     approved 状态必须有非空 approvedTree —— 0006 前允许空但报 warning）。
4. CLI `aicoding plan verify --json` 与 `aicoding plan status [--id X|--all] --json`
   （status 本项只报 frontmatter 视图，漂移检测留给 0006）。同步 HelpForm + COMMANDS.md。
5. **迁移两个历史会话**（读旧文件内容，归档不删史）：
   - `docs/spec/*.md`（product-convergence 那套 8 文件）→
     `docs/spec/product-convergence/`，status: archived。
   - `docs/decisions/plan-mode-overlay/*` → `docs/spec/c-userstyle-kit-1-2-0/`，
     status: archived。原目录留一个 `README.md` 指向新位置（一个 release 后删）。
   - `config/agent-dev-kit-plan-mode.registry.json` 的 phases artifact 路径改为
     per-plan 模板（`docs/spec/<id>/…`）。
6. docsync / governance layout 白名单相应更新（`docs/spec/<id>/` 目录结构入法）。

## 明确不做

- 不实现 approve 语义与 treeOID 写入（0006）。
- 不做 markdown 正文语义校验 —— 只校验 frontmatter 与文件存在性。
- 不删除历史内容，只迁移归档。

## 自测（可信任方式）

```powershell
go test ./internal/plan/... 
# 单测覆盖：合法 spec 通过 / id≠目录名 fail / status 非法 fail / OPTIONS 有而 DECISION 无 fail
# / frontmatter 缺失 fail-closed / 两次 ListSpecs 输出字节一致（确定性）

bin\aicoding.exe plan verify --json                  # 全绿（含迁移后的两个 archived 会话）
bin\aicoding.exe plan status --all --json            # 列出 2 个 archived
bin\aicoding.exe docsync all --json                  # 迁移后文档索引一致
bin\aicoding.exe governance layout --json
bin\aicoding.exe test --profile Full --json
git status --porcelain                               # 除本项变更外干净
```

通过判据：plan verify 对故意写坏的 fixture（testdata/plan/ 下放坏例）逐条报错；
两个历史会话在新位置可被 `plan status` 列出；旧固定路径不再被任何 registry/脚本引用
（`grep -rn "plan-mode-overlay" config/ tools/` 只剩 deprecated 标记）。
