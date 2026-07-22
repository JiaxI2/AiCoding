---
id: toolchain-semantic-identity
status: approved
approvedTree: "f2778dface7d2c0fde1f01de7cb43ff981f51812"
scope:
  - internal/validationevidence/**
  - internal/testengine/evidence_test.go
  - docs/decisions/0007-validation-evidence.md
  - docs/operations/VALIDATION_EVIDENCE_BUDGET.md
  - docs/operations/evidence/toolchain-digest-v2-matrix.md
  - docs/todolist/0032-toolchain-semantic-identity.md
  - docs/todolist/done/0032-toolchain-semantic-identity.md
  - CHANGELOG.md
gates:
  - profile: full
  - profile: release
---

# toolchainDigest.v2 语义身份计划

## 目标

把 Validation Evidence 的 toolchain 身份从“工具文件位置与时间戳”收敛为稳定的版本语义，
使同平台、同架构、同 Go/Git 版本的 Git Bash 与 PowerShell 可以互认 Receipt。路径、大小与
mtime 只负责判定本地 probe cache 是否仍可使用，不进入 Receipt 身份。

## 已批准范围

- `toolchainDigest.v2` 以显式域分隔、算法版本、规范化的 `go version` / `git --version`
  输出和 `GOOS/GOARCH` 形成语义 digest。
- 每次解析当前 Go/Git 绝对路径及 size/mtime；键不变才读取缓存版本输出，键变化必须重探。
  v1、损坏或语义不完整的 cache 均不得提供身份，只能从真实 probe 重建。
- probe 启动失败、非零退出或输出无法解析时返回
  `VALIDATION_FINGERPRINT_INVALID`，不得复用旧 cache 或 Receipt。
- ADR 0007 同批区分“普通工具版本变化”和“fingerprint 算法契约变化”；BUDGET 的晋级计数
  因 v1→v2 换域归零。
- 测试可向包内私有 probe 注入路径、版本与平台/架构，仅用于可重复验证；不增加 CLI、
  配置字段、Receipt 类型或新的治理领域。

## 不变量

- `Fingerprint` 字段集合与顺序不变，唯一 `Receipt` 权威不变。
- `test --reuse` 默认值保持 `off`；本项不累计 v2 的 1/3，不启动晋级评审。
- 版本变化只产生普通 identity miss；旧 v1 Receipt 在 v2 下同样是 miss，
  `--verify-reuse` 不得把算法换域误报为 corruption/audit mismatch。
- cache 中的路径、size、mtime 只参与完整性和重探判定，不得进入 semantic digest。
- 不触碰 `CodingKit/agents/skills`、TODO 0019 或与本项无关的验证实现。

## 验证

包内测试与真实命令共同覆盖：版本变化、等版本换路径/mtime、平台/架构注入、probe 失败与
乱码、cache 损坏重建、v1 Receipt 普通 miss，以及 PowerShell seed → Git Bash check 的首次
跨 shell 命中。保留原始命令输出；最终运行 Full 与 Release，并确认默认 `--reuse off`、
`Fingerprint` 字段集合和唯一 `Receipt` type 均未漂移。
