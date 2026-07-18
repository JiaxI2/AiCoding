# CLI、验证与测试报告 Schema

官方入口：

```powershell
bin\aicoding.exe doctor --all --json
bin\aicoding.exe verify --profile Smoke|Full|Release --json
bin\aicoding.exe test --profile Smoke|Full|Release --json
bin\aicoding.exe test latest
```

机器可读契约是 [`config/schemas/cli-report.schema.json`](../../../config/schemas/cli-report.schema.json)。
所有 CLI JSON 输出使用兼容的 `report.Result` 外壳：

```json
{
  "schemaVersion": 1,
  "command": "verify --profile Smoke",
  "ok": true,
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

失败时可选 `errorKind` 只使用：

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
生成的 summary/results。

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
  "conclusion": "FAIL"
}
```

## 2. 测试 `results.json`

```json
{
  "summary": {},
  "results": [
    {
      "id": "C99-001",
      "category": "C99_SKILL",
      "title": "C99 skill status",
      "status": "PASS",
      "severity": "REQUIRED",
      "duration_ms": 123,
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

## 3. 测试 `report.md`

报告按照功能域输出：

1. 总览。
2. 失败项。
3. 告警项。
4. 各功能域结果表。
5. 耗时排名。
6. 用户复查建议。
