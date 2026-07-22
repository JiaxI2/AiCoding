# TODO 0034: PWSH-002 PowerShell 数量与标记棘轮

Status: Done
Verify: A 落地后实测基线写入配置；PWSH-002 阻断 unspecified 与脚本数回升，PWSH-001 仍只报告

## 顺序依赖

本项必须在 TODO 0033 提交后执行。`remainingScripts` 基线不得预填：先用已提交的 A tip
真跑 `doctor pwsh --json`，把实测值和 `unspecified=0` 连同原始输出写入本项证据，再将
相同实测值写入配置。数值不是契约；契约是“基线等于落地时实测、只降不升”。

## 契约

1. `doctor pwsh` / PWSH-001 的 report-only 语义不变，`unspecified` 不改变该命令退出码。
2. 既有 `doctor pwsh-budget` / PWSH-002 读取配置基线：
   - `unspecified > 0`：非零并指出缺标记文件；
   - `remainingScripts > baseline`：非零并指出新增脚本；
   - 删除脚本后，仅允许同一提交把 baseline 单向下调；baseline 上调必须失败。
3. 不为 deprecated、thinShell 增加新规则，不新增治理命令或领域。

## 真跑负例

- 临时新增无 `RETIRE-AFTER` 的 `.ps1`：PWSH-002 非零且指出文件。
- 在实测基线下临时新增带合法 `RETIRE-AFTER` 的 `.ps1`：即使 `unspecified=0`，也因
  `remainingScripts > baseline` 非零且指出文件。
- 临时上调配置 baseline：PWSH-002 非零；恢复后重新全绿。

## 验收

- 配置提交包含生成基线所用的 `doctor pwsh` 原始输出与 commit/tree 身份。
- 单测覆盖实测基线、缺标记、数量回升、单向下调及上调拒绝。
- PWSH-001/PWSH-002 文档边界、COMMANDS、全局用例与 CHANGELOG 同步。
- 最终 Full、Release 各全绿一次，summary 路径写入本条目后归档。

## 实施证据

- 顺序基点：Phase 2 证据提交 `f56c17e1b8be8723fc4f884cdf537f2f9cd959cd`；原始输出保存于
  `docs/operations/evidence/pwsh-budget-baseline-f56c17e.json`，实测为
  `remainingScripts=19 / thinShells=1 / deprecated=1 / unspecified=0`。
- PWSH-001 保持 report-only：临时新增无标记脚本后仍 `exit=0`，报告
  `remainingScripts=20`；同一输入的 PWSH-002 `exit=1`，原始错误同时点名
  `tools/specialty/ratchet-negative.ps1`、缺少 `# RETIRE-AFTER:` 与 `20 != 19`。
- 带合法 `# RETIRE-AFTER:` 的同名第 20 个脚本仍使 PWSH-002 `exit=1`，错误点名该路径及
  `PowerShell remainingScripts=20 does not equal baseline=19`，未误报缺标记。
- 临时把配置基线抬至 20 时 PWSH-002 `exit=1`，原始错误分别指出脚本清单只有 19、
  证据不能证明 20、当前值 19 不等于基线 20；三次探针均已还原。
- B 独立 Full：`71 total / 67 pass / 0 fail / 0 warn / 4 skip`，
  `test-results/aicoding-global-test-20260722-174408/summary.json`；PWSH-001/PWSH-002 均 PASS。
- 本轮最终 Full：`test-results/0032-final-full/summary.json`；最终 Release：
  `test-results/0032-final-release/summary.json`。二者对包含 A/B/C 归档的同一 staged tree
  真跑；PWSH-001 report-only 与 PWSH-002 ratchet 均由最终 profile 再验收。

## 明确不做

- 不使 PWSH-001 失败。
- 不新增 deprecated/thinShell 棘轮，不主动加速剩余兼容脚本退役。
- 不触碰本轮以外的 PowerShell 实现。
