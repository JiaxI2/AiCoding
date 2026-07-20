# Validation Evidence 性能预算

## 1. 基线身份

- 测量日期：2026-07-20
- Git SHA：`520e14b84805260ebca03b0eb438b08ffb243552`
- Git：`2.48.1.windows.1`
- Go：`go1.26.4 windows/amd64`
- 平台：Windows，PowerShell
- 仓库：`AiCoding-main` 独立 worktree，工作区无修改

## 2. 测量方法

在承诺 `validation check` 的 SLA 前，按同一 worktree 连续执行五次下列 Git/Go 调用；
每项取中位数。命令输出重定向为空，只计进程启动、Git/Go 执行和返回的墙钟时间。

```powershell
foreach ($i in 1..5) {
  Measure-Command { git rev-parse "HEAD^{tree}" } | % TotalMilliseconds
  Measure-Command { git status --porcelain --ignore-submodules=none } | % TotalMilliseconds
  Measure-Command { git status --porcelain } | % TotalMilliseconds
  Measure-Command { git write-tree } | % TotalMilliseconds
  Measure-Command { go version } | % TotalMilliseconds
}
```

`git write-tree` 会向 Git object database 写入 tree 对象，但不修改工作区、index 或 HEAD；
该调用不是纯查询。

## 3. 第 0 期实测

| 调用 | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | 中位数 |
|---|---:|---:|---:|---:|---:|---:|
| `git rev-parse HEAD^{tree}` | 115.017 ms | 67.559 ms | 57.142 ms | 149.873 ms | 69.480 ms | **69.480 ms** |
| `git status --porcelain --ignore-submodules=none` | 288.374 ms | 239.578 ms | 186.153 ms | 178.582 ms | 168.828 ms | **186.153 ms** |
| `git status --porcelain` | 170.187 ms | 181.531 ms | 176.504 ms | 344.951 ms | 211.314 ms | **181.531 ms** |
| `git write-tree` | 88.307 ms | 83.929 ms | 61.246 ms | 64.144 ms | 75.871 ms | **75.871 ms** |
| `go version` | 164.539 ms | 61.553 ms | 65.648 ms | 65.009 ms | 66.120 ms | **65.648 ms** |

带子模块脏检测的 `git status` 中位数为 `186.153 ms`，低于方案规定的 `400 ms` 停止线，
因此 Validation Evidence 第一期可以继续。HEAD 检查的两个主要 Git 调用中位数合计为
`255.633 ms`；独立执行 `git submodule status --recursive` 不进入检查路径。

## 4. SLA 与实现预算

第一期采用以下 warm-cache SLA：

```text
validation check --target HEAD --json 的 5 次墙钟中位数 <= 300 ms
```

预算只允许一次 `git status --porcelain --ignore-submodules=none`、一次
`git rev-parse <rev>^{tree}`、一次 toolchain cache 读取和一次内容寻址 Receipt `os.Stat`。
不得递归查询 submodule、不得扫描 Receipt 目录、不得逐文件哈希、不得哈希 CLI 二进制。

`--target INDEX` 额外执行一次 `git write-tree`，其 SLA 在第一期实现完成后单独实测回填，
不沿用 HEAD 目标的数字。

## 5. 回填规则

第一期完成后必须在同一环境重建 `bin/aicoding.exe`，分别对 Receipt miss/hit 执行五次
`validation check`，把真实样本和中位数回填本文件。若 HEAD hit 中位数超过 `300 ms`，
先用调用计数与 profile 定位超额来源；在新证据获得评审前，不提高 SLA，也不启用默认复用。

## 6. 第一期实现回填

- 实现测量 Git SHA：`c03076f`；
- 二进制：按该 SHA 重新执行 `go build -o bin\aicoding.exe ./cmd/aicoding`；
- 目标：干净 linked worktree 的 Release profile，warm toolchain cache；
- 计时边界：PowerShell 从启动 `aicoding.exe` 到进程退出的外部墙钟，命令输出重定向为空；
- HEAD 路径只执行一次 porcelain-v2 status 和一次 `HEAD^{tree}`，两者并发；常规 `.git`/
  `commondir` 直接解析，异常布局才回退 `git rev-parse --git-common-dir`。

| 场景 | Run 1 | Run 2 | Run 3 | Run 4 | Run 5 | 中位数 | 结论 |
|---|---:|---:|---:|---:|---:|---:|---|
| HEAD Receipt miss | 262.284 ms | 362.701 ms | 228.590 ms | 266.570 ms | 249.938 ms | **262.284 ms** | PASS |
| HEAD Receipt hit | 304.146 ms | 250.578 ms | 333.665 ms | 229.967 ms | 285.355 ms | **285.355 ms** | PASS |
| INDEX Receipt hit | 293.992 ms | 300.286 ms | 385.933 ms | 268.734 ms | 300.930 ms | **300.286 ms** | 参考值 |

首版串行实现的 warm miss 中位数为 `386.390 ms`，超额主要来自额外的 common-dir Git
进程（独立中位数 `72.890 ms`）以及 status/TreeOID 串行等待。移除额外进程并并发两个必需
只读 probe 后，HEAD miss/hit 均通过原定 `300 ms` SLA；没有提高 SLA，也没有删除
untracked 或 submodule dirty 检查。INDEX 会执行 `git write-tree`，本期只记录参考值，不建立
HEAD SLA 的错误等价关系。

## 7. 第一期功能验收

2026-07-20 在同一干净 SHA 上执行：

| 验收路径 | 结果 | 墙钟/报告耗时 |
|---|---|---:|
| `test --profile Release --reuse off` | executed，58/58 PASS，生成可复用 Receipt | 171.026 s |
| `validation check --profile Release --target HEAD` | `VALIDATION_RECEIPT_HIT` | 报告 225 ms |
| `test --profile Release --reuse auto` | reused，58/58 PASS，同一 Receipt | 外部 392.763 ms |
| Smoke seed | executed，40/40 PASS，可复用 | 24.162 s |
| Smoke `--reuse auto --force` | executed，40/40 PASS，同一 Receipt | 23.089 s |
| Smoke `--reuse auto --verify-reuse` | executed，40/40 PASS，`VALIDATION_RECEIPT_HIT` | 22.893 s |

Release Receipt ID 为
`sha256:3e978d3eea94bcd083dfde8800f6505f03d3684a193be56ff28a90fd195c009b`；Smoke Receipt
ID 为 `sha256:10849976d0e6940fca527fa57fe51a54384bce5b49e28d36bab0c3323e9b603b`。
默认值仍是 `--reuse off`。本次验收不启用 Hook、不切换默认复用，也不进入 profile 继承。
