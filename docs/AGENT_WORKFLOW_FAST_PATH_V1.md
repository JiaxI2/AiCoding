# AiCoding Fast Path V1 Agent Workflow

## 1. 接手任务时

先确认任务是否属于 V1 范围：

```text
属于：hook、governance、kit Smoke、doctor perf、安装脚本、测试脚本、文档约束
不属于：repo index、MCP、worktree、多 agent、完整 Python repair、Full/Release 重构
```

不属于 V1 的需求，先记录为 V2+，不要混入当前改动。

## 2. 修改前检查

```powershell
git status --short
bin\aicoding.exe doctor perf --json
```

如果二进制不存在：

```powershell
go build -o bin\aicoding.exe ./cmd/aicoding
```

## 3. 修改流程

```text
1. 修改 Go CLI 或脚本
2. 运行 gofmt
3. 运行 go test ./...
4. 构建 bin/aicoding.exe
5. 运行 Smoke 命令
6. 更新 docs 或 AGENT 提示词
7. 确认不影响旧 PowerShell Full/Release
```

## 4. 最小完成证明

```powershell
go test ./...
go build -o bin\aicoding.exe ./cmd/aicoding
bin\aicoding.exe version
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe governance lint --json
bin\aicoding.exe doctor perf --json
```

## 5. 失败处理

如果 Fast Path 失败但旧路径正常：

```text
- 不要删除旧路径
- 优先修复 Go CLI
- 临时让 hook fallback 到 PowerShell
- 在 CHANGELOG 记录已知限制
```
