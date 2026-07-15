# 外部 C 文件验证

`cstylekit verify` 用一个固定、可计时的主机验证流水线检查 Kit 仓库之外的 C99
源文件和头文件。它与 `scripts/verify.ps1`、`scripts/verify.sh` 的 Kit 发布门禁职责不同。

## 安全边界

外部验证只查找并调用 `gcc`、`g++`、`clang`、`clang++`，并执行本次验证在系统
临时目录中生成的 host test。所有编译器与 host test 的当前工作目录均固定为该临时目录。
目标 manifest 不能提供命令、脚本或构建步骤，因此该入口本身不会：

- 调用 TI 编译器、CCS、工程 `gmake` 或固件构建；
- 因相对路径而把编译产物写入候选源码所在目录；
- 执行 manifest 中的任意 shell 文本；
- 把 C Kit 发布门禁重复应用于外部文件。

host harness 是调用者提供的受信 C 源，不是安全沙箱。它以当前用户权限运行，仍可能主动使用
绝对路径、系统调用或其他 C 接口访问外部资源；不要使用来源不明的 harness。

## 命令

```powershell
cstylekit verify `
  --config F:\Study\AI\c-userstyle-kit\examples\c-kit.json `
  --overlay .\project.c-kit.overlay.json `
  --target .\project.verify.json `
  --profile fast `
  --timings `
  --json
```

`--overlay` 可以重复。覆盖文件只写相对基础配置发生变化的字段；对象递归合并，数组整体
替换。`schema`、`standard` 和 `reference` 由基础配置锁定，覆盖文件不能包含 `null` 或未知字段。
verify 还会强制保留严格 C99/C++17 核心告警参数，并拒绝空 flag 集、`warningsAsErrors=false`
以及任何白名单外参数；因此不能通过 overlay 注入 `-fplugin`、`-specs`、`-wrapper` 或响应文件。

目标 manifest 中的相对路径以 manifest 所在目录为基准。显式目标若命中基础配置的
`scope.exclude`，验证直接失败，不能静默跳过。

`scope-hash` 读取每个显式源文件和头文件后，会把相同字节复制到临时快照目录。lint、编译、
头文件探针和行为测试随后只引用快照路径，报告中的原路径与 SHA-256 则保持不变。为避免重新
读取可变的原 include 目录，所有非系统本地头文件必须通过 `candidate.header`、
`baseline.header` 或 `host.supportHeaders` 显式列出；同名但内容来源不同的文件会被拒绝。

## Profile

`fast` 执行：范围与 SHA-256、lint、可读性摘要、GCC 严格 C99、GCC C99 头文件探针和
候选 host test。

`full` 包含 `fast`，并增加 Clang 严格 C99、Clang C99 头文件探针、G++/Clang++ C++17
头文件探针，以及 original/candidate 行为等价检查。

host test 必须由目标提供。完整模式使用同一个 harness 分别链接 baseline 与 candidate，
要求两边退出码为零且标准输出逐字节相同。建议 harness 输出稳定 JSON 或测试向量摘要。
缺少 harness 时，要求 host test 的 profile 必须失败，工具不能用“未配置”冒充通过。

## 目标 manifest

```json
{
  "schema": "cstylekit-verify-target-v1",
  "id": "pdo-dynamic",
  "candidate": {
    "source": "pdo_dynamic_refactored.c",
    "header": "pdo_dynamic_refactored.h"
  },
  "baseline": {
    "source": "pdo_dynamic.c",
    "header": "pdo_dynamic.h"
  },
  "host": {
    "testSource": "tests/pdo_dynamic_host_test.c",
    "supportSources": [],
    "supportHeaders": ["tests/stubs/ecat_def.h"],
    "defines": []
  }
}
```

`fast` 不要求 `baseline`；`full` 的行为等价步骤要求 `baseline.source` 与
`baseline.header` 同时存在。`supportSources` 和 `supportHeaders` 只用于 host harness
与平台桩，不参与候选 lint。
