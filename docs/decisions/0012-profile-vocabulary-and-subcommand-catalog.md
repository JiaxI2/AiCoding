# ADR 0012: `--profile` 词汇表正交化与子命令目录冻结

PrimitiveReview: n/a

## Status

Accepted。本 ADR 只扩展既有 `internal/cli` typed command catalog，不新增 Primitive、治理领域、
生命周期或报告权威。

## 1. 已确认缺陷

修复前 `internal/cli/cli.go` 的 `kit verify --help` 把 `--profile` 描述为
`Smoke, Full or Release`，运行时却只接受 `Smoke/Lifecycle`；真实执行
`kit verify --all --profile Full` 以 usage error、exit 2 失败。与此同时，
[FREEZE_AND_ACQUISITION_BOUNDARY](../architecture/FREEZE_AND_ACQUISITION_BOUNDARY.md) 与
[Roadmap](../architecture/07-roadmap.md) 已把产品测试仅有 Smoke/Full/Release 三档写为地基契约，
但 CLI 实际还让 `--profile` 承载了 `Smoke/Lifecycle` 与 `fast/full` 两套领域词汇。

修复前的 help、运行时输出与退出码原文保存在
[`profile-vocabulary-negative-matrix.md`](../operations/evidence/profile-vocabulary-negative-matrix.md)。

## 2. Decision

1. `--profile` 只表达产品级 `Smoke|Full|Release`，大小写规范化后进入现有 testengine 语义；
   不允许登记第四套 `--profile` 词汇。
2. `skill c99-standard-c verify` 的验证强度改用 `--depth fast|full`。
3. `kit verify` 的管理检查强度改用 `--level smoke|lifecycle`。
4. `kit test` 从隐藏 alias 提升为 `kit` 的正式子命令。它已有真实 manifest/quickstart 消费者，
   当前只有一种 Smoke 行为，因此正式形式不带 `--profile`。
5. typed command catalog 扩展为顶层命令、子命令及 alias 的唯一登记面。外部 argv 先由 catalog
   解析并规范化，再交给既有 handler；help 的命令/子命令前缀与 pluginview quickstart 路由均
   从同一 catalog descriptor 投影，领域实现不再各自登记命令字符串。
6. 新冻结面是“子命令与 alias 的唯一登记”，不是把既有“顶层命令唯一登记”重新解释为已覆盖；
   FREEZE 家族分别阻断 catalog 外路由与非产品 `--profile` 词汇。

## 3. 兼容退役窗口

依据 [Primitive Constitution §8](../architecture/PRIMITIVE_CONSTITUTION.md)，以下旧形式在窗口内
继续成功并输出 deprecation warning：

- `skill c99-standard-c verify --profile fast|full` → `--depth fast|full`；
- `kit verify --profile Smoke|Lifecycle` → `--level smoke|lifecycle`；
- `kit test --profile Smoke` → 删除该参数，直接运行 `kit test`。

旧参数与新参数同时出现时因语义歧义返回 usage error。旧形式不会出现在 canonical help、
Taskfile、capability quickstart 或当前运维文档中，但保留直接回归测试。

移除不得早于下一个主版本 `v2.0.0`，且必须同时满足：

1. 至少两个已发布版本持续输出 warning，并在各自 release notes 中记录迁移；
2. 当前 README/COMMANDS/Taskfile/capability/pluginview 投影已无旧形式；
3. 兼容正例与真实仓库搜索证明没有受管调用方仍依赖旧形式；
4. owner 批准独立 ADR/变更，明确迁移、回滚和删除后的负例。

本轮不移除任何旧形式；历史 ADR/TODO 中的原始命令记录不回写。

## 4. 一致性与失败语义

- catalog 构造时校验命令、子命令、alias、help route 和 quickstart route；重复、悬空或 catalog
  外引用直接失败并点名命令路径。
- canonical help 中凡出现 `--profile`，接受值只能是 `Smoke|Full|Release`；运行时通过同一产品
  profile 规范化器拒绝其它值。
- pluginview 只消费 catalog 投影出的结构化 route，再把 manifest kit ID 填入参数；不持有
  `aicoding kit test ...` 等完整命令字符串。
- catalog snapshot 继续是确定性、可 detached 的 evidence input；新增子命令面进入同一 digest。

## 5. 验证与回滚

验证包括 help/runtime 一致性、catalog 外子命令、旧参数 warning、第四套 profile 注入和
pluginview quickstart 逐条真跑。FREEZE 门禁进入 Smoke/Full/Release 全部 profile。

回滚时恢复旧 flags 与 quickstart 字符串、删除子命令 descriptor/FREEZE 检查并恢复文档即可；
Kit、C style、lifecycle、testengine 的领域状态和 schema 均不迁移。回滚仍不得引入第四个产品
测试 profile。

## §12 Checklist 自评

不适用：本 ADR 扩展既有 command catalog 的登记粒度和兼容层，没有新增 Primitive。实现复用
现有 catalog、handler、report warning 与 testengine FREEZE 入口，不创建第二命令或测试权威。
