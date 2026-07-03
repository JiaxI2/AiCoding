# AiCoding Fast Path V1 Agent Prompt

你正在维护 `AiCoding` 仓库的 Fast Path V1。你的目标是优化本地高频路径，而不是重构整个项目。

## 任务目标

优先保持以下命令可用：

```powershell
bin\aicoding.exe hook pre-commit
bin\aicoding.exe hook commit-msg --file .git/COMMIT_EDITMSG
bin\aicoding.exe governance lint --json
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe doctor perf --json
```

## 设计边界

必须遵守：

```text
1. Go 只做 Fast Path，不替代 Full/Release。
2. PowerShell 旧脚本必须保留 fallback。
3. Python kit 不迁移到 Go。
4. 不引入 repo-index、MCP、worktree、多 Agent 控制台。
5. 不引入第三方 Go 依赖，除非用户明确批准。
6. 不改业务 Kit 的真实 install/verify/test 行为。
```

## 修改优先级

按照此顺序处理问题：

```text
1. 修复 aicoding.exe 编译/测试失败
2. 修复 hook pre-commit / commit-msg 兼容性
3. 修复 Smoke manifest/path 检查
4. 修复 PowerShell wrapper/install/test 脚本
5. 更新文档和 Agent 约束
```

## 验证命令

每次修改后至少运行：

```powershell
go test ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin\aicoding.exe version
bin\aicoding.exe kit list --json
bin\aicoding.exe kit verify --all --profile Smoke --json
bin\aicoding.exe doctor perf --json
```

如果在 Linux/macOS：

```bash
go test ./...
go build -o bin/aicoding ./cmd/aicoding
./bin/aicoding version
./bin/aicoding kit verify --all --profile Smoke --json
```

## 安全边界

不得自动执行：

```text
flash / erase / reset / halt / run / write memory / write register / J-Link / DSS intrusive operation
```

不得把 Full/Release 放进 pre-commit hook。

## 输出要求

完成修改时必须说明：

```text
- 修改了哪些文件
- Fast Path 是否通过 go test
- Smoke verify 是否通过
- 旧 PowerShell fallback 是否保留
- 是否需要用户手工安装 Go/Git/PowerShell
```
