# AI Debug Kit 边界卡

Status: Accepted Direction

1. **位置与形态。** `CodingKit/tests/ai-debug-kit/pyproject.toml:1-30` 定义 Python 3.11+、uv/pytest 与 `ai-debug` CLI；`CodingKit/tests/ai-debug-kit/_external/Mklink-AI-Probe` 是独立 gitlink，来源登记在 `.gitmodules:6-9`。pin 策略只跟随上游 stable tag，任何前移都必须先评审目标 tag、commit 与兼容性。

2. **不进 Go 控制面。** 在出现明确的平台消费者并通过 ADR 之前，不把 ai-debug-kit 加入 `config/kit-registry.json`，不新增 AiCoding CLI 命令，也不让 `internal/testengine` 引用它；现有 Kit 注册模式可对照 `config/kit-registry.json:20-40`，这条边界是准入条件而非待补登记。

3. **不承担门禁。** Smoke、Full、Release 以及任何 verify/test profile 都不得依赖 ai-debug-kit 输出；它当前仅是 `CodingKit/tests` 下的独立实验资产，自身验证入口保留为 uv/pytest，见 `CodingKit/tests/ai-debug-kit/README.md:19-43`。

4. **上游同步纪律。** 借鉴项目的变更先在上游或 fork 验证，通过评审后只更新 `_external/Mklink-AI-Probe` 的 gitlink pin；本仓库禁止复制上游实现再本地分叉修改。上游 URL 及 submodule 边界以 `.gitmodules:6-9` 为准。
