# Latest Validation

记录最近一次官方测试 profile 的摘要。原始 `test-results/` 日志目录不提交到仓库。

## Commands

```powershell
go test ./...
go test -race ./...
go vet ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin/aicoding.exe test full --json
bin/aicoding.exe test release --json
bin/aicoding.exe test latest --json
```

## Summary

| Profile | Conclusion | Total | PASS | FAIL | WARN | SKIP | Duration ms | Started | Ended |
|---|---|---:|---:|---:|---:|---:|---:|---|---|
| full | PASS | 48 | 45 | 0 | 0 | 3 | 88855 | 2026-07-09T21:03:37+08:00 | 2026-07-09T21:05:06+08:00 |
| release | PASS | 48 | 48 | 0 | 0 | 0 | 156183 | 2026-07-09T21:05:16+08:00 | 2026-07-09T21:07:53+08:00 |

## Notes

- `test full` and `test release` both completed through the in-repository Go tester.
- `test latest` resolved the latest report to the release profile summary.
- Large per-case stdout/stderr logs are generated under `test-results/` and intentionally ignored by Git.
