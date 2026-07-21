# CLI、验证与测试报告 Schema

官方入口：

```powershell
bin\aicoding.exe doctor --all --json
bin\aicoding.exe verify --profile Smoke|Full|Release --json
bin\aicoding.exe change verify [--staged|--since REV] --json
bin\aicoding.exe test --profile Smoke|Full|Release --json
bin\aicoding.exe test latest
bin\aicoding.exe validation status --json
bin\aicoding.exe validation check --profile Smoke|Full|Release --target HEAD|INDEX [--bind-alias] --json
bin\aicoding.exe validation explain --profile Smoke|Full|Release --target HEAD|INDEX --json
bin\aicoding.exe validation list|clean [--profile Smoke|Full|Release] --json
```

机器可读契约是 [`config/schemas/cli-report.schema.json`](../../../config/schemas/cli-report.schema.json)。
所有 CLI JSON 输出使用兼容的 `report.Result` 外壳：

```json
{
  "schemaVersion": 1,
  "command": "verify --profile Smoke",
  "ok": true,
  "category": "none",
  "retryable": false,
  "message": "AiCoding product verification",
  "inputDigest": "sha256:<normalized-input>",
  "planDigest": "sha256:<execution-plan>",
  "data": {},
  "warnings": [],
  "errors": [],
  "elapsedMs": 635
}
```

`inputDigest` 与 `planDigest` 是可选的加法字段：catalog/list/lifecycle 等有稳定事实输入或
计划意图的命令提供它们；普通诊断不伪造摘要。Lifecycle 的 `data.catalogDigest` 标识静态
adapter catalog，`data.adapters[*].inputDigest` 标识各领域输入。Digest 用于完整性与追踪，
不替代来源信任、授权和运行结果。

`category` 与 `retryable` 是每个外层 Result 的必需字段；成功固定为 `none/false`。失败时
`category` 只能取 `usage`、`validation`、`transient`、`toolchain`、`evidence-missing`、
`conflict`、`internal`，其中只有 `transient` 与 `conflict` 可重试；失败结果同时给出非空
`nextAction`。非法或互相矛盾的组合统一 fail-closed 为 `internal/false`。消费者只需读取
`{ok,category,retryable,nextAction}`，不得解析 `errors[]` 自然语言推进工作流。

兼容字段 `errorKind` 只使用：

- `usage`：参数、flag 或命令使用错误，退出码 `2`；
- `execution`：文件、进程或运行时执行失败，退出码 `1`；
- `validation`：命令成功执行但验证结论失败，退出码 `1`。

当请求 `--json` 时，stdout 只包含一个 JSON 文档，诊断不写入 stderr。`data` 对正式
doctor、verify、test 及已迁移的结构化领域命令使用统一 `StandardReport`。doctor/verify
默认设置 180 秒总超时，可用 `--timeout-sec` 调整，避免外部诊断进程无限等待：

```json
{
  "schemaVersion": 1,
  "status": "PASS",
  "summary": {},
  "findings": [],
  "command": "test --profile Full",
  "profile": "Full",
  "duration_ms": 63550,
  "logs": [
    { "label": "report", "path": "test-results/aicoding-global-test-*/report.md" },
    { "label": "summary", "path": "test-results/aicoding-global-test-*/summary.json" },
    { "label": "results", "path": "test-results/aicoding-global-test-*/results.json" }
  ]
}
```

`status` 只使用 `PASS`、`PASS_WITH_WARNINGS`、`FAIL`。doctor/verify 的 `details`
是共享 check 列表，每项包含稳定的 `id`、`category`、`ok`、`status`、
`duration_ms`、warnings、errors 和领域详情；`test` 的 details 仍是唯一 test engine
生成的 summary/results。test engine 在执行和写报告前校验 Registry 的 `title`：文本必须是
有效 UTF-8 且不得包含 Unicode 替换字符 `U+FFFD`，避免不可读标题进入任何 JSON 消费者。

