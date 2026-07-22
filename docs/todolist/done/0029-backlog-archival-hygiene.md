# TODO 0029: backlog 归档与队列卫生（Done 项归档 + ps1 退役窗口显性化）

Status: Done
Verify: aicoding todolist --json 只列活跃项（Done 归档后 total 大幅下降）；done/ 归档完整可查；doctor pwsh 输出每个薄壳的退役触发条件

> 来源：FORWARD_PLAN C4 + C3。现状：docs/todolist/ 27 项里 26 个 Done ——
> todolist Primitive 是"待办队列"，不是"已完成档案馆"，**队列该短**。
> ps1 侧：0002 裁决"22 个脚本各有退役窗口"，但窗口是隐性的，靠人记。

## 实现计划

### A. Done 项归档（C4）

1. 新建 `docs/todolist/done/`，把全部 `Status: Done` 的 todo 移入（保留文件名与内容，
   归档不删史；git mv 保历史）。
2. **先确认 todolist Primitive 的目录语义**：`internal/todolist` 只读
   `docs/todolist/*.md` 单层（不递归）——移入 done/ 后自动退出队列视图，
   这正是想要的效果。**不改 Primitive**；若其实现会递归则加单层限定（先查后改）。
3. `docs/todolist/README.md` 说明归档约定：翻 Done → 下一批入仓时顺手归档。
4. governance layout / docsync 白名单按需登记 `done/` 子目录（先跑门禁看是否报错，
   报错才登记，不预防性加白）。

### B. ps1 退役窗口显性化（C3）

1. `doctor pwsh` 的既有退役计数（0022 刀5）扩展：每个 thin-shell / deprecated 脚本
   输出 `retirementTrigger` 字段（如 "next release after vX" / "after plan-mode GA"）。
   数据源：脚本头部注释的约定标记（`# RETIRE-AFTER: <condition>`），
   缺标记的列为 `unspecified` —— **只报数不设门禁**（ps1 面冻结，减少是自然过程）。
2. 给现有已知退役候选（plan-mode 两个薄壳等）补 `# RETIRE-AFTER:` 标记。

## 明确不做

- 不删除任何 Done 文件内容（归档≠删除）。
- 不改 todolist Primitive 的解析逻辑（除非它意外递归）。
- 不为 ps1 退役设强制门禁、不重写任何 ps1 为 Go。

## 自测

```powershell
bin\aicoding.exe todolist --json          # total 只剩活跃项（预期 ≤3）
ls docs/todolist/done/*.md | Measure-Object  # 归档数 = 移出的 Done 数
git log --follow docs/todolist/done/0003-loop-engineering-kit.md | head -3  # git mv 历史保留
bin\aicoding.exe doctor pwsh --json       # 含 retirementTrigger；unspecified 计数
bin\aicoding.exe governance layout --json ; bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Smoke --json
```

通过判据：队列视图只剩活跃项；归档文件 git 历史可追溯；
doctor pwsh 每脚本有退役条件或 unspecified 标注；全部门禁绿。
