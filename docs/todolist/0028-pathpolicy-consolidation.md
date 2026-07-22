# TODO 0028: pathpolicy 解析收敛（三个同构 policy 复用一个 Primitive + 补齐缺失 schema）

Status: Planned
Verify: internal/pathpolicy 单测全绿；plan/impact/validation 三个 policy 的 pattern 解析走同一实现；6 个 *policy*.json 全部有 schema 且 docsync 校验覆盖

> 来源：FORWARD_PLAN C1。实测（2026-07-22）：6 个 `*policy*.json` 只有 1 个有 schema；
> `plan-policy` / `impact-policy` / `validation-policy` 三个**结构同构**
> （路径 pattern → 裁决），却各写各的解析。宪法 §1「避免重复实现」的现成违规点。

## 实现计划

1. 抽 `internal/pathpolicy` Primitive（单一职责：pattern 匹配 + 去重排序 + fail-closed
   校验）。公开 API ≤4 个函数：`Compile(patterns) / Match(compiled, path) /
   Validate(patterns)`。**确定性：同输入同输出，pattern 排序稳定。**
2. `internal/plan`、`internal/testengine`（impact/raceScope 判定）、
   `internal/validationevidence`（policy 匹配）改为消费 pathpolicy。
   **只合并解析逻辑，不合并配置文件**（三份 policy 语义不同，保持各自文件）。
3. 注意依赖方向：pathpolicy 是 Primitive 层，只依赖 stdlib；
   `dependency-governance.json` 给 gitx/report/runner/registry 的 forbiddenImports
   加 `internal/pathpolicy` 反向禁令按需评估（Primitive 间互不依赖）。
4. 补 5 个缺失 schema：`impact-policy` / `validation-policy` / `docs-sync.policy` /
   `tagging-policy`（+ 已有 plan-policy 共 6 个全覆盖），docsync/governance 校验接入。
5. glob 语义一致性测试：同一 pattern 在三个消费方的匹配结果必须一致
   （表驱动，含 `internal/cli/x/y.go` 命中 `internal/cli/**` 边界例）。

## 明确不做

- 不合并三个 policy 文件为一个（语义不同：敏感触发 / 影响面 / push 规则）。
- 不改任何 policy 的现有语义与字段（纯解析收敛，行为零变化）。
- 不做正则方言扩展（现有 glob 语义冻结）。

## 自测

```powershell
go test ./internal/pathpolicy/... ./internal/plan/... ./internal/testengine/... ./internal/validationevidence/...
# 行为零变化回归：收敛前后对同一组 staged 路径，plan check / change verify 输出字节一致（剔除 elapsed）
bin\aicoding.exe plan check --staged --json ; bin\aicoding.exe change verify --staged --json
# schema 负例：往 impact-policy.json 塞非法字段 → docsync/governance 必须红 → 撤销
bin\aicoding.exe governance dependencies --json ; bin\aicoding.exe docsync all --json
bin\aicoding.exe test --profile Full --json
```

通过判据：全仓 pattern 解析实现只剩一处（grep 旧解析函数为零残留）；
三消费方行为零变化（前后字节对比）；6/6 schema 齐全且负例被抓；Full 全绿。
