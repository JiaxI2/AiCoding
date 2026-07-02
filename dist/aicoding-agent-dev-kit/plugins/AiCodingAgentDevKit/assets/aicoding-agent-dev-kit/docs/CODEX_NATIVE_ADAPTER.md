# Codex Native Adapter

v0.11.1 adds a Codex-native adapter layer on top of the platform Kit.

## Why

The platform Kit remains cross-agent and executable by CLI/PowerShell.
The Codex adapter adds Codex discovery and lifecycle surfaces:

```text
plugins/aicoding-agent-dev-kit/.codex-plugin/plugin.json
plugins/aicoding-agent-dev-kit/skills/
plugins/aicoding-agent-dev-kit/hooks/
.agents/plugins/marketplace.json
.codex/hooks.json
.codex/agents/*.toml
```

## Layers

```text
Platform Kit:
  scripts/
  spec/
  docs/
  .agent-memory/
  .agent-dev-kit/
  .githooks/
  .github/workflows/

Codex Native Adapter:
  plugin manifest
  plugin-bundled Skill
  plugin-bundled lifecycle hooks
  repo marketplace
  project-level .codex hooks
  project-scoped custom agents
```

## Use Modes

### Repo marketplace mode

The plugin lives under:

```text
plugins/aicoding-agent-dev-kit/
```

and is exposed through:

```text
.agents/plugins/marketplace.json
```

Then restart Codex and open `/plugins`.

### Project hook mode

The repo also includes:

```text
.codex/hooks.json
```

These project-local hooks run only after the project `.codex/` layer is trusted by the user.

### Custom agents

Project scoped Codex agents live under:

```text
.codex/agents/
```

They are narrow and opinionated:

- `spec_reviewer`
- `implementation_planner`
- `tdd_enforcer`
- `worktree_coordinator`
- `systematic_debugger`

## Trust and Safety

Codex requires review/trust for non-managed command hooks. Do not assume lifecycle hooks will run before the user trusts them.

The Git hook and CI quality gate remain the deterministic enforcement layer.
