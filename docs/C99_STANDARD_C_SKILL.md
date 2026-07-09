# C99 Standard C Skill

This repository exposes C/H style as part of the `c99-standard-c` skill, not as a standalone formatting kit.

## Boundary

- Skill configuration: `config/skills/c99-standard-c/skill.json`
- Style backend config: `config/skills/c99-standard-c/style/clang-format.yaml`
- Comment templates: `config/skills/c99-standard-c/templates/comment-templates.json`
- Embedded C rules: `config/skills/c99-standard-c/rules/embedded-c-rules.md`
- Go implementation: `internal/cstyle`

The formatter backend is an implementation detail. The source of truth is the C skill configuration. The root `.clang-format` file is kept only as a compatibility projection for existing tools.

`.vscode` is not tracked by this repository. Editor adapters are local generated artifacts or future optional adapters, not core skill capability.

## Taskfile Entrypoints

```bash
task style:c:status
task style:c:templates
task fmt:c
task fmt-check:c
task fmt-check-staged:c
```

Taskfile remains routing only. The logic lives in Go under `internal/cstyle`.

## Skill CLI

```bash
bin/aicoding.exe skill c99-standard-c status --json
bin/aicoding.exe skill c99-standard-c templates --json
bin/aicoding.exe skill c99-standard-c fmt --scope changed --json
bin/aicoding.exe skill c99-standard-c check --scope changed --json
bin/aicoding.exe skill c99-standard-c check --scope staged --json
bin/aicoding.exe skill c99-standard-c check --scope paths --path tests/style-samples/foc_sample.c --json
```

`aicoding cstyle status|templates|fmt|check` remains as a compatibility alias, but the preferred user-facing entry is the skill command.

## Scopes

- `changed`: modified and untracked C/H files.
- `staged`: staged C/H files.
- `paths`: explicit paths supplied by `--path`.
- `all`: all repository C/H files except excluded directories.

Default excluded directories are defined in `skill.json`: `vendor`, `third_party`, `generated`, `Drivers`, `device`, `build`, `out`, and `dist`.

## Templates And Rules

`templates/comment-templates.json` validates Doxygen-style file headers, function headers, section dividers, struct comments, enum comments, and common C includes. The default author remains `HU JIAXUAN`.

`rules/embedded-c-rules.md` keeps the C skill aligned with C99 embedded rules, staged checks, and ISR/current-loop constraints.

## Validation

```bash
go test ./internal/cstyle
go run ./cmd/aicoding skill c99-standard-c status --json
go run ./cmd/aicoding skill c99-standard-c templates --json
```
