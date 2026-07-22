# AiCoding DocSync Plus Specification

Status: Accepted and Frozen

## Role

DocSync Plus is a repository-maintenance kit for AiCoding. It upgrades documentation synchronization from a path-only gate to a Git-diff-driven semantic drift gate.

It is not a Codex Skill, not a plugin, and not a replacement for `bin/aicoding.exe docsync`.

## Architecture

```text
git diff changed files
  -> path gate
  -> semantic classifier
  -> Go CLI / test profile / Taskfile / CI command-surface check
  -> PowerShell specialty parameter/ValidateSet check
  -> JSON policy rule check
  -> Markdown command index check
  -> DOCSYNC-NO-DOC-CHANGE quality check
  -> doc drift score
  -> text/json/markdown report
```

## Compatibility

The following existing calls must remain valid:

```powershell
bin/aicoding.exe docsync staged --json
bin/aicoding.exe docsync all --json
```

DocSync Plus adds:

```powershell
bin/aicoding.exe docsync ci --json
bin/aicoding.exe docsync release --json
```

## Policy schema closure

`internal/docsync/policy_schema.go` 是 checked-in 配置/schema 闭合的唯一 binding authority。
当前 35/35 个非 schema JSON 配置逐项执行 schema 校验；29/29 个 schema 必须由配置 binding
或 standalone 工件登记反向引用。schema 与配置均缺一不可，未知字段 fail-closed。

`governance dependencies` 把既有单次 repository inventory 交给同一权威，阻断未登记配置、
幽灵 schema、幽灵排除与模糊通配。`config/schema-closure-exclusions.json` 只允许精确文件或
目录后缀 `/**`；当前 `config/schemas/**` 的排除仅表示它不是配置实例，schema 本身仍受反向
引用检查。因此依赖门禁与文档同步门禁不会形成两套 schema 解释或第二次仓库扫描。

## Scoring

Default score weights are stored in `config/docs-sync.semantic.json`:

```text
apiDrift      35
behaviorDrift 25
policyDrift   20
commandDrift  10
linkDrift     10
```

Modes:

| Mode | Behavior |
|---|---|
| `pre-commit` | Fast local gate. Warning above `preCommitWarn`; fail above `preCommitBlock`. |
| `all` | Full local gate. Fail above `allBlock`. |
| `ci` | Strict CI gate. Fail above `ciBlock`. |
| `release` | Release gate. Fail above `releaseBlock` unless a valid review note exists. |

## No-doc marker policy

`DOCSYNC-NO-DOC-CHANGE` is supported, but it must include a meaningful reason:

```text
DOCSYNC-NO-DOC-CHANGE: only renamed internal fixture directory; no user-facing command, policy, hook, or doc behavior changed.
```

Invalid examples:

```text
DOCSYNC-NO-DOC-CHANGE
DOCSYNC-NO-DOC-CHANGE: skip
DOCSYNC-NO-DOC-CHANGE: no
```

## MVP boundaries

Implemented first:

- Go CLI, test engine, Taskfile and CI command-surface bindings
- shared CLI report schema bindings
- PowerShell script surface checks
- JSON policy checks
- Markdown command index checks
- no-doc marker quality checks
- doc drift score
- staged/all/ci/release Go checks

Deferred:

- full C/C++ AST
- full Python CLI AST
- LLM-generated patches
- automatic PR creation
- release automation
```
