# TODO 0003: Loop Engineering Kit 落地（阶段 0–3）

Status: Done
Verify: go test ./internal/loopkit/... ./internal/cli/... 且 bin/aicoding.exe kit verify --all --profile Lifecycle --json 通过

> 架构权威：[LOOP_ENGINEERING_ARCHITECTURE.md](../../architecture/LOOP_ENGINEERING_ARCHITECTURE.md)（已随本项提交，Status: Proposed）。
> 定位一句话：**有界迭代工作的裁决者（adjudicator），不是执行器**。唯一新增 Primitive 是
> 转移决策函数 `Decide`；observe/verify 复用 validationevidence，act 永远归 Agent。

## 背景

来源骨架在 `F:\Study\AI\aicoding-loop-engineering-kit`（21 文件/80KB）。评审结论：定位正确，
但 5 处硬阻塞 + 2 处架构缺陷必须在落地时修正，不能照抄。

## 实现计划

### 阶段 0：导入与格式化（提交 1 `chore(loopkit)`）

1. 从来源包复制，**排除**：`CodingKit/agents/skills/**`（只读子模块，提交不进去）、
   `go.mod.fragment.txt`、`FILE_MANIFEST.txt`、`README.md`（打包元数据）。
   `CodingKit/tools/loop-engineering-kit/templates/*` 可以正常提交。
2. 立即 `gofmt -w internal/loopkit`（来源包 7 个 Go 文件全部空格缩进，全不过 gofmt）。
3. `go build ./... && go test ./internal/loopkit/...` 绿后才继续。

### 阶段 1：按裁决者定位重切契约（提交 2 `refactor(loopkit)`）

1. **删除 `internal/loopkit/evidence` 包** → 新增 `internal/loopkit/gateref`：

   ```go
   type GateRef struct {
       Profile            string `json:"profile"`
       ValidationIdentity string `json:"validationIdentity"`
       ReceiptID          string `json:"receiptID"`
   }
   ```

   理由：全仓只允许一个验证证据权威（validationevidence）。gateref 只持字符串，
   **不 import validationevidence**，由 CLI 层组装校验。
2. **重切 controlmode**：`{turn,goal,time,proactive}` 四选一是错的建模（turn 与 proactive
   并非互斥）。改为三正交轴：`Trigger ∈ {explicit, scheduled, agent-proposed}`、
   `Stop`（规则集合）、`Authority`（writeScope + requiredGates + checkpoints）。
3. **收敛全部 `map[string]interface{}`** 为具名字段 + 枚举（阻断 Workflow DSL 化）。
4. **Decide 纯函数签名**（四参数全注入，函数内零 IO）：

   ```go
   func Decide(spec Spec, history []Attempt, gates []GateStatus, now time.Time) (Decision, error)
   ```

5. **五个具名终止态**：`stop-satisfied` / `stop-budget` / `stop-stalled` /
   `stop-violation` / `checkpoint`。
6. **Stop 规则集合**（全部求值，先命中先停）：`maxAttempts`、`maxElapsedSeconds`、
   `maxTotalTokens`、`stallThreshold`、`contextPressureThreshold`（默认 80）、`requiredGates`。
7. **失速检测**：连续 `stallThreshold` 次尝试 `subjectTreeOID` 相同 ⇒ `stop-stalled`
   （复用内容身份，零新增 Git 调用）。
8. **Attempt 内嵌 `tokenusage.Usage`**（`internal/report/tokenusage` 现有结构，
   不新定义 token 类型）+ `subjectTreeOID` + `gateRefs` + `startedAt/endedAt`。
9. `ContextUsedPercent > contextPressureThreshold` ⇒ `checkpoint`（reason `context pressure`）。
10. 同步 `config/schemas/loop-*.schema.json` 与 `testdata/loopkit/examples/*`。

### 阶段 2：治理登记，Kit 不启用（提交 3 `feat(loopkit)`）

1. **从零重写 `config/kits/loop-engineering-kit.json`**（来源包那份 9 字段非法、3 必填缺失）：
   `schemaVersion: 2`、必填 `[id,name,version,kind,mode,commands]`、`additionalProperties:false`。
   第一期**不声明 skills**（SKILL.md 在只读子模块）。
2. `config/kit-registry.json` 增 `{"id":"loop-engineering-kit","enabled":false,"order":90,
   "manifest":"config/kits/loop-engineering-kit.json"}`（schema 要求四字段齐全）。
3. `config/dependency-governance.json` `goPackageBoundaries`：给 gitx/runner/report/registry
   的 forbiddenImports 各加 `internal/loopkit`。
4. **ADR 落为 `docs/decisions/0008-loop-engineering-kit.md`**（编号命名），含
   `PrimitiveReview: required` 与精确锚点 `## §12 Checklist 自评`。ADR 必答：
   与 lifecycle 的边界（目标可声明用 lifecycle，只能验证才用 loop）、与 todolist 的边界、
   权限是检测式非预防式、为何永不实现 `work run`、为何不定义第二 Receipt。
5. 架构文档 `LOOP_ENGINEERING_ARCHITECTURE.md` Status 由 Proposed 转 Accepted，
   接入 `docs/README.md` 索引。

### 阶段 3：四条命令（提交 4 `feat(cli)`）

```text
aicoding work validate --file <spec.json> --json                     只读
aicoding work next     --file <spec.json> --json                     只读（核心）
aicoding work status   --file <spec.json> --json                     只读
aicoding work record   --file <spec.json> --attempt <a.json> --json  仅追加 attempts.jsonl
```

- `next` 的门禁判定调用 validationevidence 的 check 语义，不自建。
- 状态落 `.aicoding/state/work/<id>/`（state.json + attempts.jsonl，追加不可变）。
- **必须同步** `CommandWork` + 四条 HelpForm + `docs/COMMANDS.md`（缺 HelpForm 则
  `newCommandCatalog` 启动即 panic）。
- **永不实现** `work run` / `work prepare` / `work step`。

## 自测（可信任方式，逐条贴输出）

```powershell
gofmt -l internal/loopkit                                    # 必须为空
go build ./... ; go test ./internal/loopkit/... ./internal/cli/... ; go vet ./...
bin\aicoding.exe governance dependencies --json              # 反向依赖禁令生效
bin\aicoding.exe governance layout --json
bin\aicoding.exe docsync all --json
bin\aicoding.exe kit verify --all --profile Lifecycle --json # manifest schema 真正被校验的路径
bin\aicoding.exe kit list --json                             # 新 Kit 出现且 enabled:false
bin\aicoding.exe work validate --file testdata/loopkit/examples/project-development.work.json --json
bin\aicoding.exe work next --file <...> --json               # 连续两次，决策字节一致
bin\aicoding.exe test --profile Full --json                  # 含 ADR-001 §12 门禁
git status --porcelain                                       # 干净
```

通过判据：
1. `kit verify --profile Lifecycle` 绿（注意：`--profile Smoke` 不跑结构门禁，会假绿）。
2. 全仓 `grep -rn "type Receipt" internal/` 只有 validationevidence 一处。
3. `internal/loopkit` 无残留 `map[string]interface{}`。
4. `work next` 确定性：同 spec + 同历史 + 同证据连续两次，剔除 elapsed 后字节一致。
5. `git status` 中 `CodingKit/agents/skills` 无变更（未越界改只读子模块）。
6. `test --profile Full` 全绿。
