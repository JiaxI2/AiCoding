# Worktree Parallelism

Use worktrees when multiple phases can be developed independently.

## Rules

- One worktree = one phase = one Agent session.
- Avoid parallel work if phases modify the same files.
- Every worktree gets its own memory note under `.agent-memory/worktrees/`.
- Merge only after quality gate passes in the worktree.

## Commands

```powershell
pwsh -File scripts/worktree/new-agent-worktree.ps1 -Name phase-backend-api -Base main
pwsh -File scripts/worktree/list-agent-worktrees.ps1
pwsh -File scripts/worktree/merge-agent-worktree.ps1 -Name phase-backend-api
pwsh -File scripts/worktree/remove-agent-worktree.ps1 -Name phase-backend-api
```
