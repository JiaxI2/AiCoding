# TODO 0032: toolchainDigest 语义化与复用晋级计数换域

Status: Planned
Verify: toolchainDigest.v2 只绑定版本语义与平台，路径/mtime 仅使 probe cache 重探；七项正负例真跑，旧 v1 Receipt 普通 miss，默认 reuse 保持 off

## 背景

`docs/operations/VALIDATION_EVIDENCE_BUDGET.md` §12 已记录：当前 toolchain 身份把
Go/Git 可执行文件绝对路径与 mtime 纳入 digest，使同机 Git Bash 与 PowerShell 即使使用
同版本工具也无法互认 Receipt。本项只修复该身份过严问题，不新增 Receipt 类型或复用入口。

## 契约

1. `toolchainDigest.v2` 的语义输入固定为域/版本标识、规范化后的 `go version`、
   `git --version` 与平台/架构。
2. probe cache 键只使用解析后的绝对路径、size、mtime；键变化必须重探，但相同版本语义
   仍产生相同 digest。
3. probe 失败、版本输出不可解析与缓存损坏均 fail-closed；损坏缓存可重建，但不得使用其中
   的身份。
4. Fingerprint 字段集合、唯一 `Receipt` type 与默认 `--reuse off` 均不变。
5. ADR 0007 区分普通 toolchain 版本变化与 fingerprint 算法契约变化；后者使晋级计数
   换域并从 0/3 重新累计。

## 真跑矩阵

- Git/Go 版本语义变化：digest 变化。
- 同版本换路径或 touch mtime：cache 重探，digest 不变。
- Git Bash 与 PowerShell 同版本工具：首次跨 shell Receipt 命中。
- 平台/架构注入变化：digest 变化且拒绝旧复用。
- probe 不可执行或输出不可解析：非零、fail-closed。
- probe cache 损坏：拒绝旧缓存并成功重建，不产生错误身份。
- v1 Receipt 在 v2 下：普通 miss，`--verify-reuse` 不报 corruption。

## 文档与验收

- ADR 0007 与实现同批修订，不后补。
- BUDGET §12 保留历史限制并标为已解决；§13 保留 run 29900035150，但标为 v1 历史证据，
  v2 计数从 0/3 开始。
- 首次 Full/Release 因换域全冷属于预期；收益只表述为 warm reuse 与跨 shell 命中率提升。
- 最终 Full、Release 各全绿一次，summary 路径写入本条目后归档。

## 明确不做

- 不翻转 `--reuse` 默认值，不启动新的 3/3 晋级评审。
- 不改 Fingerprint 字段集，不新增 Receipt 权威或治理领域。
- 不触碰 `CodingKit/agents/skills` 与 TODO 0019。
