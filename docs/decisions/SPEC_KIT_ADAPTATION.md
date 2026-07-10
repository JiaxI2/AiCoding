# Spec Kit Adaptation for AiCoding

> 语言要求：面向用户的计划、验证、风险、rollback 和 handoff 说明必须中文优先；英文术语保留为括号说明。

## Why adapt instead of copy

GitHub Spec Kit is a full specification-driven development toolkit. AiCoding already has its own kit lifecycle, PowerShell scripts, hook registry, skill routing, and embedded development constraints.

Therefore AiCoding should adapt Spec Kit as a pattern, not vendor it wholesale.

## Mapping

| Spec Kit concept | AiCoding adaptation |
|---|---|
| constitution | `docs/AGENT_ENGINEERING_FOUNDATION.md`, `docs/AGENT_WORKFLOW_STANDARD.md`, `AGENTS.md` |
| specify | `docs/decisions/plan-mode-overlay/PRD_OPTIONS.md`, requirements sections in `docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md` |
| clarify | AiCoding Agent Dev Kit fuzzy requirement gate |
| plan | `docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md` |
| tasks | `docs/decisions/plan-mode-overlay/TASKS.md` |
| analyze | `tools/specialty/verify-agent-dev-kit-plan-mode.ps1` and hook submodules |
| checklist | `docs/decisions/plan-mode-overlay/CHECKLIST.md`, golden tests, Smoke verify |
| implement | only after selected solution and approved plan |
| converge | update `docs/decisions/plan-mode-overlay/TRACEABILITY.md`, changelog, docs, and remaining tasks |

## AiCoding-specific additions

AiCoding adds embedded/agent safety requirements:

- dry-run first for write operations;
- no default flash/reset/halt/run/loadProgram/erase/write-memory;
- Smoke is default;
- Full/Release explicit;
- one hook bridge with module dispatch;
- state and trace output must be JSON-readable.

## Artifact lifecycle

```text
docs/decisions/plan-mode-overlay/PRD_OPTIONS.md          # options if fuzzy
docs/decisions/plan-mode-overlay/NEEDS_USER_DECISION.md  # blocks implementation
docs/decisions/plan-mode-overlay/SELECTED_SOLUTION.md    # user selection
.aicoding/memory/DECISIONS.md     # decision memory
docs/decisions/plan-mode-overlay/IMPLEMENTATION_PLAN.md  # technical plan
docs/decisions/plan-mode-overlay/TASKS.md                # execution tasks
docs/decisions/plan-mode-overlay/TRACEABILITY.md         # requirement-plan-task-verify links
docs/decisions/plan-mode-overlay/CHECKLIST.md            # quality checklist
```
