# AiCoding 全局测试验收准则

## 1. 总体验收

| 结论 | 判定 |
|---|---|
| PASS | REQUIRED 用例全部 PASS；WARN 用例无阻断性问题 |
| PASS_WITH_WARNINGS | REQUIRED 全部 PASS，但存在 WARN，需要人工判断 |
| FAIL | 任一 REQUIRED 用例 FAIL 或 TIMEOUT |
| INCOMPLETE | 测试环境缺失导致大量 SKIP，无法判断仓库质量 |

## 2. C99 Skill 验收

必须满足：

1. `skill c99-standard-c status --json` 成功。
2. `skill c99-standard-c templates --json` 成功。
3. `skill c99-standard-c verify --json` 成功，并明确执行 C UserStyle Kit `fast` profile。
4. `check --scope paths --path testdata/style-samples/foc_sample.c --json` 成功，或报告明确样例不存在。
5. `config/skills/c99-standard-c/skill.json` 指向 style、templates、rules 和 `c-userstyle-kit` 元数据。
6. `.clang-format` 明确是投影，source-of-truth 为 skill 配置。
7. 排除项包含 vendor/third_party/generated/Drivers/device/build/out/dist，并仅按仓库相对路径排除自包含的 `CodingKit/tools/c-userstyle-kit`。
8. C Kit 1.2.0 的黄金/高级样例、139 条规则目录、snippets、PDF 与 Markdown 参考副本存在。

## 3. Go 并发验收

必须满足：

1. `go test ./...` 成功。
2. CLI 并发只读调用全部返回 0。
3. 无 timeout。
4. `go test -race` 若环境支持，应通过；若环境不支持，降级为 WARN，并保留日志。

## 4. 文档同步验收

必须满足：

1. README 三件套存在。
2. COMMANDS、C99 skill 文档存在。
3. README 中的默认入口与 COMMANDS 中的命令矩阵不冲突。
5. DocSync CI/Release gate 成功。

## 5. Lifecycle / 外部 Skill 验收

必须满足：

1. `config/kit-registry.json` 可解析。
2. registry 中 manifest 均存在。
3. install/update/uninstall plan 均可输出 JSON。
4. export zip 路径成功。
5. 默认测试不污染用户全局 skill 状态。

## 6. Git 治理验收

必须满足：

1. hooks verify 成功。
2. repo-text verify 成功。
3. release-notes verify 成功。
4. governance lint 成功。
5. tag audit 成功。
6. `.gitattributes` 明确 text/binary/EOL 策略。

## 7. 失败处理建议

| 失败域 | 优先修复方向 |
|---|---|
| BOOTSTRAP | 修复 Go build、路径识别、bin 输出 |
| GO | 先修 `go test ./...`，再修 race/并发 |
| C99_SKILL | 先修 skill.json/style/templates/rules，再修 formatter |
| DOCSYNC | 修 README/COMMANDS/C99 文档索引 |
| LIFECYCLE | 修 registry/manifest/schema/version/路径 |
| GIT_GOVERNANCE | 修 hook、gitattributes、tag policy、release notes |
| PWSH_BOUNDARY | 清理默认入口中残余 PowerShell 编排 |
