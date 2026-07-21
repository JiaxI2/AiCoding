---
id: loop-engineering-backlog
status: draft
scope:
  - "internal/cache/**"
  - "internal/capability/**"
  - "internal/cli/**"
  - "internal/governance/**"
  - "internal/kit/**"
  - "internal/lifecycle/**"
  - "internal/platform/**"
  - "internal/repohealth/**"
  - "internal/repoinit/**"
  - "internal/report/**"
  - "internal/runner/**"
  - "internal/testengine/**"
  - "internal/todolist/**"
  - "internal/validationevidence/**"
  - "config/**"
  - "docs/**"
  - ".github/workflows/**"
  - "README.md"
  - "README_CN.md"
  - "README_EN.md"
  - "CHANGELOG.md"
approvedTree: ""
gates:
  - profile: full
  - profile: release
---

# Loop Engineering 剩余工程清单收敛计划

## 需求

在 `feature/loop-engineering-kit` 上按 owner 提供的阶段 A–G 顺序完成本仓剩余 TODO；
0019 仅整理跨仓外溢清单并保持 Planned。每项必须真实运行自测与负例、翻转状态并独立提交；
全量验证通过后才允许合并 main、生成 Release Receipt、推送和删除 feature 分支。

## 已决策边界

- 接受 TODO 0022 scoped race 的 `45.564s` 实测收益，不为追规划期 `60s` 估算扩张架构。
- 复用现有 command catalog、report.Result、testengine、validationevidence、cache 与 registry；
  不创建第二 runner、schema、注册表或证据权威。
- 永不实现 `loop run`、`work run`、守护进程、远端 attestation、workflow DSL 或自动晋升。
- Full 保留 scoped race + GO-007；Release 与每周 CI 保留全仓 race；真 fresh clone 能力不删除。
- 清理只触碰本仓创建的 `aicoding-*` 临时资源，不回收审计轨迹、不 kill 进程。
- 不在 AiCoding 内放 Skill 源码，不修改只读 `CodingKit/agents/skills` 子模块。

## 实施顺序

1. 合并收尾 0022/0014。
2. 入仓并依次完成 0024、0023、0025。
3. 依次完成 0017、0018。
4. 依次完成 0021、0020、0002。
5. 整理 0019 外溢清单并保持 Planned。
6. 运行全量 Go、治理、DocSync、链接、Smoke/Full/Release 与负例门禁。
7. 仅当前置全部满足时合并 main、重验、正常 push 并收敛分支。

## 回滚

每个 TODO 使用独立本地提交；未发布阶段按提交边界反向恢复对应文件。禁止用
`reset --hard`、`--no-verify` 或强推替代问题定位。合并冲突、门禁失败或前置不满足时停止并报告。
