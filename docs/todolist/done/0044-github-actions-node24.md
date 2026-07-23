# TODO 0044: GitHub Actions Node 24 迁移

Status: Done
Verify: go test ./internal/testengine/... -count=1 && bin/aicoding.exe docsync all --json

## 范围

`.github/workflows/` 的全部 action 使用各官方仓库当前稳定、`runs.using=node24` 的 release，
继续采用 `owner/action@<40位commit SHA> # vN`。不改变触发条件、job 结构、并发、
permissions 或 release-gate seed/audit 命令。

## 官方 tag → commit 核验

验证方法：`releases/latest` 取得非 draft/prerelease tag；`git/ref/tags/<tag>` 解析 ref；
annotated tag 继续用 `git/tags/<sha>` peel 到 commit；同时要求 `vN` major tag 指向同一
commit，并读取精确 tag 的 `action.yml` 验证 `runs.using=node24`。

| Action | 当前稳定 tag | 注释 | 经官方 ref 解析的 commit SHA |
|---|---|---|---|
| `actions/checkout` | `v7.0.1` | `# v7` | `3d3c42e5aac5ba805825da76410c181273ba90b1` |
| `actions/setup-go` | `v7.0.0` | `# v7` | `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` |
| `actions/upload-artifact` | `v7.0.1` | `# v7` | `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` |
| `actions/github-script` | `v9.0.0` | `# v9` | `3a2844b7e9c422d3c10d287c895573f7108da1b3` |
| `go-task/setup-task` | `v2.1.0` | `# v2` | `01a4adf9db2d14c1de7a560f09170b6e0df736aa` |

逐项 ref/peel 原始结果与 release URL 见
[GitHub Actions Node 24 迁移原始证据](../../operations/evidence/github-actions-node24-migration.md)。

## 冻结规格锚点

`internal/testengine/engine_test.go` 的
`TestScheduledCISeedsAndAuditsReleaseBeforeDefaultPromotion` 是 setup-task pin 与版本注释的
唯一规格锚点。本轮只把该确定字符串同步到 v2.1.0 的实际 commit；错误 40 位 SHA 已真实使
测试失败，证明锚点没有删除、放宽或被运行时条件绕过。今后有意改变 setup-task pin 时必须
同步这一锚点。

## 不变量

- `--reuse off` 冷种子行 SHA-256：
  `32a261f8da3dded57d44d53a2a490d5a2398d0124bd5280ccd9eab664ac55f05`。
- `--verify-reuse` 审计行 SHA-256：
  `18b75214cc2f8b16569534110bf0391d961b2bc3dc80c2d84a60163cd786352b`。
- README 三件套、`docs/ARCHITECTURE_OVERVIEW.md` 已审阅：本轮不新增用户命令、稳定工具
  身份或架构边界，无需改动；当前测试文档、COMMANDS 与 CHANGELOG 已同步。

## 完成条件

- 本地锚点与 workflow 结构门禁全绿。
- 包含本变更的 main 远端 workflow dispatch 全部适用 job 成功，Node 20 弃用警告消失。
- AiCoding CI：
  [run 30001965694](https://github.com/JiaxI2/AiCoding/actions/runs/30001965694)，四个 job
  全部成功；完整 `6567` 行日志中 Node 20/deprecation 匹配数为 `0`。
- Issue governance：
  [run 30001969328](https://github.com/JiaxI2/AiCoding/actions/runs/30001969328)，适用的
  `sync-labels` 成功；完整 `78` 行日志中 Node 20/deprecation 匹配数为 `0`。
- 最终 Release summary：`test-results/0046-final-release/summary.json`。
