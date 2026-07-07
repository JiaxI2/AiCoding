# AGENTS Fast Path V1 Overlay (Historical)

本文件是 AiCoding Fast Path V1 的仓库级 Agent 约束补充，保留用于理解历史约束和回滚背景。

当前默认路径是 Fast Path V2；新的 Fast Path 收敛工作以 V2 文档、Go CLI 和 Taskfile 入口为准。

## 历史范围

Fast Path V1 当时只优化：

```text
hook pre-commit
hook commit-msg
governance lint
kit Smoke verify
doctor perf
```

不得把当前任务扩大为 repo-index、MCP、worktree、多 Agent 控制台或完整平台重构。

## 默认命令

优先使用：

```powershell
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe doctor perf --json
```

如果二进制不存在：

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/install-fast-path-v1.ps1
```

## 修改原则

```text
1. Go CLI 是 hot path。
2. PowerShell 是兼容和完整路径。
3. Python 是 AI/repair/test 领域路径。
4. Markdown/JSON 是 agent 可读配置和规则。
5. 不在 hook 中运行 Full/Release。
6. 不执行硬件侵入式操作。
```

## 完成证明

任何 Fast Path V1 修改完成时，都要给出：

```text
- go test ./... 结果
- go build 结果
- kit Smoke verify 结果
- doctor perf 结果
```
