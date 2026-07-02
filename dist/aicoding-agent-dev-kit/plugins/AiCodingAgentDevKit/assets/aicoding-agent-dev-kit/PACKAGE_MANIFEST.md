# Package Manifest

## Version

v0.11.1.1-generic-clarify-plan-progress

## Main Additions

1. Spec Pack：
   - `spec/PRD.md`
   - `spec/APP_FLOW.md`
   - `spec/TECH_STACK.md`
   - `spec/CODING_GUIDELINES.md`
   - `spec/PROJECT_STRUCTURE.md`
   - `spec/IMPLEMENTATION_PLAN.md`
   - `spec/TEST_STRATEGY.md`

2. 三道审查门：
   - Requirement Review
   - Design Review
   - Task/TDD Plan Review

3. TDD 强制进入 Implementation Plan：
   - 写失败测试
   - 运行测试确认失败
   - 最小实现
   - 运行测试确认通过
   - 重构
   - 再次运行测试
   - 更新 progress / lessons / evidence
   - commit

4. 跨会话记忆：
   - `.agent-memory/CURRENT.md`
   - `.agent-memory/DECISIONS.md`
   - `.agent-memory/decisions.md`
   - `.agent-memory/session-handoff.md`

5. Worktree 并行开发：
   - `new-agent-worktree.ps1`
   - `list-agent-worktrees.ps1`
   - `merge-agent-worktree.ps1`
   - `remove-agent-worktree.ps1`

6. Subagent 模板：
   - `spec-reviewer.md`
   - `implementation-planner.md`
   - `tdd-enforcer.md`
   - `worktree-coordinator.md`
   - `systematic-debugger.md`

7. 两张图：
   - `diagrams/kit-execution-flow.svg`
   - `diagrams/hook-ci-trigger-sequence.svg`

## Safety Contract

- 默认 uninstall 不删除用户维护的 spec、memory、ADR、BDD、TDD 文档。
- `--purge --force` 才删除完整 Kit-owned 骨架。
- CI 不依赖 Superpowers。
- Thin Skill 不保存完整规则，只做 Agent 路由入口。
- 包内容已脱敏，不包含未开源项目名称或私有业务名称。

## v0.11.1 Token / Speed Additions

- `docs/TOKEN_SPEED_OPTIMIZATION.md`
- `docs/CLI_AUTOMATION_CONTRACT.md`
- `docs/FAST_WORKFLOW.md`
- `config/context-budget.json`
- `.agent-memory/CURRENT.md`
- `.agent-dev-kit/context/`
- `.agent-dev-kit/cache/`
- `scripts/cache-file-index.ps1`
- `scripts/list-changed-files.ps1`
- `scripts/token-audit.ps1`
- `scripts/build-agent-context-pack.ps1`
- `scripts/agent-fast-start.ps1`
- `scripts/update-session-summary.ps1`
- `scripts/compact-agent-memory.ps1`
- `scripts/plan-task-shards.ps1`
- `scripts/invoke-fast-agent-loop.ps1`
- Python CLI commands: `fast-start`, `context`, `changed`, `token-audit`, `compact`, `shard`, `index`

## v0.11.1 Lightweight Decision Memory Changes

Removed heavy memory model:

- `progress.txt`
- `lessons.md`
- `decisions.md`
- `session-handoff.md`
- `session-summary.md`
- `archive/`
- `journal/`

Added lightweight decision memory:

- `.agent-memory/README.md`
- `.agent-memory/CURRENT.md`
- `.agent-memory/DECISIONS.md`
- `docs/DECISION_MEMORY_POLICY.md`
- `docs/MEMORY_VS_DOCSYNC.md`

Added commands/scripts:

- `current show`
- `current set`
- `decision add`
- `decision list`
- `decision promote-adr`

## v0.11.1 Sequential Loader Additions

- `config/loading-policy.json`
- `config/doc-sync-bridge.json`
- `docs/SEQUENTIAL_LOADING_MODEL.md`
- `docs/DOCSYNC_BRIDGE.md`
- `docs/V0_8_QUICKSTART.md`
- `diagrams/sequential-loading-flow.svg`
- `scripts/load-agent-context.ps1`
- `scripts/analyze-change-scope.ps1`
- `scripts/show-context-manifest.ps1`
- Python CLI commands:
  - `load`
  - `manifest`
  - `scope`

v0.11.1 also updates `invoke-agent-quality-gate.ps1` so it can optionally orchestrate existing DocSync and Git governance scripts when they exist in the target repository.

## v0.11.1 Codex Native Adapter Additions

Codex-native adapter files:

```text
plugins/aicoding-agent-dev-kit/.codex-plugin/plugin.json
plugins/aicoding-agent-dev-kit/skills/aicoding-agent-dev-kit/SKILL.md
plugins/aicoding-agent-dev-kit/hooks/hooks.json
plugins/aicoding-agent-dev-kit/hooks/*.ps1
.agents/plugins/marketplace.json
.codex/config.toml
.codex/hooks.json
.codex/agents/*.toml
```

Scripts:

```text
scripts/install-codex-native-adapter.ps1
scripts/verify-codex-native-adapter.ps1
scripts/uninstall-codex-native-adapter.ps1
```

CLI:

```text
aicoding-agent-kit codex-native status --repo .
aicoding-agent-kit codex-native verify --repo .
```

## v0.11.1 Hook Bridge Additions

New policy:

```text
One repository = one Git Hook entrypoint.
Agent Dev Kit is a bridge module, not the hook owner.
```

Added:

```text
config/hook-bridge.json
docs/HOOK_BRIDGE_POLICY.md
docs/AICODING_EXISTING_HOOK_INTEGRATION.md
docs/CODEX_HOOK_OPT_IN.md
assets/hook-bridge.pre-commit.ps1.snippet
assets/hook-bridge.pre-commit.sh.snippet
scripts/detect-existing-hooks.ps1
scripts/install-hook-bridge.ps1
scripts/verify-hook-bridge.ps1
scripts/uninstall-hook-bridge.ps1
```

Updated:

```text
scripts/install-agent-dev-kit.ps1
scripts/install-codex-native-adapter.ps1
scripts/verify-codex-native-adapter.ps1
src/aicoding_agent_kit/cli.py
```

CLI:

```text
aicoding-agent-kit hook detect --repo .
aicoding-agent-kit hook install-bridge --repo . --merge-existing-hook
aicoding-agent-kit hook uninstall-bridge --repo .
```

## v0.11.1 Requirement Clarification Additions

Added PRD option matrix, selected solution document, forced solution-doc sync validator, MVP progress monitor, progress CLI, and Codex requirement clarifier agent.

## v0.11.1 Generic Clarification Patch

This patch removes application-specific examples from the reusable Kit.

Policy:

```text
The Kit provides generic planning scaffolding only.
Concrete product/domain examples are generated in the target repository after reading that repo's context.
```

Updated files:

```text
README.md
config/clarification-policy.json
spec/PRD_OPTIONS.md
spec/SELECTED_SOLUTION.md
docs/REQUIREMENT_CLARIFICATION_MODE.md
docs/TECHNICAL_OPTION_MATRIX_GUIDE.md
scripts/start-requirement-clarification.ps1
scripts/select-solution-option.ps1
```
