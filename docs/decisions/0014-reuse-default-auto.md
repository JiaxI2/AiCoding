# ADR 0014: `--reuse` 默认值晋级为 `auto`

PrimitiveReview: n/a

## Status

Accepted。只翻转既有 `test` 命令与 `testengine.Config` 的默认复用模式，并补强损坏
Receipt 在默认 `auto` 路径上的 fail-closed 退出；不增加 Profile、Test Registry leaf、
Receipt 类型、配置字段、治理领域或外部授权凭证。

## 1. Context

ADR 0007 §5 要求新身份方案在三棵不同的 main Tree 上连续完成远端 Release
`--reuse off` seed 与 `--verify-reuse` 全量审计，然后才能由独立提交晋级。Validation
Evidence Budget §13 已记录 `toolchainDigest.v2` 的 3/3：

1. [run 29916523297](https://github.com/JiaxI2/AiCoding/actions/runs/29916523297)，
   `main@41eefac7a67ac1473a5b9cf7cfc6548ca7372027`，Tree
   `529ef271c491c717202a19b10fa7127a36d83c73`；
2. [run 29921228586](https://github.com/JiaxI2/AiCoding/actions/runs/29921228586)，
   `main@48a355c32941bca2a01eb1f95e3c78c6af3f8090`，Tree
   `7afa0605ec602f040408f942de43ad6fad013979`；
3. [run 29922476097](https://github.com/JiaxI2/AiCoding/actions/runs/29922476097)，
   `main@44a99d13b9d9b84181318b7423a22595939438cd`，Tree
   `878cae97795ac7e62b21f4deee215d76d1ffb420`。

三次均来自不同 main Tree 和独立 runner，且 seed/audit 两段结论一致。ADR 0007 的
fingerprint 换域重置条款因此已经满足。

`toolchainDigest.v2` 的身份只绑定显式 domain/version、规范化的 `go version` /
`git --version` 与平台/架构。Go/Git 可执行文件的解析路径、size 与 mtime 只作为本机 probe
cache 键；键变化会强制重探，但只有版本语义变化才改变 Receipt 身份。probe 失败、不可解析
输出或损坏 cache 不能提供身份。

## 2. Decision

`test --profile ...` 的未显式 `--reuse` 默认值从 `off` 翻转为 `auto`。显式
`--reuse off`、`--force` 与 `--verify-reuse` 的行为不变；`test --profile Full` 命令继续存在。

默认值有且只有以下两个规格锚点：

- `internal/testengine/evidence_test.go` 的
  `TestNormalizeConfigDefaultsReuseAutoAndRejectsAuditForce`；
- `internal/cli/validation_test.go` 的
  `TestTestCommandWiresExplicitEvidenceFlagsAndDefaultsAuto`。

两处都对确定的 `auto` 做精确相等判断。今后任何默认值变更都必须同步这两处；不得把断言
放宽为任意值、非空或与默认值无关的测试。

## 3. `auto` 语义边界

命中必须同时满足：

1. 主体是干净 HEAD，或只有 index staged 内容且工作区相对 index 干净；
2. repository、Tree、profile、validation plan、engine semantics、相关配置、
   `toolchainDigest.v2` 与 options 的完整 identity 精确一致；
3. Receipt、保留报告及逐 leaf `(id,status)` 摘要全部通过完整性校验。

Tree、profile、工具链版本、配置、选项或验证语义任一变化，都映射到不同 identity，作为普通
miss 真跑。tracked 工作区变化、非 ignored untracked、unmerged 或 dirty submodule 使主体
不可复用；该次可执行但不得查询、命中或发布 Receipt。跨 profile 的 Receipt 永不互用。
`--reuse off` 和 `--force` 都强制执行全部选中 leaf，`--verify-reuse` 始终执行并审计实际状态。

普通 miss 与损坏必须区分：精确路径不存在时可以真跑并在成功后发布新 Receipt；精确路径
存在但 Receipt/报告损坏、fingerprint 非法或 store 读取失败时，默认 `auto` 必须立即非零退出，
不得把损坏静默降为 miss、不得执行后覆盖或写出错误 Receipt。执行失败永不产生 PASS Receipt。

## 4. CI 行为不变

`.github/workflows/aicoding-ci.yml` 的 `release-gate` 继续逐字执行：

```powershell
.\bin\aicoding.exe test --profile Release --reuse off --json
.\bin\aicoding.exe test --profile Release --verify-reuse --json
```

因此默认值翻转不会改变远端冷 seed 与全量 audit；两条显式命令仍是后续回归和回滚证据。

## 5. 回滚

以下任一事实出现一次即成立，不设观察窗口：

1. Git Tree 已变化却命中旧 Receipt；
2. 任意真实测试失败被 Receipt 掩盖为绿。

触发后必须立即把 CLI、testengine parse 与 `NormalizeConfig` 三处默认值以及本 ADR 登记的
两处规格锚点一起翻回 `off`；CI 的显式 seed/audit 命令保持不动。调查完成前不得以新增例外、
放宽 identity 或清空断言恢复 `auto`。

## 6. Verification

- 两个默认值锚点在默认值人为改成第三值 `force` 时均真实失败，恢复后通过；
- 八项负例矩阵逐项保留原始输出，尤其真实断言失败在已有同 profile Receipt 时仍必须 FAIL；
- 同一 Tree 连跑两次默认 `auto` Release，首次冷执行，第二次 `cache_hit_ratio > 0` 且结论一致；
- 最终 exact commit 以 Release 真跑全绿，并通过 DocSync、governance、plan 与 todolist 门禁。

## §12 Checklist 自评

本 ADR 复用既有 Validation Evidence Primitive、唯一 Test Registry、Receipt store 与报告
schema。新增的 fail-closed 分支只把已经识别为 invalid/store error 的复用决定升级为非零退出，
不建第二套 runner、缓存或策略面；普通 miss、显式 off、force、audit 与 push gate 均不改变。
