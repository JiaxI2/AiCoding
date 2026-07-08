# Branch: feature/go-fast-path-v1

AiCoding Fast Path V1 branch description. This file is the remote-facing branch description and the Draft PR body source.

## Purpose

Land AiCoding Fast Path V1: a Go native hot path for local high-frequency checks, keeping the existing PowerShell/Python kit system untouched.

Core goals: efficient, fast, convenient, maintainable.

## Scope

Included:

- `cmd/aicoding` Go CLI: `hook pre-commit`, `hook commit-msg`, `governance lint`, `kit list/doctor/verify --profile Smoke`, `doctor perf`
- `.githooks/pre-commit`, `.githooks/commit-msg`: fast CLI first, `go run` fallback, PowerShell legacy fallback
- `scripts/install-fast-path-v1.ps1`, `test-fast-path-v1.ps1`, `measure-fast-path-v1.ps1`, `rollback-fast-path-v1.ps1`, `aicoding-fast.ps1`
- `.github/workflows/fast-path.yml` fast smoke CI
- Fast Path docs, agent prompt/workflow docs, `AGENTS_FAST_PATH_V1.md`, `.agents/prompts/aicoding-fast-path-v1.md`, `.codex/skills/aicoding-fast-path-v1/SKILL.md`
- `README.md` / `README_CN.md` / `README_EN.md` environment preview Go row + Fast Path V1 section, `CHANGELOG.md` entry, `.gitignore` `/bin/`

Not included:

- repo-index, tree-sitter, MCP, worktree orchestration, memory.sqlite, VS Code extension, Rust rewrite
- Full/Release rewrite; `bin/aicoding.exe` and legacy verify/test scripts unchanged
- Skill external cache work (`scripts/aicoding-skill.ps1`, `config/skill-sources.json`, `docs/THIRD_PARTY_SKILL_POLICY.md`, `scripts/lib/AiCoding.SkillAudit.psm1`) stays on a separate branch

## Verification

- `go test ./...` PASS
- `go build -o bin/aicoding.exe ./cmd/aicoding` PASS
- `bin/aicoding.exe kit verify --all --profile Smoke --json` PASS (7 kits)
- `bin/aicoding.exe governance lint --json` PASS
- `bin/aicoding.exe doctor perf --json` PASS
- `bin/aicoding.exe docsync -Mode all` PASS
- Fast hook path is roughly 10x faster than the legacy double-pwsh pre-commit path

## Rollback

```powershell
bin/aicoding.exe lifecycle rollback --last --json
```

Hooks fall back to PowerShell automatically when `bin/aicoding.exe` is absent. See `docs/ROLLBACK_FAST_PATH_V1.md`.

## Maintenance Rules

- Maintain Fast Path V1 on this branch; do not continue Fast Path work on `main`.
- Sync with `git fetch origin` + `git rebase origin/main` (ask before merging instead).
- Full/Release keep using `bin/aicoding.exe` and `bin/aicoding.exe fresh-clone`.