`change verify` 的 `data` 固定公开 `mode/paths/matches/chosenProfile/target/executionMode`、
`executedCases/reusedCases/receipt/steps`；内部步骤名称为 `changes.detect`、`impact.select`、
`receipt.check`、`test.run`。Receipt 精确命中不运行 test engine，返回
`executionMode=receipt-hit` 与 `executedCases=0`；未命中则内嵌既有 `testReport`，不定义第二份
结果 schema。

## 1. 测试 `summary.json`

```json
{
  "repo": "F:\\Study\\AI\\AiCoding",
  "profile": "full",
  "started_at": "2026-07-09T17:00:00-07:00",
  "ended_at": "2026-07-09T17:05:00-07:00",
  "duration_ms": 300000,
  "total": 42,
  "pass": 38,
  "fail": 1,
  "warn": 2,
  "skip": 1,
  "conclusion": "FAIL",
  "slowest_cases": [
    { "id": "FRESH-001", "duration_ms": 101860 }
  ],
  "cache_hit_ratio": 0,
  "receipt_invalid_reason": "VALIDATION_RECEIPT_MISS: no reusable Receipt exists"
}
```

`slowest_cases` 是本次非 SKIP 用例按 `duration_ms` 降序（同耗时按 ID）排列的 Top 5。
`cache_hit_ratio` 按当前 profile 选中的 TestCase 计权：分母是选中用例数，分子是由整树或节点
Receipt 提供状态的用例数。新鲜执行、`--reuse off`、`--force` 和 `--verify-reuse` 为 `0`；整树
命中为 `1`；节点部分命中为 `reused cases / selected cases` 的小数。
`receipt_invalid_reason` 只在请求复用但整树或节点 Receipt miss/invalid 时出现。

## 2. 测试 `results.json`

```json
{
	"executionMode": "executed",
	"receiptID": "sha256:<receipt>",
	"validationIdentity": "sha256:<content-and-semantics>",
	"resultsDigest": "sha256:<profile-selected-id-statuses>",
	"subjectTreeOID": "<git-tree-oid>",
	"subjectMode": "head",
	"reusable": true,
	"reusableReason": "",
	"validationCode": "VALIDATION_RECEIPT_HIT",
	"checkDurationMs": 0,
  "summary": {},
  "results": [
    {
      "id": "C99-001",
      "category": "C99_SKILL",
      "title": "C99 skill status",
      "status": "PASS",
      "severity": "REQUIRED",
      "duration_ms": 123,
      "queue_ms": 0,
      "setup_ms": 1,
      "execute_ms": 118,
      "persist_ms": 4,
      "exit_code": 0,
      "timed_out": false,
      "json_valid": true,
      "command": "bin/aicoding.exe skill c99-standard-c status --json",
      "stdout_file": "logs/C99-001.stdout.txt",
      "stderr_file": "logs/C99-001.stderr.txt",
      "reason": "command passed"
    }
  ]
}
```

实际执行的用例包含 `queue_ms`、`setup_ms`、`execute_ms`、`persist_ms`，四段之和等于
`duration_ms`；未被 profile 选中的旧式/SKIP 结果省略这四个加法字段。当前串行调度下
`queue_ms` 通常为 `0`，保留该字段是为了让后续调度改造有可比基线。显式 `fresh-clone` 与
Release 的 FRESH-001 共用既有 `FreshCloneReport` 数据形状；`data.steps[*].elapsed_ms` 始终存在
（包括小于 1 ms 时的 `0`）。`sourceMode=cloned` 只表示公共命令执行真实 clone/submodule/
overlay/build/profile verify；`sourceMode=materialized` 表示 FRESH-001 从 Git 对象本地物化、
build 并执行 `release verify`，步骤中不得出现 `git.clone`。FRESH-001 的该 JSON 保存在其
`stdout_file`。

materialized 报告额外内嵌并在运行期间写出源码树外的 `source-manifest.json`：

