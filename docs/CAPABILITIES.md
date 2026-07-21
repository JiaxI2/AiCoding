# AiCoding 平台能力索引

> 本文件由 `config/internal-capabilities.json` 生成，请运行 `bin/aicoding.exe capability index --write` 更新。

Registry digest: `sha256:9bc4958912fedd4f95a592d6588076cdeab4c39655eecbc97da829fab7e33d14`

共登记 28 个 `internal/` 一级包；文档义务按公共入口、内部实现域和 Primitive 分级。

- `publicEntries` 非空：必须指向 typed command catalog 中的现存入口，并登记架构文档。
- `internal-only`：没有公共入口时可不单建架构文档，避免文档剧场。
- `stable`：必须登记至少一条可执行验证命令；`beta`/`experimental` 仍需明确状态。

| ID | Package | Type | Status | Summary | Public entries | Architecture | Verification |
|---|---|---|---|---|---|---|---|
| `adr-review` | `internal/adrreview` | `internal-only` | `stable` | 检查新 Primitive ADR 是否包含必需的自评清单。 | — | — | `go test ./internal/adrreview/...` |
| `bootstrap` | `internal/bootstrap` | `product-workflow` | `stable` | 检查并构建 AiCoding Go CLI 的最小本地启动路径。 | `aicoding bootstrap` | [文档](architecture/AICODING_CORE_ARCHITECTURE.md) | `go test ./internal/bootstrap/...` |
| `c-style` | `internal/cstyle` | `domain-capability` | `stable` | 统一 C99 风格、注释、格式化与宿主验证入口。 | `aicoding skill c99-standard-c check`<br>`aicoding skill c99-standard-c verify` | [文档](architecture/C_USERSTYLE_KIT_ARCHITECTURE.md) | `go test ./internal/cstyle/...` |
| `cache` | `internal/cache` | `domain-capability` | `stable` | 观测并按证据保护规则回收已注册的本地生成物与临时资源。 | `aicoding cache status`<br>`aicoding cache clean` | [文档](architecture/01-system-architecture.md) | `go test ./internal/cache/...` |
| `capability` | `internal/capability` | `domain-capability` | `beta` | 把 internal 包投影为可查询、可生成且可治理的单一能力目录。 | `aicoding capability list`<br>`aicoding capability describe`<br>`aicoding capability index` | [文档](architecture/01-system-architecture.md) | `go test ./internal/capability/...` |
| `cli` | `internal/cli` | `product-workflow` | `stable` | 拥有 typed command catalog、参数解析、帮助、JSON stdout 与退出码。 | `aicoding --help` | [文档](architecture/AICODING_CORE_ARCHITECTURE.md) | `go test ./internal/cli/...` |
| `docsync` | `internal/docsync` | `domain-capability` | `stable` | 检测源码、配置与权威文档之间的同步漂移。 | `aicoding docsync all` | [文档](architecture/DOC_SYNC_PLUS_SPEC.md) | `go test ./internal/docsync/...` |
| `git` | `internal/gitx` | `primitive` | `stable` | 拥有全仓唯一 Git 子进程边界与内容状态读取。 | — | — | `go test ./internal/gitx/...` |
| `governance` | `internal/governance` | `domain-capability` | `stable` | 执行提交、依赖方向、目录布局与能力孤儿门禁。 | `aicoding governance lint`<br>`aicoding governance dependencies`<br>`aicoding governance layout`<br>`aicoding governance capabilities` | [文档](architecture/GRAPH_FIRST.md) | `go test ./internal/governance/...` |
| `kit` | `internal/kit` | `domain-capability` | `stable` | 加载、投影、验证并脚手架化 Kit 能力。 | `aicoding kit list`<br>`aicoding kit describe`<br>`aicoding kit init`<br>`aicoding kit verify` | [文档](architecture/KIT_LIFECYCLE_ARCHITECTURE.md) | `go test ./internal/kit/...` |
| `lifecycle` | `internal/lifecycle` | `product-workflow` | `stable` | 以统一 adapter catalog 编排 Kit、MCP、runtime Skill 与 repo-context 生命周期。 | `aicoding lifecycle plan`<br>`aicoding lifecycle status`<br>`aicoding lifecycle verify` | [文档](architecture/KIT_LIFECYCLE_ARCHITECTURE.md) | `go test ./internal/lifecycle/...` |
| `loop-engineering` | `internal/loopkit` | `domain-capability` | `stable` | 校验有界 WorkSpec、裁决下一步并追加记录尝试，不执行循环。 | `aicoding work validate`<br>`aicoding work next`<br>`aicoding work status`<br>`aicoding work record` | [文档](architecture/LOOP_ENGINEERING_ARCHITECTURE.md) | `go test ./internal/loopkit/...` |
| `mcp-control` | `internal/mcpcontrol` | `domain-capability` | `stable` | 读取 MCP 注册表并执行状态、诊断、验证与生命周期动作。 | `aicoding mcp list`<br>`aicoding mcp status`<br>`aicoding mcp doctor`<br>`aicoding mcp verify` | [文档](architecture/MCP_CONTROL_PLANE.md) | `go test ./internal/mcpcontrol/...` |
| `plan-mode` | `internal/plan` | `domain-capability` | `stable` | 校验计划产物、批准绑定与 Git Tree 漂移。 | `aicoding plan check`<br>`aicoding plan verify`<br>`aicoding plan status`<br>`aicoding plan approve` | [文档](architecture/PLAN_MODE_ARCHITECTURE.md) | `go test ./internal/plan/...` |
| `platform` | `internal/platform` | `primitive` | `stable` | 提供仓库路径、文件事实与临时资源生命周期原语。 | — | — | `go test ./internal/platform/...` |
| `powershell-regex` | `internal/pwshregex` | `domain-capability` | `stable` | 对 PowerShell 正则高风险写法执行 Go-native 快速检查。 | `aicoding powershell regex-lint` | [文档](architecture/POWERSHELL_BOUNDARY.md) | `go test ./internal/pwshregex/...` |
| `registry` | `internal/registry` | `primitive` | `stable` | 提供确定性 snapshot、digest 与防可变泄漏的注册表原语。 | — | — | `go test ./internal/registry/...` |
| `release-gate` | `internal/releasegate` | `product-workflow` | `stable` | 执行发布结构验证并组合正式 Release 门禁。 | `aicoding release verify`<br>`aicoding release gate` | [文档](governance/RELEASE_POLICY.md) | `go test ./internal/releasegate/...` |
| `repo-context` | `internal/repocontext` | `domain-capability` | `stable` | 构建并同步仓库事实快照，供生命周期只读使用。 | `aicoding lifecycle status` | [文档](architecture/02-context-architecture.md) | `go test ./internal/repocontext/...` |
| `repo-health` | `internal/repohealth` | `product-workflow` | `stable` | 聚合产品 doctor 与确定性 verify 检查。 | `aicoding doctor --all`<br>`aicoding verify` | [文档](architecture/01-system-architecture.md) | `go test ./internal/repohealth/...` |
| `repo-init` | `internal/repoinit` | `product-workflow` | `stable` | 幂等初始化 Git 本地设置、Hook、状态根与文档骨架。 | `aicoding provision` | [文档](decisions/0005-repo-init.md) | `go test ./internal/repoinit/...` |
| `report` | `internal/report` | `primitive` | `stable` | 拥有全仓唯一 Result、StandardReport 与共享证据信封。 | — | — | `go test ./internal/report/...` |
| `reuse-governance` | `internal/reuse` | `domain-capability` | `stable` | 验证可复用模块边界与既有复用证据。 | `aicoding governance reuse` | [文档](architecture/GIT_REUSE_BOUNDARY.md) | `go test ./internal/reuse/...` |
| `runner` | `internal/runner` | `primitive` | `stable` | 按确定性顺序执行有界并发任务，不拥有测试语义。 | — | — | `go test ./internal/runner/...` |
| `tag-policy` | `internal/tagpolicy` | `domain-capability` | `stable` | 只读审计 Git tag 命名空间与发布标签策略。 | `aicoding tag audit` | [文档](governance/TAGGING_POLICY.md) | `go test ./internal/tagpolicy/...` |
| `test-engine` | `internal/testengine` | `product-workflow` | `stable` | 拥有 Smoke、Full、Release 测试注册、执行、超时与报告。 | `aicoding test` | [文档](architecture/AICODING_CORE_ARCHITECTURE.md) | `go test ./internal/testengine/...` |
| `todolist` | `internal/todolist` | `domain-capability` | `stable` | 只读投影 docs/todolist 的状态、标题与验证入口。 | `aicoding todolist` | [文档](decisions/0004-todolist-primitive.md) | `go test ./internal/todolist/...` |
| `validation-evidence` | `internal/validationevidence` | `domain-capability` | `stable` | 把测试结论绑定到 Git 内容身份，同一内容可审计复用。 | `aicoding validation status`<br>`aicoding validation check`<br>`aicoding validation explain`<br>`aicoding validation list` | [文档](decisions/0007-validation-evidence.md) | `go test ./internal/validationevidence/...` |
