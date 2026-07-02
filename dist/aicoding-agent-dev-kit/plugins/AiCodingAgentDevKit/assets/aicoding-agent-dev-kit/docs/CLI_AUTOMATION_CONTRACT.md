# CLI Automation Contract

本 Kit 的效率目标是：Agent 只做判断、设计、审查；机械性检查交给 CLI 和脚本。

## CLI 命令分层

```text
install     安装和分发 Kit 资产
status      返回安装状态
index       缓存仓库文件索引
changed     返回变更文件列表
brief       输出短上下文简报
context     构建最小上下文包
token-audit 估算上下文成本并提示大文件
compact     压缩 progress/session summary
shard       根据 IMPLEMENTATION_PLAN 生成任务切片
gate        调用质量门禁
uninstall   安全卸载或完整卸载
```

## PowerShell 对应脚本

```text
scripts/agent-fast-start.ps1
scripts/cache-file-index.ps1
scripts/list-changed-files.ps1
scripts/build-agent-context-pack.ps1
scripts/token-audit.ps1
scripts/update-session-summary.ps1
scripts/compact-agent-memory.ps1
scripts/plan-task-shards.ps1
scripts/invoke-fast-agent-loop.ps1
```

## Hook 与 CI 模式区别

```text
pre-commit:
  只检查 staged/diff 相关内容，追求快。
  默认不做全仓库深扫。

ci:
  可做完整检查，但仍复用缓存和 diff。
  失败时输出最小错误报告，不输出大段文件内容。

release:
  执行完整 traceability、desensitization、spec、TDD、docs 检查。
```
