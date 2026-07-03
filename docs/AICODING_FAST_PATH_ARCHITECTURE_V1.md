# AiCoding Fast Path Architecture V1

> 目标：保留现有 PowerShell/Python Kit 生态，不推翻旧脚本；只把高频、低延迟路径迁移到 Go native CLI，降低 Git hook、Smoke gate、registry/manifest 检查和性能诊断的启动成本。

## 1. 当前问题判断

当前仓库已经有较好的生命周期抽象：`config/kit-registry.json` 统一登记 7 个 kit，`scripts/aicoding-kit.ps1` 作为 registry/manifest adapter 入口，`Smoke / Full / Release` 已经分层。主要性能问题不在业务逻辑，而在热路径仍然通过多次 `sh -> pwsh -> ps1 -> git/python/ps1` 进程链路执行。

最明显的热路径是：

```text
.githooks/pre-commit
  -> scripts/lint-git-governance.ps1
  -> scripts/check-documentation-sync.ps1
```

这意味着一次提交至少两次 PowerShell 冷启动，还可能触发 docsync 模块加载、Git 查询和文件扫描。

## 2. V1 架构原则

### 2.1 保留旧系统

- 不删除 `scripts/aicoding-kit.ps1`。
- 不删除任何 `verify-*.ps1`、`test-*.ps1`、`install-*.ps1`。
- Full / Release 仍然通过 PowerShell/Python/CI 运行。
- 新 Go CLI 只接管热路径和 Smoke 级检查。

### 2.2 Go CLI 只做确定性检查

Go CLI 不做 AI 推理，不做复杂 repair loop，不碰 TI DSS，不写 flash，不改用户文件。V1 只做：

- Git governance fast lint。
- staged-only DocSync fast gate。
- commit-msg 检查。
- kit registry / manifest 解析。
- kit Smoke verify/test。
- doctor perf 基础测速。

### 2.3 分层执行

```text
L0 Git hook / hot path
  bin/aicoding hook pre-commit
  bin/aicoding hook commit-msg

L1 local Smoke
  bin/aicoding kit verify --all --profile Smoke
  bin/aicoding governance lint
  bin/aicoding doctor perf

L2 local Full
  pwsh scripts/aicoding-kit.ps1 test -All -Profile Full -Json

L3 CI / Release
  pwsh scripts/test-kit-fresh-clone.ps1 -Profile Release -Json
  pwsh scripts/aicoding-kit.ps1 export -All -Zip -Json
```

## 3. Kit 分层约束

| Kit | 当前角色 | V1 快路径处理 | 仍保留在慢路径 |
|---|---|---|---|
| `aicoding-platform` | Codex plugin / CodingKit marketplace / submodule 校验 | manifest、路径存在、Smoke shape 检查 | install、update、真实 Codex marketplace 注册 |
| `agent-patch-kit` | apatch 安全补丁、扫描、事务快照 | manifest、plugin/skill 路径存在、Smoke shape 检查 | `apatch doctor`、真实 patch/scan/summary |
| `ai-debug-repair-kit` | build/test repair loop、TI DSS 只读调试 | manifest、script 路径存在、Smoke shape 检查 | Python repair loop、pytest、DSS profile/硬件相关命令 |
| `codex-agent-powershell-skill-kit` | PowerShell AST、安全改写、PSScriptAnalyzer | manifest、script 路径存在、Smoke shape 检查 | AST 深度分析、PSScriptAnalyzer、自动安装工具 |
| `docsync-plus` | 文档同步/漂移门禁 | staged-only doc requirement 快速检查 | semantic drift、link drift、完整 policy、CI all/release |
| `aicoding-agent-dev-kit` | Spec/TDD/Plan Mode/hooks/quality gate | manifest、script 路径存在、Smoke shape 检查 | Python CLI status、质量门禁、Plan Mode 复杂校验 |
| `common-control-kit` | C99 控制模块资产 | builtin required paths 检查 | 模块级静态分析、单元测试、报告生成 |

## 4. 目录约束

新增目录：

```text
cmd/aicoding/main.go             # Go native CLI；V1 单文件实现，后续可拆 internal/*
go.mod                           # 无第三方依赖，避免首次下载慢
scripts/aicoding-fast.ps1         # PowerShell 薄封装；兼容 Windows 使用习惯
.github/workflows/fast-path.yml   # Linux runner 上的快速 smoke CI
```

修改目录：

```text
.githooks/pre-commit              # 优先调用 bin/aicoding / go run；失败再回落 pwsh
.githooks/commit-msg              # 同上
```

不改动：

```text
scripts/aicoding-kit.ps1
scripts/lib/AiCoding.*.psm1
dist/**
CodingKit/**
config/kits/*.json
```

## 5. 命令接口

```powershell
# 构建
mkdir bin
# Windows
 go build -o bin/aicoding.exe ./cmd/aicoding
# Linux/macOS
 go build -o bin/aicoding ./cmd/aicoding

# 快速 hook
bin/aicoding.exe hook pre-commit
bin/aicoding.exe hook commit-msg --file .git/COMMIT_EDITMSG

# 快速 kit smoke
bin/aicoding.exe kit list --json
bin/aicoding.exe kit doctor --json
bin/aicoding.exe kit verify --all --profile Smoke --json

# 性能诊断
bin/aicoding.exe doctor perf --json
```

## 6. V1 明确不做的事

- 不替代 Full / Release。
- 不替代 `scripts/check-documentation-sync.ps1 -Mode all|ci|release`。
- 不运行 pytest。
- 不运行 `python -m ai_debug_repair...`。
- 不运行 `apatch doctor`。
- 不运行 PSScriptAnalyzer。
- 不连接硬件、不运行 DSS。

## 7. 后续 V2 拆分建议

V1 是单文件，便于直接落地。稳定后拆为：

```text
internal/git
internal/governance
internal/docsyncfast
internal/kitregistry
internal/manifest
internal/perf
internal/report
```

V2 再加入 cache：

```text
.aicoding/cache/kit-index.json
.aicoding/cache/repo-index.json
.aicoding/cache/perf.json
```

V3 再把部分 PowerShell `Smoke` 脚本替换为 declarative manifest check；Full/Release 仍保留原脚本。

## 8. 验收标准

本地：

```powershell
go test ./...
go build -o bin/aicoding.exe ./cmd/aicoding
bin/aicoding.exe kit verify --all --profile Smoke --json
bin/aicoding.exe governance lint --json
bin/aicoding.exe doctor perf --json
```

Git hook：

```powershell
git config core.hooksPath .githooks
git commit
```

CI：

- `fast-path.yml` 通过。
- 原 `docs-sync.yml` 继续保留，用于完整 DocSync。

## 9. 推荐合并策略

1. 先合并 Go CLI、wrapper、fast-path CI。
2. 本地构建 `bin/aicoding.exe`，测试 hook。
3. 观察 `doctor perf` 输出。
4. 一周后再考虑把更多 Smoke 检查从 PowerShell 搬进 Go。
