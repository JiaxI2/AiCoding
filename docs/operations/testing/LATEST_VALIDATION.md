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
bin/aicoding.exe test --profile Release --json
bin/aicoding.exe test latest --json
bin/aicoding.exe kit verify --all --level lifecycle --json
bin/aicoding.exe skill verify --all --profile Release --json
bin/aicoding.exe mcp verify --all --profile Release --configured --repo-root F:\Study\AI\AiCoding --json
bin/aicoding.exe docsync release --json
bin/aicoding.exe governance dependencies --json
git hook run pre-commit
```

## Summary

| Profile | Conclusion | Total | PASS | FAIL | WARN | SKIP | Duration ms | Started | Ended |
|---|---|---:|---:|---:|---:|---:|---:|---|---|
| smoke | PASS | 54 | 38 | 0 | 0 | 16 | 11627 | 2026-07-18T14:30:54+08:00 | 2026-07-18T14:31:05+08:00 |
| full | PASS | 54 | 52 | 0 | 0 | 2 | 156497 | 2026-07-18T14:32:03+08:00 | 2026-07-18T14:34:40+08:00 |
| release | PASS | 54 | 53 | 0 | 0 | 1 | 65071 | 2026-07-17T00:43:33+08:00 | 2026-07-17T00:44:38+08:00 |

## Notes

- 当前发布流程只执行 Release；Release 对 Full 的 73-leaf 直接超集证据见
  [TODO 0041](../../todolist/done/0041-release-only-publication.md)。下表的 Full 行保留
  2026-07-18 历史快照，不是现行发布步骤。
- Smoke、Full、Release 均通过唯一 `internal/testengine` Registry；跳过项只属于未选择的更高或相邻 profile。
- Full/Release 的 rollback 用例只验证 `lifecycle rollback --help`，没有读取或应用本地 rollback snapshot。
- `test latest` 指向本次 release profile；原始日志保存在忽略的 `test-results/`。
- 当前 worktree 已通过 lifecycle 安装 Visio/PPT MCP，并跟踪 `CodingKit/examples` 与 `CodingKit/platforms` 稳定根；`doctor --all` 为 5/5 PASS、`verify --profile Smoke` 为 12/12 PASS，均为 0 warning。
- 正式安装在 main 工作区的 MCP Release Verify 通过，耗时 114949 ms，并实际执行 visible Visio COM smoke。
- PowerPoint MCP Release 通过 585 个 mock 回归并执行 visible COM smoke，生成 1 页 `ppt-mcp-smoke.pptx`；结束后没有残留 `POWERPNT.EXE`。
- 全仓 Markdown link audit 检查 1368 个链接，0 error、0 stderr；根 `lychee.toml` 仅按所有权排除两个只读 Git submodule，并按编码边界精确排除 1 个 GBK README。
- Large per-case stdout/stderr logs are generated under `test-results/` and intentionally ignored by Git.
