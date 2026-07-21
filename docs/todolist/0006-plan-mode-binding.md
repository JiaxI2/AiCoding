# TODO 0006: Plan Mode 重构 III —— 批准绑定内容（approve + 漂移检测 + ADR 0009）

Status: Done
Verify: approve 后 amend-only 提交不产生漂移告警、scope 内实改一文件即产生漂移告警（两条端到端用例）

> 依赖 0004 + 0005。回答"文档和实际代码会不会同步"：**批准的不再是一段文字，是一棵树。**

## 实现计划

1. `aicoding plan approve --id X --json`：
   - 前置：status ∈ {draft, needs-decision}；OPTIONS 存在则 DECISION 必须存在；
     工作区干净（复用 `gitx.StatusSnapshot`，dirty 即拒绝 —— 批准一个漂着的树没有意义）。
   - 动作：写 frontmatter `approvedTree = HEAD^{tree}`（复用 `gitx.TreeOID`），
     status → approved。这是 plan 域**唯一的写命令**，只改该 plan 的 PLAN.md。
2. `plan status --id X` 升级为漂移裁决（复用 gitx，单次 diff）：

   ```text
   changed = git diff --name-only <approvedTree> HEAD^{tree}
   漂移    = changed ∩ scope        → 列出，提示"实现进行中/需复核 plan"
   越界    = changed ∖ scope ∖ exempt → 警告（检测式，与 loop 同一纪律——
                                        AiCoding 阻断的是下一步，不是已发生的写入）
   完成建议 = scope 全覆盖 且 gates 的 profile 在 validationevidence 有当前树的
             有效 Receipt（复用 validation check 语义）→ 建议 status: implemented
   ```

3. `plan check --staged`（0004）升级：命中敏感路径时查是否存在 approved plan 的
   scope 覆盖这些路径 —— 覆盖则放行并回显 plan id，不覆盖则 fail。
   pre-commit 由 warn 升 **enforce**。
4. **ADR 0009 `docs/decisions/0009-plan-mode-rework.md`**（`PrimitiveReview: required` +
   `## §12 Checklist 自评`），必答：
   - plan/loop/todolist/validationevidence 四者边界（一张图）；
   - 为什么触发在 pre-commit 而非 agent-hook（自觉→强制）；
   - 为什么批准绑定 Tree 而非 commit（amend/rebase 语义，与 ADR 0007 同理）；
   - ps1 面退役路径（薄壳 → deprecated → 删除的 release 节奏）；
   - 检测式权限的诚实声明。
5. 新增 `docs/architecture/PLAN_MODE_ARCHITECTURE.md`（沿用 LOOP 模板骨架：
   结论→系统位置→边界判据→数据契约→可靠性与安全→演进边界），Status: Accepted，
   替换 ADR 0002 描述的旧 ps1 流程（ADR 0002 追加 Superseded-by 注记，不删）。

## 明确不做

- 不自动把 status 改成 implemented（只建议，人确认）—— plan 是契约不是自动机。
- 不做跨 plan 依赖图、不做 plan 模板生成器（无第二消费者前不加）。
- 不在 hook 里跑任何验证 profile。

## 自测（可信任方式）

```powershell
go test ./internal/plan/... ; go vet ./...

# 端到端（临时分支上做，完后删）：
# 1) 建 plan → approve → 记录 approvedTree
# 2) git commit --allow-empty（或 amend message-only）→ plan status 无漂移（tree 未变）
# 3) 改 scope 内一个文件并提交 → plan status 列出该文件为漂移
# 4) 改 scope 外一个文件并提交 → plan status 报越界警告
# 5) plan check --staged 对"敏感路径 + approved plan 覆盖"放行、对无覆盖 fail

# 性能：plan status 单 plan 中位数 < 300ms（一次 diff + 一次 evidence stat）
1..5 | % { (Measure-Command { bin\aicoding.exe plan status --id <X> --json }).TotalMilliseconds }

bin\aicoding.exe test --profile Full --json          # 含 ADR-001 门禁（0009 有 §12 锚点）
bin\aicoding.exe governance dependencies --json
```

通过判据：上面 5 条端到端全部符合预期；ADR 0009 过 ADR-001 门禁；
`grep -rn "type Receipt" internal/` 仍只有 validationevidence 一处（plan 只持引用）；
pre-commit enforce 后，无 plan 的敏感变更被真实拦截（贴 hook 输出）。
