# C99 Standard C Skill

C/H 风格能力属于 `c99-standard-c` skill，不作为独立 formatting kit 暴露。C UserStyle Kit
1.2.0 是该 skill 使用的首方外部工具资产，注册为 `c-userstyle-kit`，但不新增用户命令命名空间。

## 配置边界

- Skill 配置：`config/skills/c99-standard-c/skill.json`
- 机械格式：`config/skills/c99-standard-c/style/clang-format.yaml`
- 注释模板：`config/skills/c99-standard-c/templates/comment-templates.json`
- 嵌入式 C 规则：`config/skills/c99-standard-c/rules/embedded-c-rules.md`
- Go 实现：`internal/cstyle`
- C Kit 根目录：`CodingKit/tools/c-userstyle-kit`
- C Kit 配置：`CodingKit/tools/c-userstyle-kit/examples/c-kit.json`
- 用户 snippets：`CodingKit/tools/c-userstyle-kit/examples/c-snippets.json`
- 快速验证目标：`CodingKit/tools/c-userstyle-kit/examples/verify-target.json`
- Kit manifest：`config/kits/c-userstyle-kit.json`

根目录 `.clang-format` 只是工具投影；source of truth 是 skill 配置。`.vscode` 不入仓。

机械格式和原有模板入口继续使用仓库级 skill 配置；黄金 Demo、139 条规则目录、第 8 章注释方法、
高级规则覆盖样例、snippets、lint、主机编译和行为测试由 C UserStyle Kit 提供。高级样例对最终用户可见，
便于先检查代码与注释表达，再调整 JSON 配置。

华为 C 语言编程规范 DKBA 2826-2011.5 的 PDF 与完整 Markdown 参考副本位于 C Kit 的
`references/`，经用户明确授权可随 AiCoding 的 kit/component 发布包分发。

## Taskfile 入口

```bash
task style:c:status
task style:c:templates
task style:c:verify
task fmt:c
task fmt-check:c
task fmt-check-staged:c
```

## Go CLI 入口

```bash
bin/aicoding.exe skill c99-standard-c status --json
bin/aicoding.exe skill c99-standard-c templates --json
bin/aicoding.exe skill c99-standard-c verify --json
bin/aicoding.exe skill c99-standard-c verify --profile full --timings --json
bin/aicoding.exe skill c99-standard-c fmt --scope changed --json
bin/aicoding.exe skill c99-standard-c check --scope changed --json
bin/aicoding.exe skill c99-standard-c check --scope staged --json
bin/aicoding.exe skill c99-standard-c check --scope paths --path testdata/style-samples/foc_sample.c --json
```

当前用户入口只保留 `skill c99-standard-c` 和 Taskfile 短路由。

## C Kit 验证

`verify` 通过 `skill.json` 的 Kit 元数据定位 C Kit。默认 `fast` profile 适合编辑循环，执行配置、
lint、GCC C99、头文件探针和主机行为测试；`full` 在收口时增加 Clang、C++17 头文件探针及
baseline/candidate 行为等价。registry manifest 登记同一资产及其维护命令，两条路径共享相同配置
和快速目标。入口返回统一 JSON envelope，不调用 TI/CCS 或其他固件工具链，也不会把候选文件
加入固件构建。

```bash
# 默认黄金目标的秒级反馈
bin/aicoding.exe skill c99-standard-c verify --profile fast --timings --json

# 发布前完整主机门禁
bin/aicoding.exe skill c99-standard-c verify --profile full --timings --json

# 验证项目自有 target；overlay 可重复指定
bin/aicoding.exe skill c99-standard-c verify --profile full \
  --target path/to/verify-target.json \
  --overlay path/to/project-overlay.json --timings --json
```

`--target` 和 `--overlay` 的相对路径均以 AiCoding 仓库根解析；不存在的文件、未知参数、错误
schema 或子进程返回的 profile 不匹配都会使命令非零退出。

底层等价命令由 manifest 固定为：

```bash
go -C CodingKit/tools/c-userstyle-kit run ./cmd/cstylekit verify \
  --config ./examples/c-kit.json \
  --target ./examples/verify-target.json \
  --profile fast --timings --json
```

底层命令用于 registry/维护验证；日常用户仍应使用 `skill c99-standard-c verify` 路由。

## 范围

- `changed`: modified 和 untracked C/H 文件。
- `staged`: staged C/H 文件。
- `paths`: `--path` 显式路径。
- `all`: 全仓 C/H 文件，但不作为默认 Taskfile 路由。

默认排除项来自 `skill.json`：单个目录名（例如 `vendor`、`generated`）在任意层级匹配；含 `/` 的条目按仓库相对路径前缀匹配。除常规生成目录外，平台格式检查排除自带独立规则与门禁的 `CodingKit/tools/c-userstyle-kit`，但不排除其他 `CodingKit` 或 `tools` 内容。

## 验证

```bash
go test ./internal/cstyle
go run ./cmd/aicoding skill c99-standard-c status --json
go run ./cmd/aicoding skill c99-standard-c templates --json
go run ./cmd/aicoding skill c99-standard-c verify --json
go run ./cmd/aicoding skill c99-standard-c verify --profile full --timings --json
```
