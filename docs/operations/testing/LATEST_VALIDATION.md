# Latest Validation

记录最近一次官方测试 profile 的摘要。原始 `test-results/` 日志目录不提交到仓库。

## Commands

```powershell
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/aicoding bootstrap --json
bin/aicoding.exe doctor --all --json
bin/aicoding.exe verify --profile Release --json
bin/aicoding.exe test --profile Smoke --json
bin/aicoding.exe test --profile Full --json
bin/aicoding.exe test --profile Release --json
bin/aicoding.exe test latest --json
bin/aicoding.exe kit verify --all --profile Lifecycle --json
bin/aicoding.exe skill verify --all --profile Release --json
bin/aicoding.exe mcp verify --all --profile Release --configured --repo-root F:\Study\AI\AiCoding --json
bin/aicoding.exe docsync release --json
bin/aicoding.exe governance dependencies --json
git hook run pre-commit
```

## Summary

| Profile | Conclusion | Total | PASS | FAIL | WARN | SKIP | Duration ms | Started | Ended |
|---|---|---:|---:|---:|---:|---:|---:|---|---|
| smoke | PASS | 54 | 38 | 0 | 0 | 16 | 9784 | 2026-07-17T00:35:49+08:00 | 2026-07-17T00:35:59+08:00 |
| full | PASS | 54 | 52 | 0 | 0 | 2 | 126297 | 2026-07-17T00:41:07+08:00 | 2026-07-17T00:43:13+08:00 |
| release | PASS | 54 | 53 | 0 | 0 | 1 | 65071 | 2026-07-17T00:43:33+08:00 | 2026-07-17T00:44:38+08:00 |

## Notes

- Smoke、Full、Release 均通过唯一 `internal/testengine` Registry；跳过项只属于未选择的更高或相邻 profile。
- Full/Release 的 rollback 用例只验证 `lifecycle rollback --help`，没有读取或应用本地 rollback snapshot。
- `test latest` 指向本次 release profile；原始日志保存在忽略的 `test-results/`。
- Doctor 为 `PASS_WITH_WARNINGS`，仅提示当前 worktree 未安装本地 Visio MCP venv；Release Verify 为 `PASS_WITH_WARNINGS`，仅提示 Git 不跟踪空目录 `CodingKit/examples` 与 `CodingKit/platforms`。
- 正式安装在 main 工作区的 MCP Release Verify 通过，耗时 114949 ms，并实际执行 visible Visio COM smoke。
- Markdown link validator 验证 136 个文件通过；Release 模板占位符、1 个 GBK 文档和 2 个 PDF 转换参考文档按其格式边界排除。
- Large per-case stdout/stderr logs are generated under `test-results/` and intentionally ignored by Git.
