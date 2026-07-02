# Token and Speed Optimization

目标：在工作效率最大化的前提下，尽量减少 Agent 消耗的上下文 token，并提升本地执行速度。

## 核心原则

```text
能用 CLI / 脚本判断的，不让 Agent 靠阅读全文判断。
能生成摘要的，不把大文件全文塞进上下文。
能按 diff 定位的，不扫描整个仓库。
能缓存索引的，不重复遍历文件树。
能并行 worktree 的，不在长会话里串行堆上下文。
```

## 分层读取策略

Agent 不应一上来读取整个仓库。按四层读取：

```text
L0: 快速状态
  .agent-dev-kit/cache/repo-index.json
  .agent-memory/CURRENT.md
  .agent-memory/CURRENT.md

L1: 当前任务上下文
  .agent-dev-kit/context/context-pack.md
  git changed files
  linked spec sections

L2: 必要源文件
  changed files only
  touched modules only
  test files only

L3: 深度追溯
  PRD / SDD / ADR / BDD / TDD full docs
  only when validation fails or design changes
```

## 推荐启动命令

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/agent-fast-start.ps1 -RepoRoot . -Json
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/build-agent-context-pack.ps1 -RepoRoot . -Mode changed -MaxChars 12000 -Json
```

## 推荐 Agent 行为

1. 会话开始只读 `agent-fast-start` 输出。
2. 需要更多上下文时读 `context-pack.md`。
3. 只有修改接口、架构、行为时才读完整 spec/ADR/BDD。
4. 每完成一个任务更新 `progress.txt` 和 `session-summary.md`。
5. 每次用户纠正或失败后更新 `lessons.md`。
6. 长任务拆成 worktree，避免单会话上下文爆炸。

## 禁止模式

```text
禁止：把整个 README、全部 docs、全部源码一次性贴入上下文。
禁止：每轮都重新遍历仓库全文。
禁止：没有 diff 依据就读所有文件。
禁止：把二进制、构建产物、日志全文交给 Agent。
禁止：让 Agent 手动判断可由脚本判断的规则。
```

## v0.11.1 Decision Memory Simplification

v0.11.1 removes the heavy memory model.

Use only:

```text
.agent-memory/CURRENT.md
.agent-memory/DECISIONS.md
```

Do not use long session journals or full conversation summaries.
