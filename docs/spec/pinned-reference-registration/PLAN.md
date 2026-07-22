---
id: pinned-reference-registration
status: draft
approvedTree: ""
scope:
  - config/internal-capabilities.json
  - config/schemas/kit-manifest.schema.json
  - internal/cache/**
  - internal/cli/**
  - internal/kit/**
  - docs/decisions/0010-pinned-reference-registration.md
  - docs/decisions/README.md
  - docs/architecture/06-plugin-sdk.md
  - docs/architecture/KIT_LIFECYCLE_ARCHITECTURE.md
  - docs/architecture/COMPOUNDING_KNOWLEDGE.md
  - docs/architecture/README.md
  - docs/reference/KIT_PLUGIN_VIEW.md
  - docs/CAPABILITIES.md
  - docs/COMMANDS.md
  - README.md
  - CHANGELOG.md
  - docs/todolist/0026-pinned-reference-registration.md
gates:
  - profile: full
---

# 内容钉死引用注册计划

## 目标

在既有 Kit registry、manifest、lifecycle、cache 与 Validation Evidence 身份链上补齐
content-pinned reference：登记只保存 manifest 与不可变 pin，不 vendoring 外部能力正文；
prefetch 承担网络获取，install/update 只从本地内容寻址缓存物化，并在缓存缺失时 fail-closed。

同一计划覆盖方向草稿 `docs/architecture/COMPOUNDING_KNOWLEDGE.md` 的原文落位与架构索引接入；
该草稿保持 `Status: Draft`，不借本轮实现 promotion ledger 或任何开放问题。

## 不变量

- `source` 是 schema v2 的可选字段；旧 manifest 字节语义与生命周期保持兼容。
- Git source 只接受 40-hex commit，content source 只接受 `sha256:` digest；branch/tag/纯路径拒绝。
- pin cache 位于 Git common-dir，内容寻址；registry 正在引用的 pin 不进入 clean 候选。
- import/install/update 路径不调用 network；缺缓存返回 `evidence-missing` 与可执行 required action。
- registry/manifest digest 继续是唯一 Kit catalog identity；不新增 Receipt 或第二 registry。
- 用户定制继续流经 manifest 输入，外部正文与可变用户资产不进入仓库 owned 资产。

## 实施与验证

1. 先以暂存的冻结 schema 真实运行 `plan check --staged`，确认命中 `config/schemas/**`；
   再在 clean tree 上批准本计划并落 ADR 0010。
2. 扩展 manifest/source 校验、prefetch、内容寻址 cache 与本地 materialization；扩展既有
   `kit` 子命令和 typed HelpForm，不新增命令域。
3. 真跑 pinned Git 正例、list/import 延迟与 network=0 断言，以及 branch、坏 SHA、纯路径、
   未预取、旧 manifest、改 pin 旧 Receipt 失效六条负例。
4. 同步 architecture/commands/capability 生成投影，最后运行局部 Go、治理、DocSync、链接、
   `git diff --check` 与 Full profile；仅硬判据全部满足才把 TODO 0026 标为 Done。
