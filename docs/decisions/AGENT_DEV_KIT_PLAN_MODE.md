# AiCoding Agent Dev Kit Plan Mode

## 语言要求 / Language Requirement

本工作流默认中文优先。向用户展示的执行计划、权限请求摘要、命令目的说明、验证结果、风险提示和 rollback 说明，都必须使用中文。英文术语可以保留，但应写成中文 + 英文括号，例如：计划模式（Plan Mode）、规格驱动开发（Spec-Driven Development / SDD）、注册表（registry）、总 hook bridge、子模块（hook module）。

请求执行命令时，不要生成英文摘要；应写成“读取 Plan Mode registry，用于验证前检查。”或“运行 Plan Mode 验证脚本，确认规格、计划、任务和决策记录完整。”。

## Purpose

This document upgrades AiCoding Agent Dev Kit from "clarify and implement" into a plan-first workflow.

The goal is not to copy Spec Kit or Superpower internals. The goal is to align AiCoding with the stable ideas behind them:

- make the plan explicit before code changes;
- keep user intent, architecture choice, tasks, and validation traceable;
- force user selection when architecture is ambiguous;
- keep the Agent in a visible mode: Clarify, Specify, Plan, Tasks, Implement, Verify, or Handoff.

## Plan Mode state machine

```text
Clarify -> Specify -> Plan -> User Decision -> Tasks -> Implement -> Verify -> Handoff
```

## Mode definitions

| Mode | Agent may do | Agent must not do |
|---|---|---|
| Clarify | ask questions, inspect repo read-only, create `docs/spec/<id>/OPTIONS.md` | edit implementation files |
| Specify | write user intent, acceptance criteria, constraints | choose architecture silently |
| Plan | write `docs/spec/<id>/PLAN.md`, risks, validation, rollback | implement code |
| User Decision | show 2-5 options and wait | continue implementation |
| Tasks | write `docs/spec/<id>/TASKS.md` and update traceability in `PLAN.md` | skip test/verify tasks |
| Implement | change files according to selected plan | change plan silently |
| Verify | run Smoke/golden/schema/lint checks | hide failures |
| Handoff | summarize evidence and rollback path | claim unverified success |

## Required artifacts

For fuzzy architecture:

```text
docs/spec/<id>/PLAN.md        # status: needs-decision
docs/spec/<id>/OPTIONS.md
```

For selected architecture:

```text
docs/spec/<id>/DECISION.md
.aicoding/memory/DECISIONS.md
```

For implementation:

```text
docs/spec/<id>/PLAN.md
docs/spec/<id>/TASKS.md
```

## Fuzzy architecture rule

If the requirement has more than one plausible architecture, integration pattern, data model, safety strategy, or lifecycle boundary, the Agent must stop and ask for user selection.

The Agent must show 2-5 options with:

- option name;
- when it fits;
- implementation impact;
- verification impact;
- rollback impact;
- risk;
- recommendation.

No implementation may continue while the selected `docs/spec/<id>/PLAN.md` has
`status: needs-decision`.

## Codex Plan Mode adaptation

Codex Plan Mode should be used as a behavioral mode even when the local Codex UI does not expose a dedicated Plan Mode switch.

In AiCoding, Plan Mode means:

1. read context and current registries;
2. map the request to capabilities;
3. write or update spec artifacts;
4. detect uncertainty;
5. ask the user to choose if needed;
6. only then implement.

## Enforcement

This workflow is enforced by:

```text
tools/specialty/verify-agent-dev-kit-plan-mode.ps1
tools/specialty/hooks/aef/plan-mode-gate.ps1
tools/specialty/hooks/aef/spec-artifact-gate.ps1
config/agent-dev-kit-plan-mode.registry.json
config/agent-hook-modules.registry.json
```