```json
{
  "sourceMode": "materialized",
  "sourceTreeOID": "<validation-subject-tree>",
  "sourceManifest": {
    "schemaVersion": 1,
    "sourceMode": "materialized",
    "sourceIdentity": "sha256:<superproject-tree-and-recursive-gitlinks>",
    "superprojectTreeOID": "<tree>",
    "submodules": [
      {"path": "CodingKit/agents/skills", "commitOID": "<commit>", "treeOID": "<tree>"}
    ],
    "fileCount": 123
  }
}
```

`sourceIdentity` 只由 superproject Tree 以及排序后的递归 submodule path/commit/tree 组成；
不含临时路径、时间或工作区文件。manifest 位于源码根之外，因此物化源码文件集严格等于这些
Git 对象的 blob 集；成功后临时目录按现有 ledger 释放，但内嵌报告继续保留身份与计数证据。

`executionMode` 只使用 `executed` 或 `reused`：只要任一选中用例真实执行就是 `executed`；整树
命中或全部选中用例均由节点 Receipt 提供时才是 `reused`。复用不引入新的测试结论，
`conclusion` 仍使用既有 `PASS`、`PASS_WITH_WARNINGS`、`FAIL`。`receiptID` 只指向聚合后的完整
PASS 整树 Receipt；节点 Receipt 不通过报告、alias 或 push gate 暴露。`subjectMode` 使用
`head`、`index`、`dirty`；`dirty` 永远不能生成 Receipt。
`validationCode` 是可选的稳定机器码，用于表达命中、主体不可复用、执行期内容漂移、存储错误
或复用审计不一致；调用者不能靠解析 `reusableReason` 文本做分支。

节点命中的 Result 保持 Registry 顺序和原 `id/category/title/severity/command/profile`，状态来自
完整性校验后的节点私有报告，`reason` 固定为 `reused-from-node:<node>`；其
`duration_ms/exit_code` 为 `0`，不伪造 queue/setup/execute/persist 或 stdout/stderr/meta 路径。
部分命中与真实执行结果合并后，仍由同一 summary、`resultsDigest` 和整树 Receipt 聚合。

`resultsDigest` 对当前 profile 选中的 TestCase `(id,status)` 排序后生成，不包含耗时、日志路径或
其他 profile 的未选用例。Receipt schema v2 固定保存该摘要；复用读取会从留存报告重新计算，
`--verify-reuse` 则把新鲜执行摘要与 Receipt 比对，因此即使整体仍归一为 `PASS`，单用例
`PASS` 变为 `WARN` 也会以 `VALIDATION_REUSE_AUDIT_MISMATCH` fail-closed。

可复用的充要条件按 Severity 判定：没有 `FAIL`，所有 profile 内 `REQUIRED` 用例均为
`PASS`，不存在未声明 optional-path 的 profile 内 `SKIP`，且 start/end validation identity
一致。`WARN` 默认不阻断 Receipt；`--strict` 导致的失败仍会阻断。Receipt 的 scope 明确声明
ignored files 不在证明范围，因此它证明 Git 追踪内容，不证明本机 ignored local state。

## 3. `validation explain` 数据

```json
{
  "decision": "miss",
  "checkCode": "VALIDATION_RECEIPT_MISS",
  "referenceIdentity": "sha256:<latest-same-profile-receipt>",
  "referenceSelection": "latest same-profile Receipt by receipt-file mtime; diagnostic only",
  "changed": [
    { "field": "subjectTreeOID", "old": "<old-tree>", "new": "<new-tree>" }
  ],
  "unchanged": [
    "repositoryID",
    "profile",
    "validationPlanDigest",
    "engineSemanticDigest",
    "configDigest",
    "toolchainDigest",
    "optionsDigest"
  ]
}
```

对比字段顺序固定；派生字段 `identity` 不重复列入 `changed`。explain 只在精确 miss 后按
Receipt 文件 mtime 选择最新同 profile 参考并执行完整性校验，不改变 `validation check` 的
O(1) 精确读取路径，也不写 Receipt 或 alias。

## 4. 测试 `report.md`

报告按照功能域输出：

1. 总览。
2. 失败项。
3. 告警项。
4. 各功能域结果表。
5. 耗时排名。
6. 用户复查建议。
