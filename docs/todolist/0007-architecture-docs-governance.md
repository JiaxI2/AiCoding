# TODO 0007: 架构文档治理（阅读路径 + Status 门禁 + MCP 权威收敛）

Status: Planned
Verify: bin/aicoding.exe docsync all --json 通过且含 architecture Status 检查；docs/architecture/README.md 存在

> 对抗性审计结论：22 份/2969 行架构文档中，真正定义契约的只有 4 份共 755 行；
> 26% 语料（编号系列 00–07）自我声明"不定义新契约"但读者无从得知可跳过；
> 5 份文档缺 `Status:` 头（含被 docs/README 列为"稳定入口"的 KIT_LIFECYCLE）；
> 两份 MCP 文档主题重叠且被三份上游同时引用，违反"一主题一权威"。

## 实现计划

1. **新增 `docs/architecture/README.md`**（分层阅读路径，零删改）：

   ```markdown
   ## 必读（改任何代码之前，共约 755 行）
   1. AICODING_CORE_ARCHITECTURE.md      是什么（契约事实）
   2. PRIMITIVE_CONSTITUTION.md          该长什么样（设计法）
   3. FREEZE_AND_ACQUISITION_BOUNDARY.md 哪里不能碰
   4. EXTENSION_ADAPTER_CONTRACT.md      新东西怎么进来
   ## 按需（做特定领域时）
   CLI_MCP_CONTROL_PLANE / KIT_LIFECYCLE / GIT_REUSE_BOUNDARY / POWERSHELL_BOUNDARY /
   DOC_SYNC_PLUS_SPEC / GRAPH_FIRST / LOOP_ENGINEERING / （0006 后：PLAN_MODE）
   ## 派生视图（不定义契约，可不读）
   00–07 编号系列 · ARCHITECTURE_HANDBOOK
   ## Kit 架构文档模板骨架（约定）
   结论 → 系统位置图 → 邻居边界判据 → 数据契约 → 可靠性与安全表 → 演进边界（明确不做）
   ```

2. **五份补 `Status:` 头**（一行改动）：
   - `KIT_LIFECYCLE_ARCHITECTURE.md` → `Status: Accepted and Frozen`（它是稳定入口）
   - `CODEX_KIT_ARCHITECTURE.md` / `DOC_SYNC_PLUS_SPEC.md` / `POWERSHELL_BOUNDARY.md`
     → 按内容判定 Frozen 或 Derived View
   - `MCP_CONTROL_PLANE.md` → 见第 3 条
3. **MCP 两份收敛为一权威**：`MCP_CONTROL_PLANE.md`（130 行，无 Status）降级
   `Status: Derived View`，开篇加一行"权威见 CLI_MCP_CONTROL_PLANE.md，冲突以其为准"；
   引用它的三份上游（CORE/HANDBOOK/GIT_REUSE）不必改链接（降级后语义已明确）。
4. **docsync 加 Status 门禁**（~20 行 Go，`internal/docsync`）：
   `all/ci/release` 模式下，`docs/architecture/*.md`（除 README.md）必须含
   `^Status: ` 行，缺失即 error。**不做** markdown 语义理解/双向同步/自动改文档
   （守住 docsync 的职责边界，防 God Core）。
5. **冻结面可执行断言**（testengine 静态用例，每条一个，Category `DOCS` 或新 `FREEZE`）：
   - `FREEZE-001`：FREEZE_AND_ACQUISITION_BOUNDARY 列出的冻结 schema 文件全部存在。
   - `FREEZE-002`：`grep` 断言 `type Result struct` 在 `internal/report` 唯一
     （§11"不新增第二 report authority"的机器化）。
   - `FREEZE-003`：`type Receipt` 在 `internal/validationevidence` 唯一。

## 明确不做

- 不删任何现有文档、不合并编号系列。
- 不给 docsync 加语义分析。
- 依赖表述四处收敛（DEPENDENCY_DIRECTION_POLICY 唯一权威）列为可选项，
  时间不够就只做本项 1–5。

## 自测（可信任方式）

```powershell
go test ./internal/docsync/... ./internal/testengine/...
# docsync 负例：临时删掉某文档 Status 行 → docsync all 必须 error → 恢复
bin\aicoding.exe docsync all --json          # 绿，且输出含 architecture status 检查项
bin\aicoding.exe docsync staged --json
bin\aicoding.exe test --profile Full --json  # FREEZE-00x 用例出现且绿
bin\aicoding.exe governance layout --json
git status --porcelain
```

通过判据：负例被抓（贴输出）；README 阅读路径中每个链接可点开（lychee 或手查）；
`MCP_CONTROL_PLANE.md` 头两行明确降级；FREEZE-002/003 在故意注入重复类型时变红（本地验证后撤销）。
