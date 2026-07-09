# C99 Standard C Skill

C/H 风格能力属于 `c99-standard-c` skill，不作为独立 formatting kit 暴露。

## 配置边界

- Skill 配置：`config/skills/c99-standard-c/skill.json`
- 机械格式：`config/skills/c99-standard-c/style/clang-format.yaml`
- 注释模板：`config/skills/c99-standard-c/templates/comment-templates.json`
- 嵌入式 C 规则：`config/skills/c99-standard-c/rules/embedded-c-rules.md`
- Go 实现：`internal/cstyle`

根目录 `.clang-format` 只是工具投影；source of truth 是 skill 配置。`.vscode` 不入仓。

## Taskfile 入口

```bash
task style:c:status
task style:c:templates
task fmt:c
task fmt-check:c
task fmt-check-staged:c
```

## Go CLI 入口

```bash
bin/aicoding.exe skill c99-standard-c status --json
bin/aicoding.exe skill c99-standard-c templates --json
bin/aicoding.exe skill c99-standard-c fmt --scope changed --json
bin/aicoding.exe skill c99-standard-c check --scope changed --json
bin/aicoding.exe skill c99-standard-c check --scope staged --json
bin/aicoding.exe skill c99-standard-c check --scope paths --path tests/style-samples/foc_sample.c --json
```

当前用户入口只保留 `skill c99-standard-c` 和 Taskfile 短路由。

## 范围

- `changed`: modified 和 untracked C/H 文件。
- `staged`: staged C/H 文件。
- `paths`: `--path` 显式路径。
- `all`: 全仓 C/H 文件，但不作为默认 Taskfile 路由。

默认排除目录来自 `skill.json`：`vendor`, `third_party`, `generated`, `Drivers`, `device`, `build`, `out`, `dist`。

## 验证

```bash
go test ./internal/cstyle
go run ./cmd/aicoding skill c99-standard-c status --json
go run ./cmd/aicoding skill c99-standard-c templates --json
```