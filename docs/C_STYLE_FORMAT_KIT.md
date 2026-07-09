# C Style Format Kit

This kit makes C/H formatting deterministic, fast, and agent-friendly.

## Design

Natural-language instructions are not precise enough to enforce code style. The kit boundary is:

- `.clang-format` defines mechanical C/H formatting.
- `internal/cstyle` applies formatting, checks formatting, and validates template configuration through Go.
- `config/cstyle/comment-templates.json` is the repository-native source for C comment templates.
- `config/skills/c99-standard-c-overlay.md` aligns C skill behavior with this repository policy.
- `aicoding cstyle` exposes the stable CLI.
- Taskfile routes short commands only.

`.vscode` is not tracked by this repository. Editor snippets or IDE adapters are local generated artifacts, or future optional adapters, not a core kit capability.

Formatter reference version: `clang-format` 17.0.2, LLVM release URL: <https://github.com/llvm/llvm-project/releases/tag/llvmorg-17.0.2>.

## Default workflow

```bash
task fmt:c
task fmt-check:c
```

The default scope is `changed`, so only modified and untracked `.c/.h` files are processed.

For staged-only pre-commit checks, prefer:

```bash
task fmt-check-staged:c
```

## CLI commands

```bash
bin/aicoding.exe cstyle status --json
bin/aicoding.exe cstyle templates --json
bin/aicoding.exe cstyle fmt --scope changed --json
bin/aicoding.exe cstyle check --scope changed --json
bin/aicoding.exe cstyle fmt --scope staged --json
bin/aicoding.exe cstyle check --scope staged --json
bin/aicoding.exe cstyle fmt --scope all --json
bin/aicoding.exe cstyle fmt --scope paths --path src/foc.c --json
bin/aicoding.exe cstyle fmt --scope changed --preview --json
```

For source execution:

```bash
go run ./cmd/aicoding cstyle check --scope changed --json
go run ./cmd/aicoding cstyle templates --json
```

## Scopes

- `changed`: modified + untracked C/H files.
- `staged`: staged C/H files.
- `all`: all repository C/H files except excluded directories.
- `paths`: explicit file list, useful for tests and targeted edits.

## Default exclusions

```text
vendor/
third_party/
generated/
Drivers/
device/
build/
out/
dist/
.git/
```

Do not auto-format TI/vendor/generated files unless intentionally doing a one-time style migration.

## Repository-Native Comment Templates

C comment templates are stored in:

```text
config/cstyle/comment-templates.json
```

The file provides Doxygen-style templates for:

- C file headers
- function headers
- section dividers
- struct definitions
- enum definitions
- common embedded includes

The JSON file is the authoritative template source. `aicoding cstyle templates --json` validates template IDs, language, kind, and non-empty bodies. The command does not generate code.

## C Skill Overlay

C skill behavior for this repository is documented in:

```text
config/skills/c99-standard-c-overlay.md
```

The overlay keeps the C skill aligned with `.clang-format`, repository-native comment templates, staged format checks, and embedded real-time constraints.

## Tests

```bash
go test ./internal/cstyle
```

The formatter-dependent test is skipped when `clang-format` is not available on `PATH`.

## Agent Rule Snippet

```md
When modifying `.c` or `.h` files, never rely on manual style edits. Run:

1. `task fmt:c`
2. `task fmt-check:c`

Before commit, prefer `task fmt-check-staged:c` for staged-only checks.

Do not format `vendor/`, `third_party/`, `generated/`, `Drivers/`, or `device/` unless the task explicitly requests a full style migration.
```
