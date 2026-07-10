# 测试报告与 JSON Schema 说明

官方入口：

```powershell
bin\aicoding.exe test full --json
bin\aicoding.exe test release --json
bin\aicoding.exe test latest
```

CLI 输出仍使用 `report.Result` 外壳；`data` 字段使用统一报告结构：

```json
{
  "status": "PASS",
  "summary": {},
  "findings": [],
  "command": "test full",
  "profile": "full",
  "duration_ms": 63550,
  "logs": [
    { "label": "report", "path": "test-results/aicoding-global-test-*/report.md" },
    { "label": "summary", "path": "test-results/aicoding-global-test-*/summary.json" },
    { "label": "results", "path": "test-results/aicoding-global-test-*/results.json" }
  ]
}
```

## 1. `summary.json`

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

## 2. `results.json`

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

## 3. `report.md`

报告按照功能域输出：

1. 总览。
2. 失败项。
3. 告警项。
4. 各功能域结果表。
5. 耗时排名。
6. 用户复查建议。
