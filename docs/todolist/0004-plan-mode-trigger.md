# TODO 0004: Plan Mode 重构 I —— 触发机器化（plan-policy + plan check）

Status: Planned
Verify: go test ./internal/plan/... 且 bin/aicoding.exe plan check --staged --json 在敏感/非敏感两组 staged 变更上给出正确判定

> Plan Mode 重构三部曲之一（0004 触发 → 0005 产物 → 0006 绑定），合并为 ADR 0009。
> 现状诊断：触发链是 agent-hook → plan-mode-gate.ps1 → spawn pwsh 跑 237 行验证脚本，
> "架构敏感"pattern 埋在 ps1 第 176 行 —— **靠 Agent 自觉，不是 git hook 强制**。

## 设计原则（照搬已验证的三板斧）

```text
validationevidence:  验证绑定内容，hook 只查不跑
loop engineering:    裁决而非执行
plan mode:           批准绑定内容，触发机器可判，产物有 schema
```

## 实现计划

1. 新增 `config/plan-policy.json`（与 validation-policy.json 同构的 pattern 语义）：

   ```json
   {
     "schemaVersion": 1,
     "sensitivePaths": [
       { "pattern": "internal/cli/**",              "reason": "frozen kernel" },
       { "pattern": "internal/lifecycle/**",        "reason": "frozen kernel" },
       { "pattern": "internal/runner/**",           "reason": "frozen kernel" },
       { "pattern": "internal/report/**",           "reason": "frozen kernel" },
       { "pattern": "internal/registry/**",         "reason": "frozen kernel" },
       { "pattern": "internal/testengine/**",       "reason": "frozen kernel" },
       { "pattern": "config/schemas/**",            "reason": "frozen schema" },
       { "pattern": "docs/architecture/**",         "reason": "architecture authority" },
       { "pattern": "config/kit-registry.json",     "reason": "kit activation" },
       { "pattern": ".githooks/**",                 "reason": "hook execution boundary" }
     ],
     "exemptPaths": ["docs/spec/**", "docs/todolist/**"]
   }
   ```

   配套 `config/schemas/plan-policy.schema.json`。
2. 新增 `internal/plan` 包（只依赖 gitx/platform/report；**不依赖 loopkit**，两者是兄弟：
   plan 管"要不要做、什么范围"，loop 管"迭代到何时停"）。第一刀只做：
   - `LoadPolicy(repo)`：读 + schema 校验 + pattern 去重排序（确定性）。
   - `CheckPaths(policy, paths) → {sensitive:[{path,reason}], exempt:[]}`：纯函数。
3. CLI `aicoding plan check [--staged | --paths P ...] --json`：
   - `--staged` 用 `gitx.StagedFiles`（已有），单次 Git 调用。
   - 命中敏感且无 approved plan 覆盖（0006 之前先只报命中，不查 plan 覆盖）→
     `ok:false, errorKind:validation`，`requiredAction` 给出建 plan 的命令。
   - **同步 `CommandPlan` + HelpForm + docs/COMMANDS.md**（缺 HelpForm 启动即 panic）。
4. pre-commit 接线：`.githooks/pre-commit` 现有链尾追加 `plan check --staged`，
   第一阶段 **warn**（输出但不 exit 1），0006 完成后再升 enforce。
5. `tools/specialty/hooks/aef/plan-mode-gate.ps1` 降级为薄壳调
   `bin/aicoding.exe plan check --staged --json`（ps1 面已冻结，只减不增）。
   `verify-agent-dev-kit-plan-mode.ps1` 中的敏感判定逻辑标记 deprecated，留一个
   release 周期后删（沿用 verify-codex-kit 退役节奏）。

## 明确不做

- 不在本项实现 approve/漂移检测（归 0006）。
- 不做 plan 产物迁移（归 0005）。
- pre-commit 不跑任何测试/验证（毫秒级路径判定 only）。

## 自测（可信任方式）

```powershell
go test ./internal/plan/... ; go vet ./...
# 表驱动单测必须覆盖：敏感命中/exempt 豁免优先/glob 边界（internal/cli/x/y.go 命中 internal/cli/**）
# /pattern 非法时 LoadPolicy fail-closed

# 端到端正反两例：
git stash ; git checkout -b tmp-plan-check-test
echo x >> internal/cli/cli.go ; git add internal/cli/cli.go
bin\aicoding.exe plan check --staged --json          # 期望 ok:false，列出 internal/cli/cli.go + reason
git restore --staged internal/cli/cli.go ; git checkout internal/cli/cli.go
echo x >> docs/todolist/0004-plan-mode-trigger.md ; git add docs/todolist/
bin\aicoding.exe plan check --staged --json          # 期望 ok:true（exempt）
git checkout - ; git branch -D tmp-plan-check-test

# 性能（pre-commit 路径）：
1..5 | % { (Measure-Command { bin\aicoding.exe plan check --staged --json }).TotalMilliseconds }
# 中位数必须 < 200ms

bin\aicoding.exe governance dependencies --json      # plan 包依赖方向合法
bin\aicoding.exe test --profile Full --json
```

通过判据：正反两例判定正确；中位数 <200ms；`internal/plan` 不 import loopkit/testengine；
Full 全绿。
