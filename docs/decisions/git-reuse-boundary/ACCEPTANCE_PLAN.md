# Acceptance Plan: Git Reuse Boundary

验收人：独立于实现者的审查会话。所有阶段全部通过才算验收完成；任一 FAIL 打回并
附具体证据（文件:行、命令输出）。

## Phase 0：diff 半径

```powershell
git -C <worktree> diff --stat main...HEAD
```

- [ ] 改动文件全部落在 IMPLEMENTATION_PLAN 列出的范围内；
- [ ] 无新增 CLI 命令（`internal/cli/catalog.go` 的 CommandID 集合不变）；
- [ ] `internal/registry`、`internal/runner`、`internal/report` 公开契约零改动；
- [ ] 已冻结架构文档只允许新增交叉引用行，无契约内容改动。

## Phase 1：静态审查

- [ ] `docs/architecture/GIT_REUSE_BOUNDARY.md` 存在，§9 三条门禁与实际实现一一对应；
- [ ] gitx 仍是薄封装：无语义函数（只有进程启动 + 输出解析），import 列表只有标准库；
- [ ] 生产代码中 gitx 之外零处直接启动 git 进程：

```powershell
# 期望：只命中 internal/gitx 与 *_test.go 与 CodingKit/tools/**
grep -rn "exec.Command" --include="*.go" internal cmd | grep '"git"'
```

- [ ] `config/dependency-governance.json` 含 gitx boundary entry 与 `gitProcessBoundary`
  节，allowedImporters 与实际 importer 集合精确一致（不多不少——用 grep import 核对）；
- [ ] schema 覆盖新节且 `additionalProperties` 约束风格与现有一致；
- [ ] catalog 禁用测试的动词集合与 GIT_REUSE_BOUNDARY.md §9 清单逐词一致，
  含 alias 检查，含 status/tag/fresh-clone 豁免注释。

## Phase 2：门禁执行（全部退出码 0 且 ok=true）

```powershell
go build ./...
go test ./internal/governance/... ./internal/cli/... ./internal/gitx/... ./internal/kit/... ./internal/lifecycle/... ./internal/cstyle/... ./internal/platform/...
bin\aicoding.exe governance dependencies --json
bin\aicoding.exe docsync --json
bin\aicoding.exe test --profile Smoke --json
bin\aicoding.exe test --profile Full --json
```

- [ ] `governance dependencies --json` 的 checks 列表包含
  `git process ownership` 与 `gitx importer allowlist` 两项且 OK；
- [ ] JSON 按 `schemaVersion`/`ok`/`errorKind` 判读，不以人类文本判断成功。

说明：本次跑 Full 是因为改动触及跨模块公开契约（governance JSON 报告、catalog
契约、四个包的调用点），符合验证半径规则。冻结后的 gitx 内部修改按
GIT_REUSE_BOUNDARY.md §9.1 的最小半径执行，不默认全量。

## Phase 3：突变验证（门禁真的会咬人）

在临时分支上逐项注入违规 → 断言门禁失败 → 还原。四个突变：

| # | 注入 | 期望失败点 |
|---|---|---|
| M1 | 在 `internal/report` 任一生产文件加 `exec.Command("git", "version")` | `governance dependencies` 报 git process ownership error，指明文件 |
| M2 | 在 `internal/runner` 任一生产文件 import gitx | `governance dependencies` 报 importer allowlist error |
| M3 | 在 gitx 中 import `internal/platform` | `governance dependencies` 报 goPackageBoundaries error |
| M4 | 在 catalog 中注册名为 `commit`（或 alias `push`）的命令 | `TestCommandCatalogRejectsGitPorcelainVerbs` 失败 |

- [ ] M1–M4 全部按期望失败，失败信息可定位到文件；
- [ ] 还原后门禁恢复全绿（防止突变残留）。

## Phase 4：行为等价抽查（迁移未破坏原功能）

- [ ] pre-commit 链路：构造一个 staged docs 改动，`.githooks/pre-commit` 行为与
  迁移前一致（docsync staged 检查仍触发）；
- [ ] `bin\aicoding.exe doctor --all --json` ok=true（覆盖 repohealth/platform 的
  rev-parse 路径）；
- [ ] kit 结构检查（structure.go 的 `status --short` 路径）由 Smoke/Full 覆盖，
  确认对应测试确实执行而非被跳过；
- [ ] lifecycle runtime-skill 只读路径：
  `bin\aicoding.exe lifecycle status --scope runtime-skill --json` ok=true
  （覆盖 rev-parse HEAD / git-common-dir 迁移）。

## Phase 5：文档与治理收尾

- [ ] CHANGELOG 条目存在；任何 identity（目录、包名、check 名、schema ID）不含版本号；
- [ ] docsync 门禁通过即视为文档登记完成（不做额外人工登记）；
- [ ] GIT_REUSE_BOUNDARY.md 中链接全部可解析（相对路径正确）。

## 签收

以上全部通过后：

1. 将 `GIT_REUSE_BOUNDARY.md` Status 改为 `Accepted and Frozen`；
2. 将 IMPLEMENTATION_PLAN.md Plan Status 改为 `Approved`（若用户尚未改）并勾满 TASKS.md；
3. 在本文件末尾追加签收记录：验收日期、验收会话、Phase 2 各命令的 ok/elapsed 摘要。

签收记录：

- （待填）
