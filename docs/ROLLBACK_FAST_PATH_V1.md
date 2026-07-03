# AiCoding Fast Path V1 Rollback

Fast Path V1 是增量加速层，不应破坏旧 PowerShell/Python 路径。

## 1. 仅停用 Git hook fast path

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/rollback-fast-path-v1.ps1 -UnsetHooksPath
```

或者：

```powershell
git config --unset core.hooksPath
```

## 2. 删除本地二进制

```powershell
pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/rollback-fast-path-v1.ps1 -RemoveBinary
```

## 3. 回退文件改动

如果是通过 Git 管理：

```powershell
git checkout -- .githooks/pre-commit .githooks/commit-msg .github/workflows/fast-path.yml cmd/aicoding go.mod scripts/aicoding-fast.ps1 scripts/install-fast-path-v1.ps1 scripts/test-fast-path-v1.ps1 scripts/measure-fast-path-v1.ps1 scripts/rollback-fast-path-v1.ps1 docs/AICODING_FAST_PATH_ARCHITECTURE_V1.md docs/KIT_LAYER_CONSTRAINTS_FAST_PATH_V1.md docs/AGENT_PROMPT_FAST_PATH_V1.md docs/AGENT_WORKFLOW_FAST_PATH_V1.md docs/ROLLBACK_FAST_PATH_V1.md AGENTS_FAST_PATH_V1.md .agents/prompts/aicoding-fast-path-v1.md .codex/skills/aicoding-fast-path-v1/SKILL.md
```

## 4. 保留旧路径验证

```powershell
pwsh scripts/aicoding-kit.ps1 test -All -Profile Smoke -Json
pwsh scripts/aicoding-kit.ps1 test -All -Profile Full -Json
```
