# Todolist（待实现工作清单）

本目录记录**已规划、尚未实现**的工作项。每个 `.md` 是一项待办，先把完整实现计划丢进来
（`Status: Planned`），后续实现完成、其 `Verify` 命令绿灯后改为 `Status: Done`。

由 `todolist` Primitive（`internal/todolist`）读取并汇报：

```bash
bin/aicoding.exe todolist --json
```

它只读取本目录（不扫描仓库），输出每项的 `title`/`status`/`verify` 与汇总计数。

## 文件格式（约定优于配置）

文件名：`NNNN-slug.md`（四位序号 + 短横线短名）。头部前几行：

```markdown
# TODO NNNN: 标题

Status: Planned          # Planned | In-Progress | Done
Verify: <绿灯命令>        # 能证明"已完成"的可执行命令（如 go test ./... -run TestX）

## 背景 / 实现计划 / 完成定义（绿灯）
...
```

`README.md` 不计入待办。`Status` 支持别名：`wip`→In-Progress，`green|complete`→Done。

## 生命周期

`Planned` →（开始）`In-Progress` →（`Verify` 命令绿灯）`Done`。
"绿灯"由该项声明的 `Verify` 命令证明，而不是口头声明——与仓库其它门禁一致。
